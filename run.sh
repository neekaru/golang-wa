#!/bin/bash

# Colors for better output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Log function for consistent logging
log() {
    local level=$1
    local message=$2
    local color=$BLUE

    case "$level" in
        "INFO")
            color=$BLUE
            ;;
        "SUCCESS")
            color=$GREEN
            ;;
        "WARNING")
            color=$YELLOW
            ;;
        "ERROR")
            color=$RED
            ;;
    esac

    echo -e "${color}[$level] $message${NC}"
}

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
    echo -e "  ${GREEN}logs api${NC}   - Show logs for the WhatsApp API container"
    echo -e "  ${GREEN}status${NC}     - Check the status of all containers"
    echo -e "  ${GREEN}health${NC}     - Get basic health status of API"
    echo -e "  ${GREEN}health-detailed${NC} - Get detailed health status of API"
    echo -e "  ${GREEN}wa-status <user>${NC} - Check WhatsApp connection status for a user"
    echo -e "  ${GREEN}wa-restart <user>${NC} - Restart a WhatsApp session"
    echo -e "  ${GREEN}init-dirs${NC}  - Create data and logs directories"
    echo -e "  ${GREEN}app-logs${NC}   - View the WhatsApp application logs"
    echo -e "  ${GREEN}app-logs <user>${NC} - Filter logs for a specific user"
    echo -e "  ${GREEN}reset-images${NC} - Stop and remove all containers, images, and prune Docker resources"
    echo ""
}

# Check if Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}Error: Docker is not installed or not in PATH${NC}"
        exit 1
    fi
}

# Initialize data and logs directories
initialize_directories() {
    echo -e "${BLUE}Creating necessary directories...${NC}"

    # Create data directory if it doesn't exist
    if [ ! -d "data" ]; then
        mkdir -p data
        echo -e "${GREEN}Created data directory${NC}"
    else
        echo -e "${GREEN}Data directory already exists${NC}"
    fi

    # Create logs directory if it doesn't exist
    if [ ! -d "logs" ]; then
        mkdir -p logs
        echo -e "${GREEN}Created logs directory${NC}"
    else
        echo -e "${GREEN}Logs directory already exists${NC}"
    fi

    # Ensure proper permissions
    chmod -R 755 data logs

    echo -e "${GREEN}Directories initialized successfully${NC}"
}

# Start containers
start_containers() {
    # Ensure required directories exist before starting
    initialize_directories

    log "INFO" "Starting containers..."
    docker compose up -d
    if [ $? -eq 0 ]; then
        log "SUCCESS" "Containers started successfully"
    else
        log "ERROR" "Failed to start containers"
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
    # Ensure required directories exist before rebuilding
    initialize_directories

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
    elif [ "$1" = "api" ]; then
        echo -e "${BLUE}Showing logs for WhatsApp API container...${NC}"
        docker compose logs --tail=100 -f whatsapp-api
    else
        echo -e "${RED}Unknown container: $1${NC}"
        print_usage
        exit 1
    fi
}


# Show container status
show_status() {
    echo -e "${BLUE}Checking container status...${NC}"
    docker compose ps
}

# Check basic health status
check_health() {
    echo -e "${BLUE}Checking API health status...${NC}"
    response=$(curl -s http://localhost:8080/)

    if [ $? -eq 0 ]; then
        status=$(echo $response | jq -r '.status')
        uptime=$(echo $response | jq -r '.uptime')
        session_count=$(echo $response | jq -r '.session_count')
        version=$(echo $response | jq -r '.version')

        echo ""
        echo -e "Status: ${GREEN}$status${NC}"
        echo -e "Uptime: ${GREEN}$uptime${NC}"
        echo -e "Session Count: ${GREEN}$session_count${NC}"
        echo -e "Version: ${GREEN}$version${NC}"
    else
        echo -e "${RED}Failed to get health status${NC}"
    fi
}

# Check detailed health status
check_health_detailed() {
    echo -e "${BLUE}Checking detailed API health status...${NC}"
    response=$(curl -s http://localhost:8080/health)

    if [ $? -eq 0 ]; then
        status=$(echo $response | jq -r '.status')
        uptime=$(echo $response | jq -r '.uptime')
        total_sessions=$(echo $response | jq -r '.total_sessions')
        active_sessions=$(echo $response | jq -r '.active_sessions')
        timestamp=$(echo $response | jq -r '.timestamp')

        echo ""
        echo -e "Status: ${GREEN}$status${NC}"
        echo -e "Uptime: ${GREEN}$uptime${NC}"
        echo -e "Total Sessions: ${GREEN}$total_sessions${NC}"
        echo -e "Active Sessions: ${GREEN}$active_sessions${NC}"
        echo -e "Timestamp: ${GREEN}$timestamp${NC}"
    else
        echo -e "${RED}Failed to get detailed health status${NC}"
    fi
}

# Check WhatsApp session status
check_wa_status() {
    local user=$1

    if [ -z "$user" ]; then
        echo -e "${RED}Error: User parameter is required${NC}"
        return 1
    fi

    echo -e "${BLUE}Checking WhatsApp status for user $user...${NC}"
    response=$(curl -s "http://localhost:8080/wa/status?user=$user")

    if [ $? -eq 0 ]; then
        user=$(echo $response | jq -r '.user')
        logged_in=$(echo $response | jq -r '.logged_in')
        connected=$(echo $response | jq -r '.connected')

        echo ""
        echo -e "User: ${GREEN}$user${NC}"

        if [ "$logged_in" = "true" ]; then
            echo -e "Logged In: ${GREEN}Yes${NC}"
        else
            echo -e "Logged In: ${YELLOW}No${NC}"
        fi

        if [ "$connected" = "true" ]; then
            echo -e "Connected: ${GREEN}Yes${NC}"
        else
            echo -e "Connected: ${YELLOW}No${NC}"
        fi
    else
        echo -e "${RED}Failed to get WhatsApp status${NC}"
    fi
}

# Restart WhatsApp session
restart_wa_session() {
    local user=$1

    if [ -z "$user" ]; then
        echo -e "${RED}Error: User parameter is required${NC}"
        return 1
    fi

    echo -e "${BLUE}Restarting WhatsApp session for user $user...${NC}"
    response=$(curl -s -X POST "http://localhost:8080/wa/restart?user=$user")

    if [ $? -eq 0 ]; then
        msg=$(echo $response | jq -r '.msg')
        echo -e "Session restart result: ${GREEN}$msg${NC}"
    else
        echo -e "${RED}Failed to restart WhatsApp session${NC}"
    fi
}

# View WhatsApp application logs
view_app_logs() {
    local user=$1

    log "INFO" "Checking WhatsApp application logs..."

    # Check if logs directory exists
    if [ ! -d "logs" ]; then
        log "WARNING" "Logs directory not found. Creating it now..."
        mkdir -p logs
        log "WARNING" "No log files found yet. Start the application first."
        return
    fi

    # Get log files
    log_files=$(find logs -name "whatsapp-api-*.log" | sort -r)

    if [ -z "$log_files" ]; then
        log "WARNING" "No log files found in the logs directory"
        return
    fi

    # Get the most recent log file
    latest_log=$(echo "$log_files" | head -n 1)

    echo -e "Viewing latest log file: ${GREEN}$(basename $latest_log)${NC}"

    if [ -z "$user" ]; then
        # Show the entire log file
        tail -n 100 -f "$latest_log"
    else
        # Filter for a specific user
        log "INFO" "Filtering logs for user: $user"
        tail -f "$latest_log" | grep "$user"
    fi
}

# Reset Docker images and containers
reset_images() {
    log "INFO" "This will stop and remove ALL Docker containers, images, and volumes."
    log "WARNING" "This is a destructive operation and cannot be undone."

    read -p "Are you sure you want to continue? (y/N): " confirm
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        log "INFO" "Operation cancelled."
        return
    fi

    # Stop and remove all containers
    log "INFO" "Stopping and removing all containers..."
    docker stop $(docker ps -a -q) >/dev/null 2>&1 || true
    docker rm $(docker ps -a -q) >/dev/null 2>&1 || true

    # Remove all images
    log "INFO" "Removing all Docker images..."
    docker rmi $(docker images -a -q) >/dev/null 2>&1 || true

    # Prune everything
    log "INFO" "Pruning all Docker resources..."
    docker system prune -a -f --volumes
    docker builder prune -a -f

    log "SUCCESS" "Docker environment has been reset."
    log "INFO" "You can now rebuild the application with './run.sh rebuild'"
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
        status)
            show_status
            ;;
        health)
            check_health
            ;;
        health-detailed)
            check_health_detailed
            ;;
        wa-status)
            check_wa_status "$2"
            ;;
        wa-restart)
            restart_wa_session "$2"
            ;;
        init-dirs)
            initialize_directories
            ;;
        app-logs)
            view_app_logs "$2"
            ;;
        reset-images)
            reset_images
            ;;
        *)
            log "ERROR" "Unknown command: $1"
            print_usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"