FROM golang:1.23 AS builder

WORKDIR /app

COPY src/chaos/server.go server.go

RUN CGO_ENABLED=0 GOOS=linux go build -o /server server.go

FROM alpine:latest

COPY --from=builder /server /server

ENTRYPOINT ["/server"]