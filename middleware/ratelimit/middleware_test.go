package ratelimit

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"middleware-gateway/middleware/ratelimit/infra"
)

func TestMiddleware_AllowsThenRejectsSameKey(t *testing.T) {
	store := infra.NewStore(0.02, 1)

	calls := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})

	h := Middleware(Options{
		Store:               store,
		RejectStatus:        http.StatusTooManyRequests,
		RetryAfter:          1 * time.Second,
		AddRateLimitHeaders: true,
	})(next)

	// 1) primeira passa
	r1 := httptest.NewRequest(http.MethodGet, "http://example/showTela", nil)
	r1.RemoteAddr = "10.0.0.1:1234"
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}
	if got := w1.Header().Get("X-RateLimit-Key"); got == "" {
		t.Fatalf("expected X-RateLimit-Key header to be set")
	}
	if got := w1.Header().Get("X-RateLimit-RPS"); got == "" {
		t.Fatalf("expected X-RateLimit-RPS header to be set")
	}
	if got := w1.Header().Get("X-RateLimit-Burst"); got == "" {
		t.Fatalf("expected X-RateLimit-Burst header to be set")
	}

	// 2) segunda deve bloquear (burst=1 e rps bem baixo)
	r2 := httptest.NewRequest(http.MethodGet, "http://example/showTela", nil)
	r2.RemoteAddr = "10.0.0.1:1234"
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w2.Code)
	}
	if got := w2.Header().Get("Retry-After"); got == "" {
		t.Fatalf("expected Retry-After header to be set")
	}

	if calls != 1 {
		t.Fatalf("expected next handler to be called once, got %d", calls)
	}
}

func TestMiddleware_KeyByHeader(t *testing.T) {
	store := infra.NewStore(0.02, 1)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := Middleware(Options{
		Store:      store,
		KeyHeader:  "X-Api-Key",
		RetryAfter: 1 * time.Second,
	})(next)

	// duas chaves diferentes => ambos devem passar (cada chave tem seu próprio limiter)
	r1 := httptest.NewRequest(http.MethodGet, "http://example/", nil)
	r1.Header.Set("X-Api-Key", "k1")
	r1.RemoteAddr = "10.0.0.1:1234"
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200 for key k1, got %d", w1.Code)
	}

	r2 := httptest.NewRequest(http.MethodGet, "http://example/", nil)
	r2.Header.Set("X-Api-Key", "k2")
	r2.RemoteAddr = "10.0.0.1:1234"
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for key k2, got %d", w2.Code)
	}
}

func TestMiddleware_RetryAfterUsesSeconds(t *testing.T) {
	store := infra.NewStore(0.02, 1)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := Middleware(Options{
		Store:      store,
		RetryAfter: 2500 * time.Millisecond,
	})(next)

	r1 := httptest.NewRequest(http.MethodGet, "http://example/", nil)
	r1.RemoteAddr = "10.0.0.1:1234"
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}

	r2 := httptest.NewRequest(http.MethodGet, "http://example/", nil)
	r2.RemoteAddr = "10.0.0.1:1234"
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w2.Code)
	}
	if got := strings.TrimSpace(w2.Header().Get("Retry-After")); got != "2" {
		// int(2.5s.Seconds()) == 2
		t.Fatalf("expected Retry-After=2, got %q", got)
	}
}
