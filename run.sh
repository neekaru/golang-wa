#!/bin/bash

# Colors for better output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Print header
print_header() {
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE}    WhatsApp API Docker Management Script${NC}"
    echo -e "${BLUE}================================================${NC}"
    echo ""
}

# Print usage information
print_usage() {
    echo -e "Usage: ${GREEN}./run.sh [command]${NC}"
    echo ""
    echo -e "Commands:"
    echo -e "  ${GREEN}start${NC}      - Start all containers"
    echo -e "  ${GREEN}stop${NC}       - Stop all containers"
    echo -e "  ${GREEN}restart${NC}    - Restart all containers"
    echo -e "  ${GREEN}rebuild${NC}    - Rebuild and restart all containers"
    echo -e "  ${GREEN}logs${NC}       - Show logs for all containers"
    echo -e "  ${GREEN}logs caddy${NC} - Show logs for the Caddy container"
    echo -e "  ${GREEN}logs api${NC}   - Show logs for the WhatsApp API container"
    echo -e "  ${GREEN}format-caddy${NC} - Format the Caddyfile and restart Caddy"
    echo -e "  ${GREEN}status${NC}     - Check the status of all containers"
    echo ""
}

# Check if Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}Error: Docker is not installed or not in PATH${NC}"
        exit 1
    fi
}

# Start containers
start_containers() {
    echo -e "${BLUE}Starting containers...${NC}"
    docker compose up -d
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Containers started successfully${NC}"
    else
        echo -e "${RED}Failed to start containers${NC}"
    fi
}

# Stop containers
stop_containers() {
    echo -e "${BLUE}Stopping containers...${NC}"
    docker compose down
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Containers stopped successfully${NC}"
    else
        echo -e "${RED}Failed to stop containers${NC}"
    fi
}

# Restart containers
restart_containers() {
    echo -e "${BLUE}Restarting containers...${NC}"
    docker compose restart
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Containers restarted successfully${NC}"
    else
        echo -e "${RED}Failed to restart containers${NC}"
    fi
}

# Rebuild containers
rebuild_containers() {
    echo -e "${BLUE}Rebuilding and starting containers...${NC}"
    docker compose down
    docker compose build --no-cache
    docker compose up -d
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Containers rebuilt and started successfully${NC}"
    else
        echo -e "${RED}Failed to rebuild containers${NC}"
    fi
}

# Show logs
show_logs() {
    if [ -z "$1" ]; then
        echo -e "${BLUE}Showing logs for all containers...${NC}"
        docker compose logs --tail=100 -f
    elif [ "$1" = "caddy" ]; then
        echo -e "${BLUE}Showing logs for Caddy container...${NC}"
        docker compose logs --tail=100 -f caddy
    elif [ "$1" = "api" ]; then
        echo -e "${BLUE}Showing logs for WhatsApp API container...${NC}"
        docker compose logs --tail=100 -f whatsapp-api
    else
        echo -e "${RED}Unknown container: $1${NC}"
        print_usage
        exit 1
    fi
}

# Format Caddyfile
format_caddyfile() {
    echo -e "${BLUE}Formatting Caddyfile...${NC}"
    docker run --rm -v "$(pwd)/Caddyfile:/etc/caddy/Caddyfile" caddy:2.7-alpine caddy fmt --overwrite /etc/caddy/Caddyfile
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Caddyfile formatted successfully${NC}"
        
        # Restart Caddy to apply changes
        echo -e "${BLUE}Restarting Caddy container...${NC}"
        docker compose restart caddy
        
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}Caddy container restarted successfully${NC}"
        else
            echo -e "${RED}Failed to restart Caddy container${NC}"
        fi
    else
        echo -e "${RED}Failed to format Caddyfile${NC}"
    fi
}

# Show container status
show_status() {
    echo -e "${BLUE}Checking container status...${NC}"
    docker compose ps
}

# Main function
main() {
    print_header
    check_docker

    if [ $# -eq 0 ]; then
        print_usage
        exit 0
    fi

    case "$1" in
        start)
            start_containers
            ;;
        stop)
            stop_containers
            ;;
        restart)
            restart_containers
            ;;
        rebuild)
            rebuild_containers
            ;;
        logs)
            show_logs "$2"
            ;;
        format-caddy)
            format_caddyfile
            ;;
        status)
            show_status
            ;;
        *)
            echo -e "${RED}Unknown command: $1${NC}"
            print_usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@" 