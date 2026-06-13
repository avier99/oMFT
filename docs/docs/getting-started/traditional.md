---
sidebar_position: 4
title: Traditional Installation
---

# Traditional Installation Guide

This guide covers installing oMFT directly on your system without using Docker. This approach is useful for environments where containers aren't available or when you need more direct control over the installation.

## System Requirements

- **Operating System**: Linux, macOS, or Windows
- **Go**: Version 1.20 or later
- **Node.js**: Version 18 or later
- **Build Tools**: gcc and related build tools (for SQLite compilation)

## Prerequisites Installation

### On Debian/Ubuntu Linux

```bash
# Install Go
wget https://go.dev/dl/go1.20.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.20.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
source ~/.profile

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# Install build tools
sudo apt-get install -y build-essential
```

### On macOS

```bash
# Using Homebrew
brew install go
brew install node
brew install gcc
```

### On Windows

1. Install Go from [https://golang.org/dl/](https://golang.org/dl/)
2. Install Node.js from [https://nodejs.org/](https://nodejs.org/)
3. Install Build Tools for Visual Studio

## Building oMFT from Source

1. Clone the repository:

```bash
git clone https://github.com/avier99/oMFT.git
cd oMFT
```

2. Install Node.js dependencies and build the frontend:

```bash
npm install
npm run build
```

3. Compile the Go application:

```bash
go build -o omft main.go
```

## Installation Options

### Option 1: Run Directly

After building, you can run the application directly:

```bash
./omft
```

### Option 2: Install as a System Service

#### On Linux (systemd)

Create a systemd service file:

```bash
sudo nano /etc/systemd/system/omft.service
```

Add the following content:

```ini
[Unit]
Description=oMFT - Go Managed File Transfer
After=network.target

[Service]
Type=simple
User=omft
Group=omft
WorkingDirectory=/opt/omft
ExecStart=/opt/omft/omft
Restart=on-failure
RestartSec=5s
Environment="PORT=8080"
Environment="DATA_DIR=/var/lib/omft/data"
Environment="BACKUP_DIR=/var/lib/omft/backups"
Environment="LOGS_DIR=/var/log/omft"

[Install]
WantedBy=multi-user.target
```

Create a dedicated user and set up directories:

```bash
# Create user
sudo useradd -r -s /bin/false omft

# Create directories
sudo mkdir -p /opt/omft /var/lib/omft/data /var/lib/omft/backups /var/log/omft

# Copy application
sudo cp -r * /opt/omft/

# Set permissions
sudo chown -R omft:omft /opt/omft /var/lib/omft /var/log/omft
```

Enable and start the service:

```bash
sudo systemctl enable omft
sudo systemctl start omft
```

#### On macOS (launchd)

Create a launchd plist file:

```bash
sudo nano /Library/LaunchDaemons/com.omft.plist
```

Add the following content:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.omft</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/omft/omft</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>WorkingDirectory</key>
    <string>/usr/local/omft</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PORT</key>
        <string>8080</string>
        <key>DATA_DIR</key>
        <string>/var/lib/omft/data</string>
        <key>BACKUP_DIR</key>
        <string>/var/lib/omft/backups</string>
        <key>LOGS_DIR</key>
        <string>/var/log/omft</string>
    </dict>
</dict>
</plist>
```

Set up directories and install:

```bash
# Create directories
sudo mkdir -p /usr/local/omft /var/lib/omft/data /var/lib/omft/backups /var/log/omft

# Copy application
sudo cp -r * /usr/local/omft/

# Set permissions
sudo chown -R $(whoami):staff /usr/local/omft /var/lib/omft /var/log/omft

# Load service
sudo launchctl load /Library/LaunchDaemons/com.omft.plist
```

#### On Windows (Windows Service)

1. Install [NSSM (Non-Sucking Service Manager)](https://nssm.cc/download)
2. Open Command Prompt as Administrator
3. Create the service:

```bat
nssm install oMFT C:\path\to\omft.exe
nssm set oMFT AppDirectory C:\path\to\omft\directory
nssm set oMFT AppEnvironmentExtra PORT=8080 DATA_DIR=C:\ProgramData\oMFT\data BACKUP_DIR=C:\ProgramData\oMFT\backups LOGS_DIR=C:\ProgramData\oMFT\logs
nssm start oMFT
```

## Configuration

### Environment Variables

Create a `.env` file in the application directory or set system environment variables:

```
PORT=8080
DATA_DIR=/var/lib/omft/data
BACKUP_DIR=/var/lib/omft/backups
LOGS_DIR=/var/log/omft
BASE_URL=http://localhost:8080
EMAIL_ENABLED=false
JWT_SECRET=your-secret-key
ENCRYPT_KEY=32-character-encryption-key
```

### Web Server Setup

For production use, it's recommended to run oMFT behind a web server like Nginx:

#### Nginx Configuration

```nginx
server {
    listen 80;
    server_name your-gomft-server.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Updating oMFT

To update a traditionally installed oMFT:

1. Stop the service:
   ```bash
   sudo systemctl stop omft  # For Linux
   sudo launchctl unload /Library/LaunchDaemons/com.omft.plist  # For macOS
   nssm stop oMFT  # For Windows
   ```

2. Back up your data:
   ```bash
   cp -r /var/lib/omft/data /var/lib/omft/data.backup
   ```

3. Get the latest code:
   ```bash
   cd /path/to/oMFT/source
   git pull
   ```

4. Rebuild:
   ```bash
   npm install
   npm run build
   go build -o omft main.go
   ```

5. Update the installation:
   ```bash
   sudo cp omft /opt/omft/  # For Linux
   sudo cp omft /usr/local/omft/  # For macOS
   copy omft.exe C:\path\to\omft.exe  # For Windows
   ```

6. Restart the service:
   ```bash
   sudo systemctl start omft  # For Linux
   sudo launchctl load /Library/LaunchDaemons/com.omft.plist  # For macOS
   nssm start oMFT  # For Windows
   ```

## Troubleshooting

### Common Issues

1. **Permission Errors**:
   - Check that the user running oMFT has write permissions to the data, backup, and logs directories.

2. **Database Errors**:
   - Ensure the SQLite database path is writeable.
   - Check database integrity: `sqlite3 /var/lib/omft/data/gomft.db "PRAGMA integrity_check;"`

3. **Port Already in Use**:
   - Change the port in the configuration.
   - Check what's using port 8080: `sudo lsof -i :8080`

4. **Missing Dependencies**:
   - Make sure all required Go and Node.js dependencies are installed.

### Viewing Logs

- **Application Logs**: Check `/var/log/omft/` or your configured logs directory
- **System Service Logs**:
  ```bash
  # For Linux
  journalctl -u omft
  
  # For macOS
  log show --predicate 'senderImagePath contains "gomft"'
  
  # For Windows
  Get-EventLog -LogName Application -Source oMFT
  ```

For more help, refer to the [GitHub repository](https://github.com/avier99/oMFT) or open an issue. 