package domain

// Camada de domínio do rate limit.
//
// Regras e contratos (interfaces/tipos) sem dependência de net/http.

import "time"

type Key string

// Limiter representa algo que pode decidir se uma ação é permitida agora.
//
// Observação: a implementação pode ser token-bucket, leaky-bucket, etc.
// A camada de infra pode usar libs como golang.org/x/time/rate.
type Limiter interface {
	Allow() bool
}

// LimiterStore obtém um limiter por chave (ex: IP, API key, usuário).
// A implementação pode manter cache, TTL, etc.
type LimiterStore interface {
	Get(Key) Limiter
}

type Decision struct {
	Allowed bool
	// RetryAfter é o valor a ser retornado em Retry-After quando bloquear.
	// Se 0, não há recomendação.
	RetryAfter time.Duration
}
