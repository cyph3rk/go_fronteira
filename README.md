
# Gateway / Middleware de Rate Limit (Go)

Este repositório contém:

- Um pacote de middleware para `net/http` com **rate limit** (por IP ou por header) e **limite de concorrência**.
- Um binário `gateway` que atua como **reverse-proxy** (middleware de conexão) para você colocar na frente de qualquer webserver.

## Rodar como reverse-proxy (gateway)

O `gateway` escuta em `LISTEN_ADDR` e encaminha tudo para `UPSTREAM_URL`.

Exemplo (rodando local):

```sh
UPSTREAM_URL="http://localhost:8081" \
LISTEN_ADDR=":8080" \
RATE_ENABLED=true \
RATE_RPS=10 \
RATE_BURST=20 \
CONCURRENCY_MAX=100 \
go run ./cmd/gateway
```

Variáveis de ambiente principais:

- `UPSTREAM_URL` (obrigatória): destino (ex: `http://localhost:8081`)
- `LISTEN_ADDR` (padrão `:8080`)
- `RATE_ENABLED` (padrão `true`)
- `RATE_RPS` (padrão `10`) e `RATE_BURST` (padrão `20`)
	- `RATE_BURST` é a “rajada” inicial: antes de começar a bloquear, ele pode deixar passar até `RATE_BURST` requisições quase instantaneamente.
	- Para testar um `RATE_RPS` bem baixo (ex: `0.02`), use `RATE_BURST=1` para o efeito ficar evidente.
- `RATE_KEY_HEADER` (opcional): ex `X-Api-Key` para limitar por chave
- `TRUST_XFF` (padrão `false`): usa `X-Forwarded-For` como IP do cliente
- `RETRY_AFTER` (padrão `1s`): valor do header `Retry-After` quando bloquear
- `ADD_RATELIMIT_HEADERS` (padrão `false`): adiciona headers informativos (debug)
- `RATE_STATS_ENABLED` (padrão `false`): habilita coleta/persistência de estatísticas do rate limit
	- `RATE_STATS_REDIS_ADDR` (obrigatória se `RATE_STATS_ENABLED=true`): ex `redis:6379` ou `localhost:6379`
	- `RATE_STATS_REDIS_PASSWORD` (opcional)
	- `RATE_STATS_REDIS_DB` (padrão `0`)
	- `RATE_STATS_PREFIX` (padrão `ratelimit:stats`)
	- `RATE_STATS_BUCKET` (padrão `minute`): `minute` (agrega por minuto) ou `none` (só total)
	- `RATE_STATS_TTL` (padrão `24h`): TTL aplicado às séries temporais (e por-key, se habilitar)
	- `RATE_STATS_TRACK_KEYS` (padrão `false`): registra por key (cuidado com cardinalidade)
- `CONCURRENCY_MAX` (padrão `100`)
- `CONCURRENCY_TIMEOUT` (padrão `0`): ex `200ms` para desistir de esperar vaga

## Exemplo: injetar middleware no seu webserver

O exemplo em `cmd/example-server` mostra como envolver um `http.Handler` com os middlewares:

```sh
LISTEN_ADDR=":8081" go run ./cmd/example-server

LISTEN_ADDR=":8080" UPSTREAM_URL="http://localhost:8081" go run ./cmd/gateway/main.go

```

## Docker Compose

O `docker-compose.yaml` traz um exemplo com `upstream` + `gateway`.

```sh
docker compose up --build
```

Teste:

```sh
curl -i http://localhost:8080/
```



## Testes automáticos

Para rodar **todos** os testes do módulo (a partir da raiz do repo):

```sh
cd /app/go_fronteira
go test ./...
```

O `./...` significa “rode os testes em todos os pacotes abaixo do diretório atual”.

Extras úteis:

```sh
go test -v ./...
go test -count=1 ./...
```

## Testando usando o CURL:

Exemplo rápido (testa bloqueio e mostra headers de debug):

```bash
UPSTREAM_URL="http://localhost:8081" \
LISTEN_ADDR=":8080" \
RATE_ENABLED=true \
RATE_RPS=0.02 \
RATE_BURST=1 \
ADD_RATELIMIT_HEADERS=true \
go run ./cmd/gateway

for i in $(seq 1 10); do
	curl -s -o /dev/null -D- http://localhost:8080/showTela | head -n 10
	echo "----"
done
```

## Documentação Go Doc

``` sh
go doc ./middleware/ratelimit

go doc ./middleware/ratelimit/application

go doc ./middleware/ratelimit/domain

go doc ./middleware/ratelimit/infra
```

# GERAL

## Comandos Docker Compose REMOVE TUDO

```sh
docker compose down
docker compose down -v
docker compose down --rmi all
docker compose down -v --rmi all

docker container prune
```

## Compactar e Criptografar

 * Para criptografar a pasta:

 ``` bash
 tar -cz pasta_estudos/ | gpg -c -o backup_projeto.tar.gz.gpg
 ```

 *Para descriptografar e extrair (na outra máquina):

 ``` bash
 gpg -d backup_projeto.tar.gz.gpg | tar -xz
 ```

 ## Testando o Gateway está falando com Redis

 ``` sh
 ocker exec -it $(docker ps -qf "name=redis") redis-cli
 ```

 * No container:

 ``` sh
 MONITOR
 ```

 * Em um terminal:

 ``` sh
 for i in {1..20}; do curl -I http://localhost:8080/showTela; done
 ```

 ## Testando Compose

 ``` sh
docker-compose up --build
 ```

 * Rodar o comando de subir os containers:
 ``` sh
 for i in {1..10}; do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/showTela; done
 ```

 * O que você deve ver:

	* 200
	* 200
	* 429 (Bloqueio pelo Rate Limit)