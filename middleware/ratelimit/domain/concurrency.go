package domain

import "context"

// SlotPool representa um recurso com capacidade finita (ex: conexões concorrentes).
//
// A semântica é: Acquire bloqueia até conseguir uma vaga ou até o ctx encerrar.
// Ao adquirir, retorna uma função de release que deve ser chamada exatamente uma vez.
type SlotPool interface {
	Acquire(ctx context.Context) (release func(), ok bool)
}
