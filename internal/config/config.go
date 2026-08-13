package config

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr          string
	DatabasePath      string
	EncryptionKey     []byte
	WebOrigin         string
	TrustedProxyCIDRs []string
	WorkerConcurrency int
	SetupToken        string
	SessionTTL        time.Duration
	MaxBodyBytes      int64
	Runtime           string
}

func Load() (Config, error) {
	c := Config{
		HTTPAddr:          env("HTTP_ADDR", ":4426"),
		DatabasePath:      env("DATABASE_PATH", "./data/pushrelay.db"),
		WebOrigin:         strings.TrimRight(env("WEB_ORIGIN", "http://localhost:5173"), "/"),
		WorkerConcurrency: envInt("WORKER_CONCURRENCY", 8),
		SetupToken:        strings.TrimSpace(os.Getenv("SETUP_TOKEN")),
		SessionTTL:        7 * 24 * time.Hour,
		MaxBodyBytes:      1 << 20,
		Runtime:           runtime.GOOS + "/" + runtime.GOARCH,
	}
	if raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXY_CIDRS")); raw != "" {
		for _, item := range strings.Split(raw, ",") {
			if v := strings.TrimSpace(item); v != "" {
				c.TrustedProxyCIDRs = append(c.TrustedProxyCIDRs, v)
			}
		}
	}
	key, err := parseKey(strings.TrimSpace(os.Getenv("APP_ENCRYPTION_KEY")))
	if err != nil {
		return Config{}, err
	}
	c.EncryptionKey = key
	if c.WorkerConcurrency < 1 || c.WorkerConcurrency > 64 {
		return Config{}, errors.New("WORKER_CONCURRENCY must be between 1 and 64")
	}
	return c, nil
}

func parseKey(raw string) ([]byte, error) {
	if raw == "" {
		return nil, errors.New("APP_ENCRYPTION_KEY is required (32 random bytes in base64 or 64 hexadecimal characters)")
	}
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := hex.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	return nil, errors.New("APP_ENCRYPTION_KEY must decode to exactly 32 bytes")
}

func env(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

func envInt(name string, fallback int) int {
	v, err := strconv.Atoi(env(name, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return v
}
