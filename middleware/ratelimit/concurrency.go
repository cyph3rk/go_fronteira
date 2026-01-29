package ratelimit

import (
	"net/http"
	"time"

	"middleware-gateway/middleware/ratelimit/application"
	"middleware-gateway/middleware/ratelimit/infra"
)

type ConcurrencyOptions struct {
	Max           int
	RejectStatus  int
	AcquireTimeout time.Duration
}

func ConcurrencyMiddleware(opts ConcurrencyOptions) func(next http.Handler) http.Handler {
	if opts.Max <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	if opts.RejectStatus == 0 {
		opts.RejectStatus = http.StatusServiceUnavailable
	}

	svc := application.ConcurrencyService{
		Pool:           infra.NewChanPool(opts.Max),
		AcquireTimeout: opts.AcquireTimeout,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			release, ok := svc.Acquire(r.Context())
			if !ok {
				http.Error(w, http.StatusText(opts.RejectStatus), opts.RejectStatus)
				return
			}
			defer release()

			next.ServeHTTP(w, r)
		})
	}
}
