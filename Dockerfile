FROM golang:1.24.2-alpine AS builder

WORKDIR /app
COPY . .

# Install C compiler and build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

RUN go mod download

# Disable CGO but remove static build flags
RUN CGO_ENABLED=1 GOOS=linux go build -o bot .

FROM alpine:3.21.3

RUN apk add --no-cache ca-certificates supervisor sqlite curl

WORKDIR /app

COPY --from=builder /app/bot /app/bot
COPY supervisor.conf /etc/supervisor/conf.d/supervisord.conf

# Create necessary directories and set permissions
RUN mkdir -p /var/log/supervisor /app/data /app/logs && \
    chmod +x /app/bot && \
    chmod -R 755 /app/data /app/logs

# Only exposing 8080 as Caddy will handle port 80
EXPOSE 8080

# Define volumes for both data and logs
VOLUME ["/app/data", "/app/logs"]

CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]