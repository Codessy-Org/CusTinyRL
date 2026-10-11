package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newReq(remote, xff string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = remote
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

func TestClientIPDirectPeer(t *testing.T) {
	r, err := NewIPResolver(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.ClientIP(newReq("203.0.113.7:1234", "")); got != "203.0.113.7" {
		t.Fatalf("got %q", got)
	}
}

func TestClientIPUntrustedIgnoresXFF(t *testing.T) {
	r, err := NewIPResolver([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}
	// Direct peer is not a trusted proxy, so spoofed XFF must be ignored.
	if got := r.ClientIP(newReq("198.51.100.1:9999", "1.2.3.4")); got != "198.51.100.1" {
		t.Fatalf("got %q", got)
	}
}

func TestClientIPTrustedProxyUsesXFF(t *testing.T) {
	r, err := NewIPResolver([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}
	got := r.ClientIP(newReq("10.0.0.5:4444", "203.0.113.9, 10.0.0.3"))
	if got != "203.0.113.9" {
		t.Fatalf("got %q", got)
	}
}

func TestClientIPAllTrustedFallsBack(t *testing.T) {
	r, err := NewIPResolver([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}
	// Every hop is a trusted proxy: fall back to the direct peer.
	if got := r.ClientIP(newReq("10.0.0.5:4444", "10.0.0.3, 10.0.0.4")); got != "10.0.0.5" {
		t.Fatalf("got %q", got)
	}
}

func TestNewIPResolverRejectsGarbage(t *testing.T) {
	if _, err := NewIPResolver([]string{"not-a-cidr"}); err == nil {
		t.Fatal("expected error")
	}
}
