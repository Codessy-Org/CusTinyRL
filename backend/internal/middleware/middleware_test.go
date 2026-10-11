package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gauravkrrr/url-shortner/backend/internal/httpx"
)

func TestCORSAllowlist(t *testing.T) {
	h := CORS([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Allowed origin.
	r := httptest.NewRequest(http.MethodGet, "/api/v1/urls", nil)
	r.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatal("missing allow origin for allowlisted origin")
	}

	// Disallowed origin: no CORS headers.
	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/urls", nil)
	r2.Header.Set("Origin", "https://evil.example")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("must not grant CORS to non-allowlisted origin")
	}
}

func TestCORSPreflight(t *testing.T) {
	h := CORS([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/urls", nil)
	r.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d", w.Code)
	}
}

func TestRequestID(t *testing.T) {
	var seen string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = GetRequestID(r.Context())
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if seen == "" {
		t.Fatal("expected request id in context")
	}
	if w.Header().Get("X-Request-Id") != seen {
		t.Fatal("header and context ids differ")
	}
}

func TestClientIPMiddleware(t *testing.T) {
	resolver, _ := httpx.NewIPResolver(nil)
	var seen string
	h := ClientIP(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = GetClientIP(r.Context())
	}))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.1:80"
	h.ServeHTTP(httptest.NewRecorder(), r)
	if seen != "192.0.2.1" {
		t.Fatalf("got %q", seen)
	}
}
