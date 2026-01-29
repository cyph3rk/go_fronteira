package application

import (
	"context"
	"time"

	"middleware-gateway/middleware/ratelimit/domain"
)

// ConcurrencyService concentra a regra de aquisição/liberação de vagas com timeout,
// sem saber nada sobre HTTP.
type ConcurrencyService struct {
	Pool           domain.SlotPool
	AcquireTimeout time.Duration
}

// Acquire tenta adquirir uma vaga.
// - Se `AcquireTimeout <= 0`, espera indefinidamente (até ctx cancelar).
// - Se `AcquireTimeout > 0`, espera até o timeout.
// Retorna (release, ok). Se ok=false, nenhuma vaga foi adquirida.
func (s ConcurrencyService) Acquire(ctx context.Context) (func(), bool) {
	if s.Pool == nil {
		return func() {}, true
	}

	if s.AcquireTimeout <= 0 {
		return s.Pool.Acquire(ctx)
	}

	acqCtx, cancel := context.WithTimeout(ctx, s.AcquireTimeout)
	defer cancel()
	return s.Pool.Acquire(acqCtx)
}
