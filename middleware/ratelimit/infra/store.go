package infra

import (
	"sync"
	"time"

	"middleware-gateway/middleware/ratelimit/domain"

	"golang.org/x/time/rate"
)

// Store é uma implementação de infra baseada em token-bucket (x/time/rate)
// com cache por chave e limpeza periódica.
type Store struct {
	mu           sync.Mutex
	entries      map[string]*storeEntry
	rps          rate.Limit
	burst        int
	idleTTL      time.Duration
	cleanupEvery time.Duration
}

type storeEntry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

type StoreOption func(*Store)

func WithIdleTTL(d time.Duration) StoreOption {
	return func(s *Store) { s.idleTTL = d }
}

func WithCleanupEvery(d time.Duration) StoreOption {
	return func(s *Store) { s.cleanupEvery = d }
}

func NewStore(rps float64, burst int, opts ...StoreOption) *Store {
	s := &Store{
		entries:      make(map[string]*storeEntry),
		rps:          rate.Limit(rps),
		burst:        burst,
		idleTTL:      15 * time.Minute,
		cleanupEvery: 2 * time.Minute,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Store) RPS() float64  { return float64(s.rps) }
func (s *Store) Burst() int   { return s.burst }
func (s *Store) CleanupEvery() time.Duration { return s.cleanupEvery }

// Get implementa domain.LimiterStore.
func (s *Store) Get(key domain.Key) domain.Limiter {
	return s.GetString(string(key))
}

func (s *Store) GetString(key string) *rate.Limiter {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if ent, ok := s.entries[key]; ok {
		ent.lastSeen = now
		return ent.lim
	}

	lim := rate.NewLimiter(s.rps, s.burst)
	s.entries[key] = &storeEntry{lim: lim, lastSeen: now}
	return lim
}

func (s *Store) Cleanup() {
	cutoff := time.Now().Add(-s.idleTTL)

	s.mu.Lock()
	defer s.mu.Unlock()

	for k, ent := range s.entries {
		if ent.lastSeen.Before(cutoff) {
			delete(s.entries, k)
		}
	}
}

// StartJanitor inicia uma goroutine que limpa chaves inativas periodicamente.
// Pare cancelando o contexto.
func (s *Store) StartJanitor(ctx DoneContext) {
	if s.cleanupEvery <= 0 {
		return
	}

	t := time.NewTicker(s.cleanupEvery)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.Cleanup()
			}
		}
	}()
}

// DoneContext é o mínimo necessário para aceitar context.Context sem importar context aqui.
// (Permite reuso em libs sem acoplar.)
type DoneContext interface {
	Done() <-chan struct{}
}
