package infra

import (
	"context"

	"middleware-gateway/middleware/ratelimit/domain"
)

type chanPool struct {
	sem chan struct{}
}

// NewChanPool cria um pool simples baseado em channel com capacidade `max`.
func NewChanPool(max int) domain.SlotPool {
	return &chanPool{sem: make(chan struct{}, max)}
}

func (p *chanPool) Acquire(ctx context.Context) (func(), bool) {
	select {
	case p.sem <- struct{}{}:
		return func() { <-p.sem }, true
	case <-ctx.Done():
		return nil, false
	}
}
