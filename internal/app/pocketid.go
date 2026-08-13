package app

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"pushrelay/internal/secure"
	"pushrelay/internal/store"
)

type pocketIDClient struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth    oauth2.Config
}

func (s *Server) pocketIDStatus(w http.ResponseWriter, r *http.Request) {
	settings := s.currentSettings()
	writeJSON(w, 200, map[string]any{"enabled": pocketIDConfigured(settings)})
}

func (s *Server) pocketIDStart(w http.ResponseWriter, r *http.Request) {
	client, settings, err := s.newPocketIDClient(r.Context(), requestBaseURL(r)+"/api/v1/auth/pocketid/callback")
	if err != nil {
		writeError(w, 503, "pocketid_unavailable", err.Error(), nil, r)
		return
	}
	if !settings.PocketIDEnabled {
		writeError(w, 404, "pocketid_disabled", "Pocket ID login is disabled", nil, r)
		return
	}
	state, err := secure.RandomToken(32)
	if err != nil {
		writeError(w, 500, "oauth_state_failed", err.Error(), nil, r)
		return
	}
	nonce, err := secure.RandomToken(24)
	if err != nil {
		writeError(w, 500, "oauth_nonce_failed", err.Error(), nil, r)
		return
	}
	verifier := oauth2.GenerateVerifier()
	if err = s.store.CreateOAuthState(r.Context(), secure.HashToken(state), nonce, verifier, time.Now().Add(10*time.Minute)); err != nil {
		dbError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "pushrelay_oauth_state",
		Value:    state,
		Path:     "/api/v1/auth/pocketid/callback",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsHTTPS(r),
		MaxAge:   600,
	})
	authURL := client.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Server) pocketIDCallback(w http.ResponseWriter, r *http.Request) {
	if providerError := strings.TrimSpace(r.URL.Query().Get("error")); providerError != "" {
		s.redirectOAuthError(w, r, "Pocket ID authorization was denied")
		return
	}
	stateToken := strings.TrimSpace(r.URL.Query().Get("state"))
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if stateToken == "" || code == "" {
		s.redirectOAuthError(w, r, "Pocket ID callback is missing required parameters")
		return
	}
	stateCookie, err := r.Cookie("pushrelay_oauth_state")
	if err != nil || !subtleEqual(stateCookie.Value, stateToken) {
		s.redirectOAuthError(w, r, "Pocket ID login was not started by this browser")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "pushrelay_oauth_state",
		Value:    "",
		Path:     "/api/v1/auth/pocketid/callback",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsHTTPS(r),
		MaxAge:   -1,
	})
	state, err := s.store.ConsumeOAuthState(r.Context(), secure.HashToken(stateToken))
	if err != nil {
		s.redirectOAuthError(w, r, "Pocket ID login state is invalid or expired")
		return
	}
	client, settings, err := s.newPocketIDClient(r.Context(), requestBaseURL(r)+"/api/v1/auth/pocketid/callback")
	if err != nil || !settings.PocketIDEnabled {
		s.redirectOAuthError(w, r, "Pocket ID login is unavailable")
		return
	}
	token, err := client.oauth.Exchange(r.Context(), code, oauth2.VerifierOption(state.PKCEVerifier))
	if err != nil {
		s.redirectOAuthError(w, r, "Pocket ID authorization code exchange failed")
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		s.redirectOAuthError(w, r, "Pocket ID did not return an ID token")
		return
	}
	idToken, err := client.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		s.redirectOAuthError(w, r, "Pocket ID token verification failed")
		return
	}
	if idToken.Nonce != state.Nonce {
		s.redirectOAuthError(w, r, "Pocket ID nonce verification failed")
		return
	}
	var claims struct {
		Subject           string `json:"sub"`
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err = idToken.Claims(&claims); err != nil || claims.Subject == "" {
		s.redirectOAuthError(w, r, "Pocket ID identity claims are invalid")
		return
	}
	admin, err := s.store.SoleAdminSecurity(r.Context())
	if err != nil {
		s.redirectOAuthError(w, r, "administrator account is unavailable")
		return
	}
	if admin.PocketIDSubject == "" {
		allowed := strings.TrimSpace(settings.PocketIDAllowedIdentity)
		identityMatched := strings.EqualFold(allowed, claims.PreferredUsername) || (claims.EmailVerified && strings.EqualFold(allowed, claims.Email))
		if allowed == "" || !identityMatched {
			s.redirectOAuthError(w, r, "Pocket ID identity is not allowed to bind this administrator")
			return
		}
		if err = s.store.BindPocketIDSubject(r.Context(), admin.ID, claims.Subject); err != nil {
			s.redirectOAuthError(w, r, "Pocket ID identity could not be bound")
			return
		}
	} else if admin.PocketIDSubject != claims.Subject {
		s.redirectOAuthError(w, r, "Pocket ID identity is not linked to this administrator")
		return
	}
	if _, _, err = s.issueSession(w, r, admin.ID); err != nil {
		s.redirectOAuthError(w, r, "session creation failed")
		return
	}
	http.Redirect(w, r, strings.TrimRight(s.cfg.WebOrigin, "/")+"/?oauth=success", http.StatusFound)
}

func (s *Server) newPocketIDClient(ctx context.Context, redirectURL string) (*pocketIDClient, store.RuntimeSettings, error) {
	settings := s.currentSettings()
	if !pocketIDConfigured(settings) {
		return nil, settings, errors.New("Pocket ID is not fully configured")
	}
	secretCipher, err := base64.RawStdEncoding.DecodeString(settings.PocketIDClientSecretEnc)
	if err != nil {
		return nil, settings, errors.New("Pocket ID client secret is invalid")
	}
	secret, err := s.vault.Decrypt(secretCipher)
	if err != nil {
		return nil, settings, errors.New("Pocket ID client secret could not be decrypted")
	}
	provider, err := oidc.NewProvider(ctx, settings.PocketIDIssuerURL)
	if err != nil {
		return nil, settings, err
	}
	config := oauth2.Config{
		ClientID:     settings.PocketIDClientID,
		ClientSecret: string(secret),
		Endpoint:     provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	return &pocketIDClient{provider: provider, verifier: provider.Verifier(&oidc.Config{ClientID: settings.PocketIDClientID}), oauth: config}, settings, nil
}

func pocketIDConfigured(settings store.RuntimeSettings) bool {
	return settings.PocketIDEnabled && settings.PocketIDIssuerURL != "" && settings.PocketIDClientID != "" && settings.PocketIDClientSecretSet && settings.PocketIDAllowedIdentity != ""
}

func validatePocketIDIssuer(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("Pocket ID issuer URL is invalid")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	host := strings.ToLower(parsed.Hostname())
	if parsed.Scheme == "http" && (host == "localhost" || host == "127.0.0.1" || host == "::1") {
		return nil
	}
	return errors.New("Pocket ID issuer URL must use HTTPS except on loopback hosts")
}

func normalizePocketIDIssuer(raw string) string {
	value := strings.TrimRight(strings.TrimSpace(raw), "/")
	return strings.TrimSuffix(value, "/.well-known/openid-configuration")
}

func (s *Server) redirectOAuthError(w http.ResponseWriter, r *http.Request, message string) {
	target := strings.TrimRight(s.cfg.WebOrigin, "/") + "/login?oauth_error=" + url.QueryEscape(message)
	http.Redirect(w, r, target, http.StatusFound)
}
