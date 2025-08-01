# Nginx Setup for WhatsApp API

This guide explains how to deploy the WhatsApp API with nginx as a reverse proxy on a local server (without Docker).

## Prerequisites

- Ubuntu/Debian server with nginx installed
- Go 1.19+ installed
- SSL certificate (Let's Encrypt recommended)
- Domain name pointing to your server

## Installation Steps

### 1. Install Dependencies

```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Install nginx
sudo apt install nginx -y

# Install Go (if not already installed)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Git (if not already installed)
sudo apt install git -y
```

### 2. Clone and Build the Application

```bash
# Clone the repository
git clone <your-repo-url>
cd whatsapp-api

# Build the application
go mod download
go build -o whatsapp-api main.go

# Create necessary directories
mkdir -p data logs

# Make the binary executable
chmod +x whatsapp-api
```

### 3. Create Systemd Service

Create a systemd service file to manage the WhatsApp API:

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

# Environment variables (optional)
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
```

Replace `/path/to/your/whatsapp-api` with the actual path to your application.

### 4. Configure Nginx

Create nginx configuration file:

```bash
sudo nano /etc/nginx/sites-available/whatsapp-api
```

Add the following configuration (replace `your-domain.com` with your actual domain):

```nginx
server {
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket support (if needed for future features)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        # Timeout settings
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        
        # Buffer settings for better performance
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
        proxy_busy_buffers_size 8k;
    }

    # Security headers
    add_header X-Frame-Options "DENY" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;

    listen 443 ssl; # managed by Certbot
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem; # managed by Certbot
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem; # managed by Certbot
    include /etc/letsencrypt/options-ssl-nginx.conf; # managed by Certbot
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem; # managed by Certbot
}

# HTTP to HTTPS redirect
server {
    if ($host = your-domain.com) {
        return 301 https://$host$request_uri;
    } # managed by Certbot

    listen 80;
    server_name your-domain.com;
    return 404; # managed by Certbot
}
```

Enable the site:

```bash
# Enable the site
sudo ln -s /etc/nginx/sites-available/whatsapp-api /etc/nginx/sites-enabled/

# Test nginx configuration
sudo nginx -t

# Reload nginx
sudo systemctl reload nginx
```

### 5. Setup SSL with Let's Encrypt

```bash
# Install Certbot
sudo apt install certbot python3-certbot-nginx -y

# Obtain SSL certificate
sudo certbot --nginx -d your-domain.com

# Test automatic renewal
sudo certbot renew --dry-run
```

### 6. Start Services

```bash
# Enable and start the WhatsApp API service
sudo systemctl enable whatsapp-api
sudo systemctl start whatsapp-api

# Check service status
sudo systemctl status whatsapp-api

# Enable and start nginx
sudo systemctl enable nginx
sudo systemctl start nginx
```

## Configuration

### Environment Variables

You can set environment variables in the systemd service file or create a `.env` file in your application directory:

```bash
# Optional: Create environment file
nano /path/to/your/whatsapp-api/.env
```

Add any required environment variables:

```
GIN_MODE=release
DATA_DIR=./data
LOG_DIR=./logs
```

### Firewall Configuration

```bash
# Allow HTTP and HTTPS traffic
sudo ufw allow 'Nginx Full'

# Allow SSH (if not already allowed)
sudo ufw allow ssh

# Enable firewall
sudo ufw enable
```

## Monitoring and Logs

### View Application Logs

```bash
# View real-time logs
sudo journalctl -u whatsapp-api -f

# View recent logs
sudo journalctl -u whatsapp-api -n 100
```

### View Nginx Logs

```bash
# Access logs
sudo tail -f /var/log/nginx/access.log

# Error logs
sudo tail -f /var/log/nginx/error.log
```

### Application Health Check

```bash
# Test the API directly
curl http://localhost:8080/health

# Test through nginx
curl https://your-domain.com/health
```

## Troubleshooting

### Common Issues

1. **Service won't start**: Check logs with `sudo journalctl -u whatsapp-api`
2. **Permission denied**: Ensure the user has proper permissions to the application directory
3. **Port already in use**: Check if another service is using port 8080: `sudo netstat -tlnp | grep 8080`
4. **Nginx configuration errors**: Test with `sudo nginx -t`

### Restart Services

```bash
# Restart WhatsApp API
sudo systemctl restart whatsapp-api

# Restart nginx
sudo systemctl restart nginx
```

## Security Considerations

1. **Firewall**: Only allow necessary ports (80, 443, SSH)
2. **Updates**: Keep the system and dependencies updated
3. **Monitoring**: Set up monitoring for the service
4. **Backups**: Regular backups of the `data` directory
5. **Rate Limiting**: Consider implementing rate limiting in nginx if needed

## API Access

Once deployed, your API will be available at:
- `https://your-domain.com/` - Main API endpoint
- `https://your-domain.com/health` - Health check endpoint

All API endpoints documented in [docs.md](docs.md) will be accessible through your domain instead of `localhost:8080`.