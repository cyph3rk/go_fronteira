package infra

import (
	"testing"
	"time"

	"middleware-gateway/middleware/ratelimit/domain"
)

func TestStore_GetSameKeyReturnsSameLimiter(t *testing.T) {
	s := NewStore(10, 1)

	l1 := s.Get(domain.Key("k"))
	l2 := s.Get(domain.Key("k"))
	if l1 != l2 {
		t.Fatalf("expected same limiter pointer for same key")
	}
}

func TestStore_LowBurstRejectsSecondImmediateAllow(t *testing.T) {
	s := NewStore(0.02, 1)

	lim := s.Get(domain.Key("k"))
	if !lim.Allow() {
		t.Fatalf("expected first Allow to be true")
	}
	if lim.Allow() {
		t.Fatalf("expected second immediate Allow to be false (burst=1)")
	}
}

func TestStore_CleanupRemovesIdleEntries(t *testing.T) {
	s := NewStore(10, 1, WithIdleTTL(2*time.Millisecond), WithCleanupEvery(0))

	before := s.Get(domain.Key("k"))
	time.Sleep(4 * time.Millisecond)

	s.Cleanup()

	after := s.Get(domain.Key("k"))
	if before == after {
		t.Fatalf("expected limiter to be recreated after cleanup")
	}
}
