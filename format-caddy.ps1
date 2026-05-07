# Colors for better output
$green = [System.ConsoleColor]::Green
$blue = [System.ConsoleColor]::Blue
$red = [System.ConsoleColor]::Red

Write-Host "Formatting Caddyfile..." -ForegroundColor $blue

# Check if Docker is available
try {
    $null = Get-Command docker -ErrorAction Stop
}
catch {
    Write-Host "Error: Docker is not installed or not in PATH" -ForegroundColor $red
    exit 1
}

# Format the Caddyfile using the Caddy container
$currentPath = (Get-Location).Path
$mountPath = $currentPath -replace "\\", "/"

Write-Host "Mounting path: $mountPath" -ForegroundColor $blue
docker run --rm -v "${mountPath}/Caddyfile:/etc/caddy/Caddyfile" caddy:2.7-alpine caddy fmt --overwrite /etc/caddy/Caddyfile

if ($LASTEXITCODE -eq 0) {
    Write-Host "Caddyfile formatted successfully" -ForegroundColor $green
}
else {
    Write-Host "Failed to format Caddyfile" -ForegroundColor $red
    exit 1
}

Write-Host "Restarting Caddy container..." -ForegroundColor $blue
docker compose restart caddy

if ($LASTEXITCODE -eq 0) {
    Write-Host "Caddy container restarted successfully" -ForegroundColor $green
}
else {
    Write-Host "Failed to restart Caddy container" -ForegroundColor $red
    exit 1
} 