// Package ratelimit fornece adapters HTTP (net/http) para rate limit e limite de concorrência.
//
// Visão geral (camadas):
//
//   - domain: contratos e tipos do domínio (sem dependência de net/http)
//   - application: casos de uso (decisão allow/deny, acquire/timeout) sem net/http
//   - infra: implementações concretas (token bucket, semáforo), detalhes de infraestrutura
//   - ratelimit (este pacote): middlewares HTTP + wiring/extração de chave + tradução para status/headers
//
// Fluxo no gateway:
//
//   1) Extrai a chave do cliente (IP/header/XFF)
//   2) Chama a camada application para obter a decisão
//   3) Se bloqueado, responde 429 (rate limit) ou 503 (concorrência)
//   4) Se permitido, chama o próximo handler (ex: reverse proxy)
//
// Variáveis de ambiente do binário gateway (cmd/gateway) controlam o comportamento,
// como RATE_RPS, RATE_BURST, CONCURRENCY_MAX e CONCURRENCY_TIMEOUT.
package ratelimit
