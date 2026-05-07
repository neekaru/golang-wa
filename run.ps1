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
    Write-Host "logs api" -ForegroundColor $green -NoNewline
    Write-Host " - Show logs for the WhatsApp API container"
    Write-Host "  " -NoNewline
    Write-Host "status" -ForegroundColor $green -NoNewline
    Write-Host "     - Check the status of all containers"
    Write-Host "  " -NoNewline
    Write-Host "health" -ForegroundColor $green -NoNewline
    Write-Host "     - Get basic health status of API"
    Write-Host "  " -NoNewline
    Write-Host "health-detailed" -ForegroundColor $green -NoNewline
    Write-Host " - Get detailed health status of API"
    Write-Host "  " -NoNewline
    Write-Host "wa-status [user]" -ForegroundColor $green -NoNewline
    Write-Host " - Check WhatsApp connection status for a user"
    Write-Host "  " -NoNewline
    Write-Host "wa-restart [user]" -ForegroundColor $green -NoNewline
    Write-Host " - Restart a WhatsApp session"
    Write-Host "  " -NoNewline
    Write-Host "init-dirs" -ForegroundColor $green -NoNewline
    Write-Host "   - Create data and logs directories"
    Write-Host "  " -NoNewline
    Write-Host "app-logs" -ForegroundColor $green -NoNewline
    Write-Host "   - View the WhatsApp application logs"
    Write-Host "  " -NoNewline
    Write-Host "app-logs [user]" -ForegroundColor $green -NoNewline
    Write-Host " - Filter logs for a specific user"
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
    # Ensure required directories exist before starting
    Initialize-Directories
    
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
    # Ensure required directories exist before rebuilding
    Initialize-Directories
    
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


# Show container status
function Show-Status {
    Write-Host "Checking container status..." -ForegroundColor $blue
    docker compose ps
}

# Check basic health status
function Check-Health {
    Write-Host "Checking API health status..." -ForegroundColor $blue
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/" -Method Get
        Write-Host ""
        Write-Host "Status: " -NoNewline
        Write-Host $response.status -ForegroundColor $green
        Write-Host "Uptime: " -NoNewline
        Write-Host $response.uptime -ForegroundColor $green
        Write-Host "Session Count: " -NoNewline
        Write-Host $response.session_count -ForegroundColor $green
        Write-Host "Version: " -NoNewline
        Write-Host $response.version -ForegroundColor $green
    }
    catch {
        Write-Host "Failed to get health status: $_" -ForegroundColor $red
    }
}

# Check detailed health status
function Check-Health-Detailed {
    Write-Host "Checking detailed API health status..." -ForegroundColor $blue
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/health" -Method Get
        Write-Host ""
        Write-Host "Status: " -NoNewline
        Write-Host $response.status -ForegroundColor $green
        Write-Host "Uptime: " -NoNewline
        Write-Host $response.uptime -ForegroundColor $green
        Write-Host "Total Sessions: " -NoNewline
        Write-Host $response.total_sessions -ForegroundColor $green
        Write-Host "Active Sessions: " -NoNewline
        Write-Host $response.active_sessions -ForegroundColor $green
        Write-Host "Timestamp: " -NoNewline
        Write-Host $response.timestamp -ForegroundColor $green
    }
    catch {
        Write-Host "Failed to get detailed health status: $_" -ForegroundColor $red
    }
}

# Check WhatsApp session status
function Check-WA-Status {
    param (
        [string]$User
    )

    if ([string]::IsNullOrEmpty($User)) {
        Write-Host "Error: User parameter is required" -ForegroundColor $red
        return
    }

    Write-Host "Checking WhatsApp status for user $User..." -ForegroundColor $blue
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/wa/status?user=$User" -Method Get
        Write-Host ""
        Write-Host "User: " -NoNewline
        Write-Host $response.user -ForegroundColor $green
        Write-Host "Logged In: " -NoNewline
        if ($response.logged_in) {
            Write-Host "Yes" -ForegroundColor $green
        } else {
            Write-Host "No" -ForegroundColor $yellow
        }
        Write-Host "Connected: " -NoNewline
        if ($response.connected) {
            Write-Host "Yes" -ForegroundColor $green
        } else {
            Write-Host "No" -ForegroundColor $yellow
        }
    }
    catch {
        Write-Host "Failed to get WhatsApp status: $_" -ForegroundColor $red
    }
}

# Restart WhatsApp session
function Restart-WA-Session {
    param (
        [string]$User
    )

    if ([string]::IsNullOrEmpty($User)) {
        Write-Host "Error: User parameter is required" -ForegroundColor $red
        return
    }

    Write-Host "Restarting WhatsApp session for user $User..." -ForegroundColor $blue
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/wa/restart?user=$User" -Method Post
        Write-Host "Session restart result: " -NoNewline
        Write-Host $response.msg -ForegroundColor $green
    }
    catch {
        Write-Host "Failed to restart WhatsApp session: $_" -ForegroundColor $red
    }
}

# Initialize data and logs directories
function Initialize-Directories {
    Write-Host "Creating necessary directories..." -ForegroundColor $blue
    
    # Create data directory if it doesn't exist
    if (-not (Test-Path -Path "data")) {
        New-Item -ItemType Directory -Path "data" | Out-Null
        Write-Host "Created data directory" -ForegroundColor $green
    } else {
        Write-Host "Data directory already exists" -ForegroundColor $green
    }
    
    # Create logs directory if it doesn't exist
    if (-not (Test-Path -Path "logs")) {
        New-Item -ItemType Directory -Path "logs" | Out-Null
        Write-Host "Created logs directory" -ForegroundColor $green
    } else {
        Write-Host "Logs directory already exists" -ForegroundColor $green
    }
    
    # Ensure proper permissions
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Directories initialized successfully" -ForegroundColor $green
    }
}

# View WhatsApp application logs
function View-App-Logs {
    param (
        [string]$User
    )
    
    Write-Host "Checking WhatsApp application logs..." -ForegroundColor $blue
    
    # Check if logs directory exists
    if (-not (Test-Path -Path "logs")) {
        Write-Host "Logs directory not found. Creating it now..." -ForegroundColor $yellow
        New-Item -ItemType Directory -Path "logs" | Out-Null
        Write-Host "No log files found yet. Start the application first." -ForegroundColor $yellow
        return
    }
    
    # Get log files
    $logFiles = Get-ChildItem -Path "logs" -Filter "whatsapp-api-*.log"
    
    if ($logFiles.Count -eq 0) {
        Write-Host "No log files found in the logs directory" -ForegroundColor $yellow
        return
    }
    
    # Get the most recent log file
    $latestLog = $logFiles | Sort-Object LastWriteTime -Descending | Select-Object -First 1
    
    Write-Host "Viewing latest log file: $($latestLog.Name)" -ForegroundColor $green
    
    if ([string]::IsNullOrEmpty($User)) {
        # Show the entire log file
        Get-Content -Path $latestLog.FullName -Tail 100 -Wait
    } else {
        # Filter for a specific user
        Write-Host "Filtering logs for user: $User" -ForegroundColor $blue
        Get-Content -Path $latestLog.FullName -Wait | Where-Object { $_ -like "*$User*" }
    }
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
        "status" {
            Show-Status
        }
        "health" {
            Check-Health
        }
        "health-detailed" {
            Check-Health-Detailed
        }
        "wa-status" {
            Check-WA-Status -User $SubCommand
        }
        "wa-restart" {
            Restart-WA-Session -User $SubCommand
        }
        "init-dirs" {
            Initialize-Directories
        }
        "app-logs" {
            View-App-Logs -User $SubCommand
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