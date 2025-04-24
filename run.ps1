# Colors for better output
$green = [System.ConsoleColor]::Green
$blue = [System.ConsoleColor]::Blue
$red = [System.ConsoleColor]::Red
$yellow = [System.ConsoleColor]::Yellow
$default = [System.ConsoleColor]::White

# Print header
function Print-Header {
    Write-Host "================================================" -ForegroundColor $blue
    Write-Host "    WhatsApp API Docker Management Script" -ForegroundColor $blue
    Write-Host "================================================" -ForegroundColor $blue
    Write-Host ""
}

# Print usage information
function Print-Usage {
    Write-Host "Usage: " -NoNewline
    Write-Host ".\run.ps1 [command]" -ForegroundColor $green
    Write-Host ""
    Write-Host "Commands:"
    Write-Host "  " -NoNewline
    Write-Host "start" -ForegroundColor $green -NoNewline
    Write-Host "      - Start all containers"
    Write-Host "  " -NoNewline
    Write-Host "stop" -ForegroundColor $green -NoNewline
    Write-Host "       - Stop all containers"
    Write-Host "  " -NoNewline
    Write-Host "restart" -ForegroundColor $green -NoNewline
    Write-Host "    - Restart all containers"
    Write-Host "  " -NoNewline
    Write-Host "rebuild" -ForegroundColor $green -NoNewline
    Write-Host "    - Rebuild and restart all containers"
    Write-Host "  " -NoNewline
    Write-Host "logs" -ForegroundColor $green -NoNewline
    Write-Host "       - Show logs for all containers"
    Write-Host "  " -NoNewline
    Write-Host "logs caddy" -ForegroundColor $green -NoNewline
    Write-Host " - Show logs for the Caddy container"
    Write-Host "  " -NoNewline
    Write-Host "logs api" -ForegroundColor $green -NoNewline
    Write-Host "   - Show logs for the WhatsApp API container"
    Write-Host "  " -NoNewline
    Write-Host "format-caddy" -ForegroundColor $green -NoNewline
    Write-Host " - Format the Caddyfile and restart Caddy"
    Write-Host "  " -NoNewline
    Write-Host "status" -ForegroundColor $green -NoNewline
    Write-Host "     - Check the status of all containers"
    Write-Host ""
}

# Check if Docker is available
function Check-Docker {
    try {
        $null = Get-Command docker -ErrorAction Stop
        return $true
    }
    catch {
        Write-Host "Error: Docker is not installed or not in PATH" -ForegroundColor $red
        return $false
    }
}

# Start containers
function Start-Containers {
    Write-Host "Starting containers..." -ForegroundColor $blue
    docker compose up -d
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Containers started successfully" -ForegroundColor $green
    }
    else {
        Write-Host "Failed to start containers" -ForegroundColor $red
    }
}

# Stop containers
function Stop-Containers {
    Write-Host "Stopping containers..." -ForegroundColor $blue
    docker compose down
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Containers stopped successfully" -ForegroundColor $green
    }
    else {
        Write-Host "Failed to stop containers" -ForegroundColor $red
    }
}

# Restart containers
function Restart-Containers {
    Write-Host "Restarting containers..." -ForegroundColor $blue
    docker compose restart
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Containers restarted successfully" -ForegroundColor $green
    }
    else {
        Write-Host "Failed to restart containers" -ForegroundColor $red
    }
}

# Rebuild containers
function Rebuild-Containers {
    Write-Host "Rebuilding and starting containers..." -ForegroundColor $blue
    docker compose down
    docker compose build --no-cache
    docker compose up -d
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Containers rebuilt and started successfully" -ForegroundColor $green
    }
    else {
        Write-Host "Failed to rebuild containers" -ForegroundColor $red
    }
}

# Show logs
function Show-Logs {
    param (
        [string]$Container
    )

    if ([string]::IsNullOrEmpty($Container)) {
        Write-Host "Showing logs for all containers..." -ForegroundColor $blue
        docker compose logs --tail=100 -f
    }
    elseif ($Container -eq "caddy") {
        Write-Host "Showing logs for Caddy container..." -ForegroundColor $blue
        docker compose logs --tail=100 -f caddy
    }
    elseif ($Container -eq "api") {
        Write-Host "Showing logs for WhatsApp API container..." -ForegroundColor $blue
        docker compose logs --tail=100 -f whatsapp-api
    }
    else {
        Write-Host "Unknown container: $Container" -ForegroundColor $red
        Print-Usage
    }
}

# Format Caddyfile
function Format-Caddyfile {
    Write-Host "Formatting Caddyfile..." -ForegroundColor $blue
    
    # Get current path for mounting
    $currentPath = (Get-Location).Path
    $mountPath = $currentPath -replace "\\", "/"
    
    # Format the Caddyfile
    docker run --rm -v "${mountPath}/Caddyfile:/etc/caddy/Caddyfile" caddy:2.7-alpine caddy fmt --overwrite /etc/caddy/Caddyfile
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Caddyfile formatted successfully" -ForegroundColor $green
        
        # Restart Caddy to apply changes
        Write-Host "Restarting Caddy container..." -ForegroundColor $blue
        docker compose restart caddy
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "Caddy container restarted successfully" -ForegroundColor $green
        }
        else {
            Write-Host "Failed to restart Caddy container" -ForegroundColor $red
        }
    }
    else {
        Write-Host "Failed to format Caddyfile" -ForegroundColor $red
    }
}

# Show container status
function Show-Status {
    Write-Host "Checking container status..." -ForegroundColor $blue
    docker compose ps
}

# Main function
function Main {
    param (
        [string]$Command,
        [string]$SubCommand
    )

    Print-Header

    if (-not (Check-Docker)) {
        exit 1
    }

    if ([string]::IsNullOrEmpty($Command)) {
        Print-Usage
        exit 0
    }

    switch ($Command) {
        "start" {
            Start-Containers
        }
        "stop" {
            Stop-Containers
        }
        "restart" {
            Restart-Containers
        }
        "rebuild" {
            Rebuild-Containers
        }
        "logs" {
            Show-Logs -Container $SubCommand
        }
        "format-caddy" {
            Format-Caddyfile
        }
        "status" {
            Show-Status
        }
        default {
            Write-Host "Unknown command: $Command" -ForegroundColor $red
            Print-Usage
            exit 1
        }
    }
}

# Run main function
Main -Command $args[0] -SubCommand $args[1] 