FROM golang:1.25-alpine AS build 

WORKDIR /app
RUN apk add --no-cache git
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go_fronteira

FROM scratch
WORKDIR /
COPY --from=build /go_fronteira .

RUN chmod +x /go_fronteira

EXPOSE 8080

ENTRYPOINT ["./go_fronteira"]