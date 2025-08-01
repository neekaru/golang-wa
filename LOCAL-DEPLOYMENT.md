# Local Deployment Guide (Without Docker)

This guide explains how to run the WhatsApp API directly on your server without Docker.

## Prerequisites

- Go 1.19+ installed
- Git installed
- Nginx installed (optional, for reverse proxy)

## Installation Steps

### 1. Install Go

```bash
# Download and install Go (Linux/Ubuntu)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Add Go to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

### 2. Clone and Build

```bash
# Clone the repository
git clone <your-repo-url>
cd whatsapp-api

# Download dependencies
go mod download

# Build the application
go build -o whatsapp-api main.go

# Make executable
chmod +x whatsapp-api

# Create necessary directories
mkdir -p data logs
```

### 3. Run the Application

```bash
# Run directly
./whatsapp-api

# Or run in background
nohup ./whatsapp-api > logs/app.log 2>&1 &
```

The API will be available at `http://localhost:3000`

### 4. Create Systemd Service (Optional)

For production deployments, create a systemd service:

```bash
sudo nano /etc/systemd/system/whatsapp-api.service
```

Add the following content:

```ini
[Unit]
Description=WhatsApp API Service
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/path/to/your/whatsapp-api
ExecStart=/path/to/your/whatsapp-api/whatsapp-api
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Environment variables
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl enable whatsapp-api
sudo systemctl start whatsapp-api
sudo systemctl status whatsapp-api
```

## Configuration

### Environment Variables

You can set the following environment variables:

- `GIN_MODE=release` - Run in production mode
- `PORT=8080` - Change the port (default: 8080)

### Custom Port

To run on a different port, modify `internal/config/config.go`:

```go
func NewConfig() *Config {
    return &Config{
        ServerPort: "3000", // Change from "8080" to your desired port
        DataDir:    "data",
    }
}
```

Then rebuild:

```bash
go build -o whatsapp-api main.go
```

## Nginx Setup (Recommended for Production)

For production deployments, use nginx as a reverse proxy. See [NGINX-SETUP.md](NGINX-SETUP.md) for detailed instructions.

## API Access

Once running, your API will be available at:

- `http://localhost:3000/` - Main API endpoint
- `http://localhost:3000/health` - Health check
- `http://localhost:3000/wa/qr-image?user=test_user` - QR code generation

For all available endpoints, see [docs.md](docs.md).

## Monitoring

### View Logs

```bash
# If running with systemd
sudo journalctl -u whatsapp-api -f

# If running manually with nohup
tail -f logs/app.log
```

### Health Check

```bash
curl http://localhost:3000/health
```

## Troubleshooting

1. **Port already in use**: Check if another service is using port 3000
   ```bash
   sudo netstat -tlnp | grep 3000
   ```

2. **Permission denied**: Ensure the user has proper permissions to the application directory

3. **Build errors**: Make sure Go is properly installed and `go mod download` completed successfully

4. **Service won't start**: Check logs with `sudo journalctl -u whatsapp-api`