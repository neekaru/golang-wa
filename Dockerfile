FROM golang:1.25.5-alpine3.23 AS builder

WORKDIR /app
COPY . .

# Install C compiler and build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

RUN go mod download

# Disable CGO but remove static build flags
RUN CGO_ENABLED=1 GOOS=linux go build -o bot .

FROM alpine:3.23.2

RUN apk add --no-cache ca-certificates supervisor sqlite curl ffmpeg

WORKDIR /app

COPY --from=builder /app/bot /app/bot
COPY supervisor.conf /etc/supervisor/conf.d/supervisord.conf

# Create necessary directories and set permissions
RUN mkdir -p /var/log/supervisor /app/data /app/logs && \
    chmod +x /app/bot && \
    chmod -R 755 /app/data /app/logs

# Add a daily log rotation check to ensure logs are created daily
# This is a fallback in case the application's internal log rotation fails
RUN echo '0 0 * * * find /app/logs -name "whatsapp-api-*.log" -mtime +7 -delete' > /etc/crontabs/root

# Expose port 3000 for the API
EXPOSE 3000

# Define volumes for both data and logs
VOLUME ["/app/data", "/app/logs"]

CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]

