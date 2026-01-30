package infra

import (
	"context"
	"sync"

	"middleware-gateway/middleware/ratelimit/domain"
)

type Counters struct {
	Allowed int64
	Denied  int64
}

// MemoryStatsStore é uma implementação simples em memória.
// Útil para testes e desenvolvimento.
//
// Não faz expiração e não é indicada para produção.
type MemoryStatsStore struct {
	mu      sync.Mutex
	total   Counters
	byRoute map[string]Counters
	byKey   map[string]Counters

	trackKeys bool
}

type MemoryStatsOption func(*MemoryStatsStore)

func WithTrackKeys(track bool) MemoryStatsOption {
	return func(s *MemoryStatsStore) { s.trackKeys = track }
}

func NewMemoryStatsStore(opts ...MemoryStatsOption) *MemoryStatsStore {
	s := &MemoryStatsStore{
		byRoute: make(map[string]Counters),
		byKey:   make(map[string]Counters),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *MemoryStatsStore) Record(_ context.Context, ev domain.StatsEvent) error {
	key := string(ev.Key)
	route := ev.Method + " " + ev.Path

	s.mu.Lock()
	defer s.mu.Unlock()

	if ev.Allowed {
		s.total.Allowed++
		c := s.byRoute[route]
		c.Allowed++
		s.byRoute[route] = c
		if s.trackKeys {
			k := s.byKey[key]
			k.Allowed++
			s.byKey[key] = k
		}
		return nil
	}

	s.total.Denied++
	c := s.byRoute[route]
	c.Denied++
	s.byRoute[route] = c
	if s.trackKeys {
		k := s.byKey[key]
		k.Denied++
		s.byKey[key] = k
	}
	return nil
}

func (s *MemoryStatsStore) Total() Counters {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.total
}

func (s *MemoryStatsStore) ByRoute() map[string]Counters {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]Counters, len(s.byRoute))
	for k, v := range s.byRoute {
		out[k] = v
	}
	return out
}

func (s *MemoryStatsStore) ByKey() map[string]Counters {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]Counters, len(s.byKey))
	for k, v := range s.byKey {
		out[k] = v
	}
	return out
}
