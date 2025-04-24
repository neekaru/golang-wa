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
  platforms = ["linux/amd64", "linux/arm64"]
}

// Multi-platform build
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

// x86 only build for latest tag
target "latest" {
  context = "."
  platforms = ["linux/amd64"]
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

// Default group includes both targets
group "default" {
  targets = ["main", "latest"]
} 