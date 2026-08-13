package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"net/http"
	"strings"
	"unicode"

	"github.com/pquerna/otp/totp"

	"pushrelay/internal/secure"
	"pushrelay/internal/store"
)

func (s *Server) updateUsername(w http.ResponseWriter, r *http.Request) {
	info := r.Context().Value(authKey).(authInfo)
	var in struct {
		Username        string `json:"username"`
		CurrentPassword string `json:"current_password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	username := strings.TrimSpace(in.Username)
	if !validUsername(username) {
		writeError(w, 400, "invalid_username", "username must contain 3 to 64 printable characters", nil, r)
		return
	}
	admin, ok := s.requireCurrentPassword(w, r, info.AdminID, in.CurrentPassword)
	if !ok {
		return
	}
	if username != admin.Username {
		if err := s.store.UpdateUsername(r.Context(), info.AdminID, username); err != nil {
			dbError(w, r, err)
			return
		}
	}
	writeJSON(w, 200, map[string]any{"username": username})
}

func (s *Server) updatePassword(w http.ResponseWriter, r *http.Request) {
	info := r.Context().Value(authKey).(authInfo)
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
		OTP             string `json:"otp"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	admin, ok := s.requireCurrentPassword(w, r, info.AdminID, in.CurrentPassword)
	if !ok {
		return
	}
	if admin.TOTPEnabled {
		valid, err := s.verifySecondFactor(r.Context(), admin, in.OTP, true)
		if err != nil {
			dbError(w, r, err)
			return
		}
		if !valid {
			writeError(w, 401, "invalid_two_factor_code", "invalid two-factor authentication code", nil, r)
			return
		}
	}
	hash, err := secure.HashPassword(in.NewPassword)
	if err != nil {
		writeError(w, 400, "invalid_password", err.Error(), nil, r)
		return
	}
	if secure.VerifyPassword(admin.PasswordHash, in.NewPassword) {
		writeError(w, 400, "password_unchanged", "new password must be different from the current password", nil, r)
		return
	}
	if err = s.store.UpdatePassword(r.Context(), info.AdminID, hash, info.SessionHash); err != nil {
		dbError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) setupTOTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	info := r.Context().Value(authKey).(authInfo)
	var in struct {
		CurrentPassword string `json:"current_password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	admin, ok := s.requireCurrentPassword(w, r, info.AdminID, in.CurrentPassword)
	if !ok {
		return
	}
	if admin.TOTPEnabled {
		writeError(w, 409, "two_factor_already_enabled", "two-factor authentication is already enabled", nil, r)
		return
	}
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "PushRelay", AccountName: admin.Username})
	if err != nil {
		writeError(w, 500, "totp_generation_failed", err.Error(), nil, r)
		return
	}
	secretEnc, err := s.vault.Encrypt([]byte(key.Secret()))
	if err != nil {
		dbError(w, r, err)
		return
	}
	if err = s.store.SetTOTPPending(r.Context(), info.AdminID, secretEnc); err != nil {
		dbError(w, r, err)
		return
	}
	image, err := key.Image(256, 256)
	if err != nil {
		writeError(w, 500, "totp_qr_failed", err.Error(), nil, r)
		return
	}
	var qr bytes.Buffer
	if err = png.Encode(&qr, image); err != nil {
		writeError(w, 500, "totp_qr_failed", err.Error(), nil, r)
		return
	}
	writeJSON(w, 200, map[string]any{
		"secret":           key.Secret(),
		"provisioning_uri": key.URL(),
		"qr_code_data_url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(qr.Bytes()),
	})
}

func (s *Server) enableTOTP(w http.ResponseWriter, r *http.Request) {
	info := r.Context().Value(authKey).(authInfo)
	var in struct {
		Code string `json:"code"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	admin, err := s.store.AdminSecurityByID(r.Context(), info.AdminID)
	if err != nil {
		dbError(w, r, err)
		return
	}
	if len(admin.TOTPSecretEnc) == 0 {
		writeError(w, 409, "two_factor_setup_required", "start two-factor setup first", nil, r)
		return
	}
	valid, err := s.verifySecondFactor(r.Context(), admin, in.Code, false)
	if err != nil {
		dbError(w, r, err)
		return
	}
	if !valid {
		writeError(w, 400, "invalid_two_factor_code", "invalid two-factor authentication code", nil, r)
		return
	}
	codes := make([]string, 8)
	hashes := make([]string, 8)
	for i := range codes {
		codes[i], err = secure.RecoveryCode()
		if err != nil {
			writeError(w, 500, "recovery_code_generation_failed", err.Error(), nil, r)
			return
		}
		hashes[i] = secure.HashToken(secure.NormalizeRecoveryCode(codes[i]))
	}
	if err = s.store.EnableTOTP(r.Context(), info.AdminID, hashes); err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"two_factor_enabled": true, "recovery_codes": codes})
}

func (s *Server) disableTOTP(w http.ResponseWriter, r *http.Request) {
	info := r.Context().Value(authKey).(authInfo)
	var in struct {
		CurrentPassword string `json:"current_password"`
		Code            string `json:"code"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	admin, ok := s.requireCurrentPassword(w, r, info.AdminID, in.CurrentPassword)
	if !ok {
		return
	}
	if !admin.TOTPEnabled {
		writeError(w, 409, "two_factor_not_enabled", "two-factor authentication is not enabled", nil, r)
		return
	}
	valid, err := s.verifySecondFactor(r.Context(), admin, in.Code, true)
	if err != nil {
		dbError(w, r, err)
		return
	}
	if !valid {
		writeError(w, 401, "invalid_two_factor_code", "invalid two-factor authentication code", nil, r)
		return
	}
	if err = s.store.DisableTOTP(r.Context(), info.AdminID); err != nil {
		dbError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) requireCurrentPassword(w http.ResponseWriter, r *http.Request, adminID, password string) (store.AdminSecurity, bool) {
	admin, err := s.store.AdminSecurityByID(r.Context(), adminID)
	if err != nil {
		dbError(w, r, err)
		return store.AdminSecurity{}, false
	}
	if !secure.VerifyPassword(admin.PasswordHash, password) {
		writeError(w, 401, "invalid_current_password", "current password is incorrect", nil, r)
		return store.AdminSecurity{}, false
	}
	return admin, true
}

func (s *Server) verifySecondFactor(ctx context.Context, admin store.AdminSecurity, code string, allowRecovery bool) (bool, error) {
	secret, err := s.vault.Decrypt(admin.TOTPSecretEnc)
	if err != nil {
		return false, err
	}
	normalized := strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if totp.Validate(normalized, string(secret)) {
		return true, nil
	}
	if !allowRecovery {
		return false, nil
	}
	recovery := secure.NormalizeRecoveryCode(code)
	if len(recovery) != 16 {
		return false, nil
	}
	return s.store.ConsumeRecoveryCode(ctx, admin.ID, secure.HashToken(recovery))
}

func validUsername(username string) bool {
	runes := []rune(username)
	if len(runes) < 3 || len(runes) > 64 {
		return false
	}
	for _, value := range runes {
		if unicode.IsControl(value) {
			return false
		}
	}
	return true
}
