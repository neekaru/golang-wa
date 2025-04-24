FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download

# Disable CGO but remove static build flags
RUN CGO_ENABLED=0 GOOS=linux go build -o bot .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates supervisor

WORKDIR /app

COPY --from=builder /app/bot /app/bot
COPY supervisor.conf /etc/supervisor/conf.d/supervisord.conf

RUN mkdir -p /var/log/supervisor /app/data && \
    chmod +x /app/bot

# Only exposing 8080 as Caddy will handle port 80
EXPOSE 8080

VOLUME ["/app/data"]

CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]