package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultKeyFunc_PrefersHeaderWhenSet(t *testing.T) {
	fn := DefaultKeyFunc("X-Client", false)

	r := httptest.NewRequest(http.MethodGet, "http://example/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Client", " client-123 ")

	if got := fn(r); got != "client-123" {
		t.Fatalf("expected header key, got %q", got)
	}
}

func TestDefaultKeyFunc_TrustXForwardedForUsesFirstIP(t *testing.T) {
	fn := DefaultKeyFunc("", true)

	r := httptest.NewRequest(http.MethodGet, "http://example/", nil)
	r.RemoteAddr = "10.0.0.9:5555"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")

	if got := fn(r); got != "1.2.3.4" {
		t.Fatalf("expected first XFF ip, got %q", got)
	}
}

func TestDefaultKeyFunc_FallbacksToRemoteAddrHost(t *testing.T) {
	fn := DefaultKeyFunc("", false)

	r := httptest.NewRequest(http.MethodGet, "http://example/", nil)
	r.RemoteAddr = "10.0.0.9:5555"

	if got := fn(r); got != "10.0.0.9" {
		t.Fatalf("expected remote host, got %q", got)
	}
}
