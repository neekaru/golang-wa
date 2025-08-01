variable "REGISTRY" {
  default = "docker.io"
}

variable "DOCKERHUB_USERNAME" {
  default = "nekru"
}

variable "IMAGE_NAME" {
  default = "whatsmeow-maiga"
}

// Common settings for all targets
target "docker-metadata-action" {
  context = "."
  platforms = ["linux/amd64"]
}

// Main build
target "main" {
  inherits = ["docker-metadata-action"]
  tags = [
    "${REGISTRY}/${DOCKERHUB_USERNAME}/${IMAGE_NAME}:main"
  ]
  cache-from = [
    "type=gha,scope=main"
  ]
  cache-to = [
    "type=gha,mode=max,scope=main"
  ]
}

// Latest tag build
target "latest" {
  inherits = ["docker-metadata-action"]
  tags = [
    "${REGISTRY}/${DOCKERHUB_USERNAME}/${IMAGE_NAME}:latest"
  ]
  cache-from = [
    "type=gha,scope=latest"
  ]
  cache-to = [
    "type=gha,mode=max,scope=latest"
  ]
}

// Latest nginx build (from nginx-local branch)
target "latest-nginx" {
  inherits = ["docker-metadata-action"]
  tags = [
    "${REGISTRY}/${DOCKERHUB_USERNAME}/${IMAGE_NAME}:latest-nginx"
  ]
  cache-from = [
    "type=gha,scope=latest-nginx"
  ]
  cache-to = [
    "type=gha,mode=max,scope=latest-nginx"
  ]
}

// Default group includes both targets
group "default" {
  targets = ["main", "latest"]
}

// Nginx group for nginx builds
group "nginx" {
  targets = ["latest-nginx"]
} 