variable "REGISTRY" {
  default = "docker.io"
}

variable "DOCKERHUB_USERNAME" {
  default = ""
}

variable "IMAGE_NAME" {
  default = "whatsmeow-maiga"
}

// Common settings for all targets
target "docker-metadata-action" {
  context = "."
  dockerfile = "Dockerfile"
  platforms = ["linux/amd64", "linux/arm64"]
}

// Main branch target - multi-platform
target "main" {
  inherits = ["docker-metadata-action"]
  tags = ["${REGISTRY}/${DOCKERHUB_USERNAME}/${IMAGE_NAME}:main"]
}

// Latest tag target - x86 only
target "latest" {
  context = "."
  dockerfile = "Dockerfile"
  platforms = ["linux/amd64"]
  tags = ["${REGISTRY}/${DOCKERHUB_USERNAME}/${IMAGE_NAME}:latest"]
}

// Default group includes both targets for concurrent building
group "default" {
  targets = ["main", "latest"]
} 