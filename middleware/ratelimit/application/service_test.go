package application

import (
	"testing"
	"time"

	"middleware-gateway/middleware/ratelimit/domain"
)

type fakeLimiter struct {
	allow bool
}

func (f fakeLimiter) Allow() bool { return f.allow }

type fakeStore struct {
	lim domain.Limiter
}

func (s fakeStore) Get(domain.Key) domain.Limiter { return s.lim }

func TestService_Decide_AllowsWhenNoStore(t *testing.T) {
	svc := Service{}
	dec := svc.Decide("k")
	if !dec.Allowed {
		t.Fatalf("expected allowed")
	}
	if dec.RetryAfter != 0 {
		t.Fatalf("expected RetryAfter=0 when allowed, got %s", dec.RetryAfter)
	}
}

func TestService_Decide_AllowsWhenLimiterAllows(t *testing.T) {
	svc := Service{Store: fakeStore{lim: fakeLimiter{allow: true}}, RetryAfter: 5 * time.Second}
	dec := svc.Decide("k")
	if !dec.Allowed {
		t.Fatalf("expected allowed")
	}
}

func TestService_Decide_BlocksWithRetryAfterDefault(t *testing.T) {
	svc := Service{Store: fakeStore{lim: fakeLimiter{allow: false}}}
	dec := svc.Decide("k")
	if dec.Allowed {
		t.Fatalf("expected blocked")
	}
	if dec.RetryAfter != 1*time.Second {
		t.Fatalf("expected default RetryAfter=1s, got %s", dec.RetryAfter)
	}
}

func TestService_Decide_BlocksWithConfiguredRetryAfter(t *testing.T) {
	svc := Service{Store: fakeStore{lim: fakeLimiter{allow: false}}, RetryAfter: 2500 * time.Millisecond}
	dec := svc.Decide("k")
	if dec.Allowed {
		t.Fatalf("expected blocked")
	}
	if dec.RetryAfter != 2500*time.Millisecond {
		t.Fatalf("expected RetryAfter=2.5s, got %s", dec.RetryAfter)
	}
}
