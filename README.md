
# DESAFIO: Gateway / Middleware de Rate Limit (Go)

**Objetivo**: Desenvolver um rate limiter em Go que possa ser configurado para limitar o número máximo de requisições por segundo com base em um endereço IP específico ou em um token de acesso.

**Descrição**: O objetivo deste desafio é criar um rate limiter em Go que possa ser utilizado para controlar o tráfego de requisições para um serviço web. O rate limiter deve ser capaz de limitar o número de requisições com base em dois critérios:

* 1. Endereço IP: O rate limiter deve restringir o número de requisições recebidas de um único endereço IP dentro de um intervalo de tempo definido.
* 2. Token de Acesso: O rate limiter deve também poderá limitar as requisições baseadas em um token de acesso único, permitindo diferentes limites de tempo de expiração para diferentes tokens. O Token deve ser informado no header no seguinte formato:
	* 1. API_KEY: <TOKEN>
* 3. As configurações de limite do token de acesso devem se sobrepor as do IP. Ex: Se o limite por IP é de 10 req/s e a de um determinado token é de 100 req/s, o rate limiter deve utilizar as informações do token.

**Requisitos:**

* O rate limiter deve poder trabalhar como um middleware que é injetado ao servidor web
* O rate limiter deve permitir a configuração do número máximo de requisições permitidas por segundo.
* O rate limiter deve ter ter a opção de escolher o tempo de bloqueio do IP ou do Token caso a quantidade de requisições tenha sido excedida.
* As configurações de limite devem ser realizadas via variáveis de ambiente ou em um arquivo “.env” na pasta raiz.
* Deve ser possível configurar o rate limiter tanto para limitação por IP quanto por token de acesso.
* O sistema deve responder adequadamente quando o limite é excedido:
	* Código HTTP: 429
	* Mensagem: you have reached the maximum number of requests or actions allowed within a certain time frame
* Todas as informações de "limiter” devem ser armazenadas e consultadas de um banco de dados Redis. Você pode utilizar docker-compose para subir o Redis.
* Crie uma “strategy” que permita trocar facilmente o Redis por outro mecanismo de persistência.
* A lógica do limiter deve estar separada do middleware.

**Exemplos:**

* 1. Limitação por IP: Suponha que o rate limiter esteja configurado para permitir no máximo 5 requisições por segundo por IP. Se o IP **192.168.1.1** enviar 6 requisições em um segundo, a sexta requisição deve ser bloqueada.
* 2. **Limitação por Token**: Se um token **abc123** tiver um limite configurado de 10 requisições por segundo e enviar 11 requisições nesse intervalo, a décima primeira deve ser bloqueada.
* 3. Nos dois casos acima, as próximas requisições poderão ser realizadas somente quando o tempo total de expiração ocorrer. Ex: Se o tempo de expiração é de 5 minutos, determinado IP poderá realizar novas requisições somente após os 5 minutos.

**Dicas:**

* Teste seu rate limiter sob diferentes condições de carga para garantir que ele funcione conforme esperado em situações de alto tráfego.

**Entrega:**

* O código-fonte completo da implementação.
* Documentação explicando como o rate limiter funciona e como ele pode ser configurado.
* Testes automatizados demonstrando a eficácia e a robustez do rate limiter.
* Utilize docker/docker-compose para que possamos realizar os testes de sua aplicação.
* O servidor web deve responder na porta 8080.







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