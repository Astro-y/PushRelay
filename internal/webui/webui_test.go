package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHandlerServesAssetsAndSPAFallback(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":        &fstest.MapFile{Data: []byte("<main>PushRelay</main>")},
		"assets/app-123.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
		"favicon.svg":       &fstest.MapFile{Data: []byte("<svg></svg>")},
	}
	handler, ok := newHandler(dist)
	if !ok {
		t.Fatal("expected UI handler")
	}

	tests := []struct {
		name         string
		path         string
		accept       string
		wantStatus   int
		wantCache    string
		wantContains string
	}{
		{name: "root", path: "/", accept: "text/html", wantStatus: 200, wantCache: "no-cache", wantContains: "PushRelay"},
		{name: "index", path: "/index.html", accept: "text/html", wantStatus: 200, wantCache: "no-cache", wantContains: "PushRelay"},
		{name: "spa route", path: "/settings", accept: "text/html", wantStatus: 200, wantCache: "no-cache", wantContains: "PushRelay"},
		{name: "hashed asset", path: "/assets/app-123.js", accept: "*/*", wantStatus: 200, wantCache: "public, max-age=31536000, immutable", wantContains: "console.log"},
		{name: "ordinary static file", path: "/favicon.svg", accept: "image/svg+xml", wantStatus: 200, wantCache: "public, max-age=3600", wantContains: "svg"},
		{name: "unknown API", path: "/api/v1/missing", accept: "text/html", wantStatus: 404},
		{name: "unknown hook", path: "/hooks/missing/extra", accept: "text/html", wantStatus: 404},
		{name: "non document", path: "/missing.json", accept: "application/json", wantStatus: 404},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			req.Header.Set("Accept", test.accept)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if cache := response.Header().Get("Cache-Control"); cache != test.wantCache {
				t.Fatalf("cache = %q, want %q", cache, test.wantCache)
			}
			if test.wantContains != "" && !strings.Contains(response.Body.String(), test.wantContains) {
				t.Fatalf("body %q does not contain %q", response.Body.String(), test.wantContains)
			}
		})
	}
}

func TestHandlerUnavailableWithoutIndex(t *testing.T) {
	if _, ok := newHandler(fstest.MapFS{".keep": &fstest.MapFile{}}); ok {
		t.Fatal("handler should be unavailable without index.html")
	}
}

func TestReservedPaths(t *testing.T) {
	for _, requestPath := range []string{"/api", "/api/v1", "/hooks/token", "/healthz", "/readyz", "/openapi.yaml"} {
		if !isReserved(requestPath) {
			t.Fatalf("%q should be reserved", requestPath)
		}
	}
	if isReserved("/apiary") {
		t.Fatal("unrelated path should not be reserved")
	}
}
