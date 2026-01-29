// Package infra contém implementações concretas (infraestrutura) para os contratos
// definidos no pacote domain.
//
// Exemplos:
//   - Store: token bucket por chave usando golang.org/x/time/rate
//   - ChanPool: semáforo simples para limite de concorrência
package infra
