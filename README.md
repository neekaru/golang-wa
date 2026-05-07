# WhatsApp Go API

A WhatsApp API built with Go using the [whatsmeow](https://github.com/tulir/whatsmeow) library and Gin framework.

## Features

- Multiple WhatsApp session management
- Send text messages, images, videos, and files
- QR code generation for authentication
- Mark messages as read
- Session status checking and management

## Docker Setup

This project includes a Docker setup with Caddy as a reverse proxy for improved security and performance.

### Quick Start with Docker Hub Image

1. Create your environment file:
```bash
# Copy the example environment file
cp .env.example .env

# Edit the .env file with your settings
nano .env
```

2. Start the containers:
```bash
docker-compose up -d

# Access the API
curl http://localhost/
# Should return: {"msg":"it works"}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| DOCKER_USERNAME | Your Docker Hub username | yourusername |
| TAG | Docker image tag to use | latest |
| API_PORT | Port for the WhatsApp API | 8080 |
| HTTP_PORT | HTTP port for Caddy | 80 |
| HTTPS_PORT | HTTPS port for Caddy | 443 |
| DATA_DIR | Directory for WhatsApp data | ./whatsmeow-data |
| TZ | Container timezone | Asia/Jakarta |

### Health Checks

The setup includes health checks for both services:
- WhatsApp API: Checks the API endpoint every 30 seconds
- Caddy: Verifies Caddy is running every 30 seconds

## Docker Images

The project provides Docker images with different platform support:

### Latest Tag (x86_64 only)
```bash
# Use the latest stable version (x86_64/amd64 only)
docker pull nekru/whatsmeow-maiga:latest
```

### Main Branch (Multi-platform)
```bash
# Use the main branch version (supports both x86_64 and arm64)
docker pull nekru/whatsmeow-maiga:main
```

Platform support:
- `latest` tag: linux/amd64 (x86_64) only
- `main` tag: linux/amd64 (x86_64) and linux/arm64 (ARM64)

## GitHub Actions Deployment

This repository is configured with GitHub Actions to automatically build and publish the Docker image to Docker Hub.

### Setup for Docker Hub Deployment

1. In your GitHub repository, go to Settings > Secrets and variables > Actions
2. Add the following secrets:
   - `DOCKERHUB_USERNAME`: Your Docker Hub username
   - `DOCKERHUB_TOKEN`: Your Docker Hub access token (create one at https://hub.docker.com/settings/security)

3. The workflow will automatically:
   - Build multi-platform images (amd64, arm64) for the main branch
   - Build x86_64 only image for the latest tag
   - Push images to Docker Hub under your username/whatsmeow-maiga

### Using Different Tags

```bash
# Use the latest stable version (x86_64 only)
TAG=latest docker-compose up -d

# Use the main branch version (multi-platform)
TAG=main docker-compose up -d
```

## API Documentation

For detailed API documentation, see [docs.md](docs.md).

## License

MIT 