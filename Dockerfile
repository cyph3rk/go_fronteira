FROM golang:1.23-alpine AS build 
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Alvo para o Gateway
FROM build AS build-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/gateway ./cmd/gateway

# Alvo para o Upstream
FROM build AS build-upstream
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/upstream ./cmd/example-server

# Imagem Final (Leve)
FROM alpine:latest
WORKDIR /app
COPY --from=build-gateway /app/bin/gateway .
COPY --from=build-upstream /app/bin/upstream .
# Alpine precisa de permissão se você for rodar scripts, mas binários Go rodam direto