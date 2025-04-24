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

### Quick Start

```bash
# Start the containers
docker-compose up -d

# Access the API
curl http://localhost/
# Should return: {"msg":"it works"}
```

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
```

## API Documentation

For detailed API documentation, see [docs.md](docs.md).

## License

MIT 