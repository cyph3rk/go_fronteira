// Package application contém os casos de uso (regras de aplicação) para rate limit
// e limite de concorrência.
//
// Ele depende apenas do pacote domain e não conhece net/http.
// Ex.: Service.Decide(key) retorna uma Decision (allow/deny + retry-after).
package application
