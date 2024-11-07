FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY src/ ./src

RUN CGO_ENABLED=0 GOOS=linux go build -o /worker src/worker/main.go

# FROM alpine:latest

# COPY --from=builder /worker /worker

ENTRYPOINT ["/worker"]