package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"time"

	"middleware-gateway/middleware/ratelimit/application"
	"middleware-gateway/middleware/ratelimit/domain"
)

type KeyFunc func(r *http.Request) string

type Options struct {
	Store               domain.LimiterStore
	Stats               domain.StatsStore
	KeyFn               KeyFunc
	KeyHeader           string
	TrustXForwardedFor  bool
	RejectStatus        int
	RetryAfter          time.Duration
	AddRateLimitHeaders bool
}

type rateInfo interface {
	RPS() float64
	Burst() int
}

func DefaultKeyFunc(keyHeader string, trustXFF bool) KeyFunc {
	return func(r *http.Request) string {
		if keyHeader != "" {
			if v := strings.TrimSpace(r.Header.Get(keyHeader)); v != "" {
				return v
			}
		}

		if trustXFF {
			// pega o primeiro IP do X-Forwarded-For (cliente original)
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				parts := strings.Split(xff, ",")
				if len(parts) > 0 {
					ip := strings.TrimSpace(parts[0])
					if ip != "" {
						return ip
					}
				}
			}
		}

		// fallback: RemoteAddr
		host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
		if err == nil && host != "" {
			return host
		}
		if r.RemoteAddr != "" {
			return r.RemoteAddr
		}
		return "unknown"
	}
}

func Middleware(opts Options) func(next http.Handler) http.Handler {
	if opts.RejectStatus == 0 {
		opts.RejectStatus = http.StatusTooManyRequests
	}
	if opts.RetryAfter == 0 {
		opts.RetryAfter = 1 * time.Second
	}
	if opts.KeyFn == nil {
		opts.KeyFn = DefaultKeyFunc(opts.KeyHeader, opts.TrustXForwardedFor)
	}

	svc := application.Service{
		Store:      opts.Store,
		RetryAfter: opts.RetryAfter,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := opts.KeyFn(r)

			if opts.AddRateLimitHeaders {
				w.Header().Set("X-RateLimit-Key", key)
				if ri, ok := opts.Store.(rateInfo); ok {
					w.Header().Set("X-RateLimit-RPS", formatFloat(ri.RPS()))
					w.Header().Set("X-RateLimit-Burst", formatInt(ri.Burst()))
				}
			}

			dec := svc.Decide(domain.Key(key))
			if opts.Stats != nil {
				_ = opts.Stats.Record(r.Context(), domain.StatsEvent{
					Key:     domain.Key(key),
					Allowed: dec.Allowed,
					Method:  r.Method,
					Path:    r.URL.Path,
					At:      time.Now(),
				})
			}
			if !dec.Allowed {
				w.Header().Set("Retry-After", formatInt(int(dec.RetryAfter.Seconds())))
				http.Error(w, http.StatusText(opts.RejectStatus), opts.RejectStatus)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
