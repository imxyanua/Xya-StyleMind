FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /stylemind ./cmd/server

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /stylemind /app/stylemind
COPY migrations /app/migrations

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD wget -q -O- http://127.0.0.1:${PORT:-8080}/readyz || exit 1

CMD ["/app/stylemind"]
