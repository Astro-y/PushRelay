package app

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"pushrelay/internal/channels"
	"pushrelay/internal/config"
	"pushrelay/internal/secure"
	"pushrelay/internal/store"
	"pushrelay/internal/webui"
)

type Server struct {
	cfg        config.Config
	store      *store.Store
	vault      *secure.Vault
	channels   *channels.Service
	logger     *slog.Logger
	router     chi.Router
	setupToken string
	settings   atomic.Value
}
type authInfo struct{ AdminID, Username, CSRF, SessionHash string }
type contextKey string

const authKey contextKey = "auth"

func New(cfg config.Config, st *store.Store, vault *secure.Vault, logger *slog.Logger) (*Server, error) {
	runtimeSettings, err := st.RuntimeSettings(context.Background())
	if err != nil {
		return nil, err
	}
	token := cfg.SetupToken
	if token == "" {
		var err error
		token, err = secure.RandomToken(24)
		if err != nil {
			return nil, err
		}
	}
	s := &Server{cfg: cfg, store: st, vault: vault, channels: channels.New(runtimeSettings.AllowPrivateWebhookTargets), logger: logger, setupToken: token}
	s.settings.Store(runtimeSettings)
	has, err := st.HasAdmin(context.Background())
	if err != nil {
		return nil, err
	}
	if !has {
		logger.Warn("administrator setup is required", "setup_token", token)
	}
	s.router = s.routes()
	return s, nil
}
func (s *Server) Router() http.Handler { return s.router }

func (s *Server) currentSettings() store.RuntimeSettings {
	return s.settings.Load().(store.RuntimeSettings)
}

func (s *Server) routes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, s.accessLog, s.cors)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"status": "ok"}) })
	r.Get("/readyz", s.ready)
	r.Get("/openapi.yaml", s.openapi)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/setup/status", s.setupStatus)
		r.Post("/setup", s.setup)
		r.Post("/auth/login", s.login)
		r.Get("/auth/pocketid/status", s.pocketIDStatus)
		r.Get("/auth/pocketid/start", s.pocketIDStart)
		r.Get("/auth/pocketid/callback", s.pocketIDCallback)
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Get("/auth/me", s.me)
			r.Post("/auth/logout", s.requireCSRF(s.logout))
			r.Get("/dashboard", s.dashboard)
			r.Get("/channel-types", s.channelTypes)
			r.Group(func(r chi.Router) {
				r.Use(func(next http.Handler) http.Handler { return s.requireCSRF(next.ServeHTTP) })
				s.resourceRoutes(r)
			})
		})
	})
	r.HandleFunc("/hooks/{token}", s.hook)
	if ui, ok := webui.Handler(); ok {
		r.NotFound(ui.ServeHTTP)
	}
	return r
}

func (s *Server) resourceRoutes(r chi.Router) {
	r.Get("/channels", s.listChannels)
	r.Post("/channels", s.saveChannel)
	r.Put("/channels/{id}", s.saveChannel)
	r.Delete("/channels/{id}", s.deleteChannel)
	r.Post("/channels/{id}/test", s.testChannel)
	r.Get("/templates", s.listTemplates)
	r.Post("/templates", s.saveTemplate)
	r.Put("/templates/{id}", s.saveTemplate)
	r.Delete("/templates/{id}", s.deleteTemplate)
	r.Post("/preview", s.preview)
	r.Get("/target-groups", s.listGroups)
	r.Post("/target-groups", s.saveGroup)
	r.Put("/target-groups/{id}", s.saveGroup)
	r.Delete("/target-groups/{id}", s.deleteGroup)
	r.Get("/sources", s.listSources)
	r.Post("/sources", s.saveSource)
	r.Put("/sources/{id}", s.saveSource)
	r.Delete("/sources/{id}", s.deleteSource)
	r.Post("/sources/{id}/rotate", s.rotateSource)
	r.Get("/rules", s.listRules)
	r.Post("/rules", s.saveRule)
	r.Put("/rules/{id}", s.saveRule)
	r.Delete("/rules/{id}", s.deleteRule)
	r.Get("/schedules", s.listSchedules)
	r.Post("/schedules", s.saveSchedule)
	r.Put("/schedules/{id}", s.saveSchedule)
	r.Delete("/schedules/{id}", s.deleteSchedule)
	r.Post("/schedules/preview", s.previewSchedule)
	r.Get("/events", s.listEvents)
	r.Get("/deliveries", s.listDeliveries)
	r.Post("/deliveries/{id}/retry", s.retryDelivery)
	r.Get("/settings", s.getSettings)
	r.Put("/settings", s.saveSettings)
	r.Put("/account/username", s.updateUsername)
	r.Put("/account/password", s.updatePassword)
	r.Post("/account/2fa/setup", s.setupTOTP)
	r.Post("/account/2fa/enable", s.enableTOTP)
	r.Post("/account/2fa/disable", s.disableTOTP)
}

func (s *Server) Start(ctx context.Context) {
	for i := 0; i < s.cfg.WorkerConcurrency; i++ {
		go s.worker(ctx, i)
	}
	go s.scheduler(ctx)
	go s.maintenance(ctx)
}
func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		s.logger.Info("http request", "request_id", middleware.GetReqID(r.Context()), "method", r.Method, "path", r.URL.Path, "status", ww.Status(), "duration_ms", time.Since(start).Milliseconds())
	})
}
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && origin == s.cfg.WebOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, X-Setup-Token, Idempotency-Key, X-Push-Timestamp, X-Push-Signature")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	if err := s.store.DB.PingContext(ctx); err != nil {
		writeError(w, 503, "not_ready", err.Error(), nil, r)
		return
	}
	writeJSON(w, 200, map[string]any{"status": "ready"})
}
func (s *Server) openapi(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "openapi.yaml")
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("pushrelay_session")
		if err != nil {
			writeError(w, 401, "unauthorized", "authentication required", nil, r)
			return
		}
		hash := secure.HashToken(cookie.Value)
		adminID, username, csrf, _, err := s.store.Session(r.Context(), hash)
		if err != nil {
			writeError(w, 401, "unauthorized", "invalid or expired session", nil, r)
			return
		}
		info := authInfo{adminID, username, csrf, hash}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authKey, info)))
	})
}
func (s *Server) requireCSRF(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			next(w, r)
			return
		}
		info, _ := r.Context().Value(authKey).(authInfo)
		provided := r.Header.Get("X-CSRF-Token")
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(info.CSRF)) != 1 {
			writeError(w, 403, "csrf_failed", "missing or invalid CSRF token", nil, r)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != s.cfg.WebOrigin {
			writeError(w, 403, "origin_rejected", "request origin is not allowed", nil, r)
			return
		}
		next(w, r)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, 400, "invalid_json", err.Error(), nil, r)
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code, message string, details any, r *http.Request) {
	writeJSON(w, status, map[string]any{"code": code, "message": message, "details": details, "request_id": middleware.GetReqID(r.Context())})
}
func pathID(r *http.Request) string { return chi.URLParam(r, "id") }
func requireName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("name is required")
	}
	if len([]rune(name)) > 100 {
		return errors.New("name is too long")
	}
	return nil
}
func dbError(w http.ResponseWriter, r *http.Request, err error) {
	status := 500
	code := "internal_error"
	if store.IsNotFound(err) {
		status = 404
		code = "not_found"
	}
	if strings.Contains(err.Error(), "FOREIGN KEY") || strings.Contains(err.Error(), "UNIQUE constraint") {
		status = 409
		code = "conflict"
	}
	writeError(w, status, code, err.Error(), nil, r)
}
func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func isSQLNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }

var _ = fmt.Sprintf
