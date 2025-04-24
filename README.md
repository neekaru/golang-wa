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

## GitHub Actions Deployment

This repository is configured with GitHub Actions to automatically build and publish the Docker image to Docker Hub.

### Setup for Docker Hub Deployment

1. In your GitHub repository, go to Settings > Secrets and variables > Actions
2. Add the following secrets:
   - `DOCKERHUB_USERNAME`: Your Docker Hub username
   - `DOCKERHUB_TOKEN`: Your Docker Hub access token (create one at https://hub.docker.com/settings/security)

3. The workflow will automatically:
   - Build the Docker image when you push to the main branch
   - Tag the image properly when you create a release tag (e.g., v1.0.0)
   - Push the image to Docker Hub under your username/whatsmeow-maiga

### Using Tagged Releases

To create a specific version:

```bash
# Tag a release
git tag v1.0.0
git push origin v1.0.0

# This will trigger the workflow to build and push:
# yourusername/whatsmeow-maiga:1.0.0
# yourusername/whatsmeow-maiga:1.0
# yourusername/whatsmeow-maiga:1
# yourusername/whatsmeow-maiga:latest

# To use a specific version in docker-compose:
TAG=1.0.0 docker-compose up -d
```

## API Documentation

For detailed API documentation, see [docs.md](docs.md).

## License

MIT 