package app

import (
	"net/http"
	"strings"
	"time"

	"pushrelay/internal/secure"
)

func (s *Server) setupStatus(w http.ResponseWriter, r *http.Request) {
	has, err := s.store.HasAdmin(r.Context())
	if err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"required": !has})
}
func (s *Server) setup(w http.ResponseWriter, r *http.Request) {
	if subtleEqual(r.Header.Get("X-Setup-Token"), s.setupToken) == false {
		writeError(w, 403, "invalid_setup_token", "invalid setup token", nil, r)
		return
	}
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if len(in.Username) < 3 {
		writeError(w, 400, "invalid_username", "username must contain at least 3 characters", nil, r)
		return
	}
	hash, err := secure.HashPassword(in.Password)
	if err != nil {
		writeError(w, 400, "invalid_password", err.Error(), nil, r)
		return
	}
	if _, err = s.store.CreateAdmin(r.Context(), in.Username, hash); err != nil {
		dbError(w, r, err)
		return
	}
	s.setupToken = ""
	writeJSON(w, 201, map[string]any{"status": "created"})
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
		OTP      string `json:"otp"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	admin, err := s.store.AdminSecurityByUsername(r.Context(), strings.TrimSpace(in.Username))
	if err != nil || !secure.VerifyPassword(admin.PasswordHash, in.Password) {
		time.Sleep(150 * time.Millisecond)
		writeError(w, 401, "invalid_credentials", "invalid username or password", nil, r)
		return
	}
	if admin.TOTPEnabled && strings.TrimSpace(in.OTP) == "" {
		writeJSON(w, http.StatusAccepted, map[string]any{"two_factor_required": true})
		return
	}
	if admin.TOTPEnabled {
		valid, verifyErr := s.verifySecondFactor(r.Context(), admin, in.OTP, true)
		if verifyErr != nil {
			dbError(w, r, verifyErr)
			return
		}
		if !valid {
			time.Sleep(150 * time.Millisecond)
			writeError(w, 401, "invalid_two_factor_code", "invalid two-factor authentication code", nil, r)
			return
		}
	}
	csrf, expires, err := s.issueSession(w, r, admin.ID)
	if err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"username": admin.Username, "csrf_token": csrf, "expires_at": expires.Unix(), "two_factor_enabled": admin.TOTPEnabled})
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, adminID string) (string, time.Time, error) {
	token, _ := secure.RandomToken(32)
	csrf, _ := secure.RandomToken(24)
	expires := time.Now().Add(s.cfg.SessionTTL)
	if err := s.store.CreateSession(r.Context(), secure.HashToken(token), adminID, csrf, expires); err != nil {
		return "", time.Time{}, err
	}
	http.SetCookie(w, &http.Cookie{Name: "pushrelay_session", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: requestIsHTTPS(r), Expires: expires, MaxAge: int(s.cfg.SessionTTL.Seconds())})
	return csrf, expires, nil
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	info := r.Context().Value(authKey).(authInfo)
	admin, err := s.store.AdminSecurityByID(r.Context(), info.AdminID)
	if err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"id": info.AdminID, "username": info.Username, "csrf_token": info.CSRF, "two_factor_enabled": admin.TOTPEnabled, "pocketid_linked": admin.PocketIDSubject != ""})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	info := r.Context().Value(authKey).(authInfo)
	_ = s.store.DeleteSession(r.Context(), info.SessionHash)
	http.SetCookie(w, &http.Cookie{Name: "pushrelay_session", Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]), "https")
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if requestIsHTTPS(r) {
		scheme = "https"
	}
	host := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host
}

func subtleEqual(a, b string) bool {
	if len(a) != len(b) || a == "" {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
