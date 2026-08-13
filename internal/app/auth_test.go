package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"pushrelay/internal/config"
	"pushrelay/internal/secure"
	"pushrelay/internal/store"
)

func TestAccountAndTwoFactorFlow(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.DB.Close()
	password := "correct horse battery staple"
	hash, err := secure.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = st.CreateAdmin(t.Context(), "admin", hash); err != nil {
		t.Fatal(err)
	}
	vault, err := secure.NewVault(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	server, err := New(config.Config{WebOrigin: "http://example.test", SessionTTL: time.Hour}, st, vault, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	login := requestJSON(t, server.Router(), http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "admin", "password": password}, nil, "")
	if login.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", login.Code, login.Body.String())
	}
	var loginBody struct {
		CSRF string `json:"csrf_token"`
	}
	decodeResponse(t, login, &loginBody)
	session := login.Result().Cookies()[0]

	updated := requestJSON(t, server.Router(), http.MethodPut, "/api/v1/account/username", map[string]any{"username": "relay-admin", "current_password": password}, session, loginBody.CSRF)
	if updated.Code != http.StatusOK {
		t.Fatalf("username update failed: %d %s", updated.Code, updated.Body.String())
	}

	setup := requestJSON(t, server.Router(), http.MethodPost, "/api/v1/account/2fa/setup", map[string]any{"current_password": password}, session, loginBody.CSRF)
	if setup.Code != http.StatusOK {
		t.Fatalf("2FA setup failed: %d %s", setup.Code, setup.Body.String())
	}
	var setupBody struct {
		Secret string `json:"secret"`
	}
	decodeResponse(t, setup, &setupBody)
	code, err := totp.GenerateCode(setupBody.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	enabled := requestJSON(t, server.Router(), http.MethodPost, "/api/v1/account/2fa/enable", map[string]any{"code": code}, session, loginBody.CSRF)
	if enabled.Code != http.StatusOK {
		t.Fatalf("2FA enable failed: %d %s", enabled.Code, enabled.Body.String())
	}
	var enabledBody struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	decodeResponse(t, enabled, &enabledBody)
	if len(enabledBody.RecoveryCodes) != 8 {
		t.Fatalf("got %d recovery codes", len(enabledBody.RecoveryCodes))
	}

	withoutOTP := requestJSON(t, server.Router(), http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "relay-admin", "password": password}, nil, "")
	if withoutOTP.Code != http.StatusAccepted {
		t.Fatalf("expected second factor challenge, got %d %s", withoutOTP.Code, withoutOTP.Body.String())
	}
	withRecovery := requestJSON(t, server.Router(), http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "relay-admin", "password": password, "otp": enabledBody.RecoveryCodes[0]}, nil, "")
	if withRecovery.Code != http.StatusOK {
		t.Fatalf("recovery login failed: %d %s", withRecovery.Code, withRecovery.Body.String())
	}
	reusedRecovery := requestJSON(t, server.Router(), http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "relay-admin", "password": password, "otp": enabledBody.RecoveryCodes[0]}, nil, "")
	if reusedRecovery.Code != http.StatusUnauthorized {
		t.Fatalf("recovery code was reusable: %d %s", reusedRecovery.Code, reusedRecovery.Body.String())
	}
}

func TestPocketIDIssuerNormalization(t *testing.T) {
	got := normalizePocketIDIssuer(" https://id.example.com/.well-known/openid-configuration/ ")
	if got != "https://id.example.com" {
		t.Fatalf("got %q", got)
	}
	if err := validatePocketIDIssuer(got); err != nil {
		t.Fatal(err)
	}
	if err := validatePocketIDIssuer("http://id.example.com"); err == nil {
		t.Fatal("expected insecure non-loopback issuer to be rejected")
	}
	if err := validatePocketIDIssuer("http://127.0.0.1:1411"); err != nil {
		t.Fatal(err)
	}
}

func requestJSON(t *testing.T, handler http.Handler, method, path string, body any, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.test")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatal(err)
	}
}
