package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestConcurrencyMiddleware_TimesOutWhenNoSlot(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	secondDone := make(chan struct{})
	var startedOnce sync.Once

	// handler que segura a vaga até liberarmos.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedOnce.Do(func() { close(started) })
		<-release
		w.WriteHeader(http.StatusOK)
	})

	h := ConcurrencyMiddleware(ConcurrencyOptions{
		Max:            1,
		RejectStatus:   http.StatusServiceUnavailable,
		AcquireTimeout: 25 * time.Millisecond,
	})(next)

	var wg sync.WaitGroup
	wg.Add(2)

	// request 1: ocupa o semáforo e fica pendurado
	go func() {
		defer wg.Done()
		r1 := httptest.NewRequest(http.MethodGet, "http://example/", nil)
		w1 := httptest.NewRecorder()
		h.ServeHTTP(w1, r1)
		if w1.Code != http.StatusOK {
			t.Errorf("expected first request 200, got %d", w1.Code)
		}
	}()

	// espera a primeira realmente entrar no handler
	select {
	case <-started:
	case <-time.After(200 * time.Millisecond):
		close(release)
		wg.Wait()
		t.Fatalf("timeout waiting first request to start")
	}

	// request 2: deve falhar por timeout ao tentar adquirir
	go func() {
		defer wg.Done()
		r2 := httptest.NewRequest(http.MethodGet, "http://example/", nil)
		w2 := httptest.NewRecorder()
		h.ServeHTTP(w2, r2)
		if w2.Code != http.StatusServiceUnavailable {
			t.Errorf("expected second request 503, got %d", w2.Code)
		}
		close(secondDone)
	}()

	// garante que a segunda terminou antes de liberar a primeira (senão a 2ª pode adquirir)
	select {
	case <-secondDone:
	case <-time.After(500 * time.Millisecond):
		close(release)
		wg.Wait()
		t.Fatalf("timeout waiting second request to finish")
	}

	// libera a primeira
	close(release)
	wg.Wait()
}
