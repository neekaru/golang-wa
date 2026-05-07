# Docker Setup with Caddy Reverse Proxy

This project uses Docker and Caddy as a reverse proxy to provide the following benefits:

- Automatic HTTP to HTTPS redirection
- Improved security with proper headers
- Rate limiting for DoS protection
- Better request handling and load balancing

## Architecture

The setup consists of two main services:

1. **WhatsApp Bot (Go API)**: Runs on port 8080 inside the Docker network
2. **Caddy Proxy**: Handles incoming requests on port 80 (HTTP) and forwards them to the API

## Getting Started

### Prerequisites

- Docker and Docker Compose installed
- Basic understanding of Docker

### Starting the Services

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Accessing the API

The API can be accessed via:

- http://localhost/ (Caddy handles the request and proxies to the API)
- http://localhost:8080/ (Direct access to the API, bypassing Caddy)

## Configuration Files

- **docker-compose.yml**: Defines the services, networks, and volumes
- **Dockerfile**: Builds the WhatsApp bot Go application
- **Caddyfile**: Configures the Caddy reverse proxy
- **supervisor.conf**: Manages the bot process inside the container

## Customization

### Custom Domain

To use a custom domain with Caddy, modify the Caddyfile:

```
example.com {
    # Rest of the configuration remains the same
}
```

### SSL Certificates

Caddy automatically handles SSL certificates for custom domains. Just specify the domain in the Caddyfile and ensure your DNS points to the server.

### Data Persistence

The following data is persisted:

- WhatsApp session data: `./whatsmeow-data` directory
- Caddy certificates: Docker volume `caddy_data`
- Caddy configuration: Docker volume `caddy_config`

## Troubleshooting

- **API not accessible**: Check if both containers are running with `docker-compose ps`
- **Permission issues**: Ensure the data directories have proper permissions
- **SSL certificate problems**: Check Caddy logs with `docker-compose logs caddy` 