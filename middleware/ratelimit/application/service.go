package application

import (
	"time"

	"middleware-gateway/middleware/ratelimit/domain"
)

// Service concentra a regra de aplicação do rate limit.
//
// Ele não sabe nada sobre HTTP (headers/status), apenas retorna uma decisão.
type Service struct {
	Store      domain.LimiterStore
	RetryAfter time.Duration
}

func (s Service) Decide(key domain.Key) domain.Decision {
	if s.Store == nil {
		return domain.Decision{Allowed: true}
	}
	if s.RetryAfter <= 0 {
		s.RetryAfter = 1 * time.Second
	}

	lim := s.Store.Get(key)
	if lim == nil {
		return domain.Decision{Allowed: true}
	}
	if lim.Allow() {
		return domain.Decision{Allowed: true}
	}
	return domain.Decision{Allowed: false, RetryAfter: s.RetryAfter}
}
