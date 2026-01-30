# Configuration Guide

OpenPrintHub supports configuration through command-line arguments.

## Command-Line Arguments

| Argument | Default | Description |
|----------|---------|-------------|
| `-port` | 16800 | HTTP server port |
| `-cors` | `*` | Allowed CORS origins (comma-separated) |
| `-version` | - | Show version and exit |

### Examples

```bash
# Use default configuration
./oph

# Custom port
./oph -port 8080

# Restrict CORS origins
./oph -cors "https://app.example.com,https://admin.example.com"

# Show version
./oph -version
```

---

## Running as a System Service

### macOS (launchd)

Create file `~/Library/LaunchAgents/com.openprinthub.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.openprinthub</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/oph</string>
        <string>-port</string>
        <string>16800</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/openprinthub.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/openprinthub.error.log</string>
</dict>
</plist>
```

Enable the service:

```bash
# Copy binary to /usr/local/bin
sudo cp oph /usr/local/bin/

# Load the service
launchctl load ~/Library/LaunchAgents/com.openprinthub.plist

# Check status
launchctl list | grep openprinthub

# Stop the service
launchctl unload ~/Library/LaunchAgents/com.openprinthub.plist
```

### Windows (Service)

Use [NSSM](https://nssm.cc/) to install OpenPrintHub as a Windows service:

```powershell
# Download NSSM
# https://nssm.cc/download

# Install service
nssm install OpenPrintHub "C:\Program Files\OpenPrintHub\oph.exe"
nssm set OpenPrintHub AppParameters "-port 16800"
nssm set OpenPrintHub Start SERVICE_AUTO_START

# Start service
nssm start OpenPrintHub

# Stop service
nssm stop OpenPrintHub

# Remove service
nssm remove OpenPrintHub confirm
```

Or use the `sc` command:

```powershell
# Create service
sc create OpenPrintHub binPath= "C:\Program Files\OpenPrintHub\oph.exe -port 16800" start= auto

# Start
sc start OpenPrintHub

# Stop
sc stop OpenPrintHub

# Delete
sc delete OpenPrintHub
```

---

## CORS Configuration

### Development Environment

Use wildcard to allow all origins during development:

```bash
./oph -cors "*"
```

### Production Environment

In production, explicitly specify allowed origins:

```bash
./oph -cors "https://your-saas-app.com,https://admin.your-saas-app.com"
```

### Multiple Origins

Use commas to separate multiple origins:

```bash
./oph -cors "http://localhost:3000,http://localhost:8080,https://prod.example.com"
```

---

## Network Configuration

### LAN Access

OpenPrintHub listens on all network interfaces (`0.0.0.0`) by default, allowing access from other devices on the LAN.

Find your local IP:

```bash
# macOS
ipconfig getifaddr en0

# Windows
ipconfig
```

Access from other devices:

```
http://192.168.1.100:16800
```

### Firewall Configuration

Ensure your firewall allows inbound connections on port 16800.

**Windows Firewall**:

```powershell
netsh advfirewall firewall add rule name="OpenPrintHub" dir=in action=allow protocol=TCP localport=16800
```

**macOS Firewall**:

System Preferences → Security & Privacy → Firewall → Allow incoming connections for the app

---

## Troubleshooting

### Port Already in Use

```bash
# Check port usage (macOS/Linux)
lsof -i :16800

# Check port usage (Windows)
netstat -ano | findstr :16800
```

Use a different port:

```bash
./oph -port 12346
```

### Printer Not Showing

1. Verify the printer is connected and drivers are installed
2. Check the system printer list
3. Restart OpenPrintHub

**macOS Check**:

```bash
lpstat -p
```

**Windows Check**:

```powershell
Get-Printer | Select-Object Name, PrinterStatus
```

### Viewing Logs

OpenPrintHub outputs logs to standard output. To save logs:

```bash
./oph 2>&1 | tee openprinthub.log
```
