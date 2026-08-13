package webui

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// embedded contains the production Vite build. Docker replaces the placeholder
// in dist before compiling the server; local development continues to use Vite.
//
//go:embed all:dist
var embedded embed.FS

var reservedPaths = []string{
	"/api",
	"/hooks",
	"/healthz",
	"/readyz",
	"/openapi.yaml",
}

// Handler returns the embedded UI handler when a production index.html exists.
// A local Go build without a preceding frontend build remains API-only.
func Handler() (http.Handler, bool) {
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		return nil, false
	}
	return newHandler(dist)
}

func newHandler(dist fs.FS) (http.Handler, bool) {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, false
	}
	files := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		if isReserved(r.URL.Path) {
			http.NotFound(w, r)
			return
		}

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name != "" && name != "." && name != "index.html" {
			if info, statErr := fs.Stat(dist, name); statErr == nil && !info.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "public, max-age=3600")
				}
				w.Header().Set("X-Content-Type-Options", "nosniff")
				files.ServeHTTP(w, r)
				return
			}
		}

		if !acceptsHTML(r) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, "index.html", infoModTime(dist, "index.html"), bytes.NewReader(index))
	}), true
}

func isReserved(requestPath string) bool {
	for _, prefix := range reservedPaths {
		if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
			return true
		}
	}
	return false
}

func acceptsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return accept == "" || strings.Contains(accept, "text/html")
}

func infoModTime(dist fs.FS, name string) (modTime time.Time) {
	if info, err := fs.Stat(dist, name); err == nil {
		return info.ModTime()
	}
	return modTime
}
