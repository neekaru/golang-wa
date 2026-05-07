#!/bin/bash

# Colors for better output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}Formatting Caddyfile...${NC}"

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed or not in PATH${NC}"
    exit 1
fi

# Format the Caddyfile using the Caddy container
docker run --rm -v "$(pwd)/Caddyfile:/etc/caddy/Caddyfile" caddy:2.7-alpine caddy fmt --overwrite /etc/caddy/Caddyfile

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Caddyfile formatted successfully${NC}"
else
    echo -e "${RED}Failed to format Caddyfile${NC}"
    exit 1
fi

echo -e "${BLUE}Restarting Caddy container...${NC}"
docker compose restart caddy

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Caddy container restarted successfully${NC}"
else
    echo -e "${RED}Failed to restart Caddy container${NC}"
    exit 1
fi 