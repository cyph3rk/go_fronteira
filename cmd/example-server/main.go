package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"middleware-gateway/middleware/ratelimit"
	"middleware-gateway/middleware/ratelimit/infra"
)

func main() {
	// Exemplo: injetando o middleware diretamente no seu webserver (sem proxy)
	store := infra.NewStore(5, 10)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	store.StartJanitor(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	h := http.Handler(mux)
	h = ratelimit.ConcurrencyMiddleware(ratelimit.ConcurrencyOptions{Max: 50})(h)
	h = ratelimit.Middleware(ratelimit.Options{
		Store:               store,
		KeyHeader:           "X-Api-Key", // ou vazio para usar IP
		TrustXForwardedFor:  true,
		AddRateLimitHeaders: true,
	})(h)

	addr := ":8081"
	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		addr = v
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("example server listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
