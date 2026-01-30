package domain

import (
	"context"
	"time"
)

// StatsEvent representa um evento de decisão do rate limit.
//
// Ele é propositalmente "agnóstico de HTTP": Method/Path são strings genéricas
// e podem ser usadas para web, gRPC, etc.
//
// Observação: cuidado com cardinalidade (ex.: salvar Key/Path sem controle pode
// explodir o número de séries/chaves em uma base como Redis/Prometheus).
type StatsEvent struct {
	Key     Key
	Allowed bool

	Method string
	Path   string

	At time.Time
}

// StatsStore é a estratégia de persistência para estatísticas do rate limit.
//
// Implementações podem armazenar em Redis, Postgres, memória, etc.
// O middleware deve tratar erro como best-effort (não derrubar request).
type StatsStore interface {
	Record(ctx context.Context, ev StatsEvent) error
}
