package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pushrelay/internal/app"
	"pushrelay/internal/config"
	"pushrelay/internal/secure"
	"pushrelay/internal/store"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		if err := runHealthcheck(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}
	st, err := store.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("database error", "error", err)
		os.Exit(1)
	}
	defer st.DB.Close()
	vault, err := secure.NewVault(cfg.EncryptionKey)
	if err != nil {
		logger.Error("encryption error", "error", err)
		os.Exit(1)
	}
	application, err := app.New(cfg, st, vault, logger)
	if err != nil {
		logger.Error("application error", "error", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	application.Start(ctx)
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: application.Router(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		_ = httpServer.Shutdown(shutdownCtx)
	}()
	logger.Info("pushrelay listening", "address", cfg.HTTPAddr, "runtime", cfg.Runtime)
	if err = httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func runHealthcheck() error {
	healthURL := os.Getenv("PUSHRELAY_HEALTHCHECK_URL")
	if healthURL == "" {
		healthURL = "http://127.0.0.1:4426/healthz"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(healthURL)
	if err != nil {
		return fmt.Errorf("healthcheck request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("healthcheck returned %s", response.Status)
	}
	return nil
}
