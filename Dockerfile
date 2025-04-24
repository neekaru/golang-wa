FROM golang:1.24.2-alpine3.21 AS builder

WORKDIR /app
COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o bot .

FROM debian:bullseye-slim

RUN apt-get update && \
    apt-get install -y ca-certificates supervisor && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/bot /app/bot
COPY supervisor.conf /etc/supervisor/conf.d/supervisord.conf

RUN mkdir -p /var/log/supervisor /app/data

EXPOSE 8080 80

VOLUME ["/app/data"]

CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]