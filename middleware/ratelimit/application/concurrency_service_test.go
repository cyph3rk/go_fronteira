package application

import (
	"context"
	"testing"
	"time"
)

type blockingPool struct {
}

func (p *blockingPool) Acquire(ctx context.Context) (func(), bool) {
	select {
	case <-ctx.Done():
		return nil, false
	case <-time.After(5 * time.Second):
		// não deve chegar aqui nos testes
		return nil, false
	}
}

type immediatePool struct {
	acquired int
}

func (p *immediatePool) Acquire(ctx context.Context) (func(), bool) {
	p.acquired++
	return func() {}, true
}

func TestConcurrencyService_Acquire_AllowsWhenNoPool(t *testing.T) {
	svc := ConcurrencyService{}
	release, ok := svc.Acquire(context.Background())
	if !ok {
		t.Fatalf("expected ok")
	}
	release()
}

func TestConcurrencyService_Acquire_UsesTimeout(t *testing.T) {
	pool := &blockingPool{}
	svc := ConcurrencyService{Pool: pool, AcquireTimeout: 10 * time.Millisecond}

	_, ok := svc.Acquire(context.Background())
	if ok {
		t.Fatalf("expected timeout and ok=false")
	}
}

func TestConcurrencyService_Acquire_NoTimeoutDelegatesToPool(t *testing.T) {
	pool := &immediatePool{}
	svc := ConcurrencyService{Pool: pool, AcquireTimeout: 0}

	_, ok := svc.Acquire(context.Background())
	if !ok {
		t.Fatalf("expected ok")
	}
	if pool.acquired != 1 {
		t.Fatalf("expected pool Acquire to be called once, got %d", pool.acquired)
	}
}
