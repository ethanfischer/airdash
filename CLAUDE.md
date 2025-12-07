# AirDash Project Guide

## Project Overview

AirDash is an air quality monitoring system for macOS that:
1. **Menu Bar App** (`airdash`) - Displays real-time PM2.5 and CO2 in the Mac menu bar
2. **Monitor Service** (`airdash-monitor`) - Background service that sends iPhone push notifications via Pushover when air quality thresholds are exceeded

## Building

```bash
# Build menu bar app
cd /Users/work/Repos/airdash
go build -o airdash .

# Build monitor service
cd /Users/work/Repos/airdash/airdash-monitor
go build -o airdash-monitor .
```

## Configuration Files

Located in `~/.airdash/`:

### config.yaml (AirGradient API)
```yaml
token: YOUR_AIRGRADIENT_TOKEN
locationId: YOUR_LOCATION_ID
interval: 60
tempUnit: F
```

### monitor-config.yaml (Pushover alerts)
```yaml
pushoverUserKey: YOUR_PUSHOVER_USER_KEY
pushoverApiToken: YOUR_PUSHOVER_API_TOKEN
pm25ThresholdWarning: 10
pm25ThresholdUrgent: 50
pm25ThresholdEmergency: 100
co2Threshold: 1000
tvocThreshold: 400
checkInterval: 2
```

## macOS Launch Agents Setup

Launch Agents are plist files that tell macOS to run programs automatically when you log in.

### Location
```
~/Library/LaunchAgents/
```

Each user has their own LaunchAgents folder. Files here run when **that user** logs in.

### File 1: Menu Bar App (`com.airdash.plist`)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.airdash</string>
    <key>ProgramArguments</key>
    <array>
        <string>/path/to/airdash</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/airdash.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/airdash.error.log</string>
</dict>
</plist>
```

### File 2: Monitor Service (`com.airdash.monitor.plist`)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.airdash.monitor</string>
    <key>ProgramArguments</key>
    <array>
        <string>/path/to/airdash-monitor/airdash-monitor</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/airdash-monitor.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/airdash-monitor.error.log</string>
</dict>
</plist>
```

### Key Settings Explained

| Key | Purpose |
|-----|---------|
| `Label` | Unique identifier for the service |
| `ProgramArguments` | Path to the executable |
| `RunAtLoad` | Start when user logs in |
| `KeepAlive` | Restart if it crashes |
| `StandardOutPath` | Where stdout goes |
| `StandardErrorPath` | Where stderr goes |

## Setup for a New User

### Step 1: Create the plist files

```bash
# Create LaunchAgents directory if it doesn't exist
mkdir -p ~/Library/LaunchAgents
```

### Step 2: Create com.airdash.plist

Replace `/path/to/airdash` with your actual path:

```bash
cat > ~/Library/LaunchAgents/com.airdash.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.airdash</string>
    <key>ProgramArguments</key>
    <array>
        <string>/path/to/airdash</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/airdash.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/airdash.error.log</string>
</dict>
</plist>
EOF
```

### Step 3: Create com.airdash.monitor.plist

```bash
cat > ~/Library/LaunchAgents/com.airdash.monitor.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.airdash.monitor</string>
    <key>ProgramArguments</key>
    <array>
        <string>/path/to/airdash-monitor/airdash-monitor</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/airdash-monitor.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/airdash-monitor.error.log</string>
</dict>
</plist>
EOF
```

### Step 4: Create config files

```bash
mkdir -p ~/.airdash

# AirGradient config
cat > ~/.airdash/config.yaml << 'EOF'
token: YOUR_AIRGRADIENT_TOKEN
locationId: YOUR_LOCATION_ID
interval: 60
tempUnit: F
EOF

# Monitor config
cat > ~/.airdash/monitor-config.yaml << 'EOF'
pushoverUserKey: YOUR_PUSHOVER_USER_KEY
pushoverApiToken: YOUR_PUSHOVER_API_TOKEN
pm25ThresholdWarning: 10
pm25ThresholdUrgent: 50
pm25ThresholdEmergency: 100
co2Threshold: 1000
tvocThreshold: 400
checkInterval: 2
EOF
```

### Step 5: Load the services

```bash
launchctl load ~/Library/LaunchAgents/com.airdash.plist
launchctl load ~/Library/LaunchAgents/com.airdash.monitor.plist
```

### Step 6: Verify they're running

```bash
launchctl list | grep airdash
```

## Managing Services

```bash
# Stop services
launchctl unload ~/Library/LaunchAgents/com.airdash.plist
launchctl unload ~/Library/LaunchAgents/com.airdash.monitor.plist

# Start services
launchctl load ~/Library/LaunchAgents/com.airdash.plist
launchctl load ~/Library/LaunchAgents/com.airdash.monitor.plist

# Restart both (use the script)
./restart-services.sh

# Check status
launchctl list | grep airdash

# View logs
tail -f /tmp/airdash.error.log
tail -f /tmp/airdash-monitor.error.log
```

## Alert Levels

### PM2.5 (Tiered)
- **Warning** (default 10 μg/m³) - Moderate air quality
- **Urgent** (default 50 μg/m³) - Unhealthy for sensitive groups
- **Emergency** (default 100 μg/m³) - Unhealthy air, repeating alerts until acknowledged

### CO2
- Single threshold (default 1000 ppm)

### TVOC
- Single threshold (default 400 ppb)

## Testing

```bash
cd /Users/work/Repos/airdash/airdash-monitor
go test -v
```

## Troubleshooting

### Multi-User Systems: Log File Conflicts

**Problem:** On systems with multiple users, LaunchAgents may fail with exit code 78 if log files in `/tmp/` are owned by a different user.

**Symptoms:**
- `launchctl list | grep airdash` shows `-` for PID and exit code `78`
- Services work when run directly from terminal but fail via launchctl
- Log files in `/tmp/` owned by another user (check with `ls -la /tmp/airdash*.log`)

**Solution:** Use user-specific log file names in your plist files:

```xml
<key>StandardOutPath</key>
<string>/tmp/airdash-USERNAME.log</string>
<key>StandardErrorPath</key>
<string>/tmp/airdash-USERNAME.error.log</string>
```

### Menu Bar App (GUI) Requirements

The menu bar app requires access to the macOS GUI session. Add `LimitLoadToSessionType` to ensure it only runs in graphical sessions:

```xml
<key>LimitLoadToSessionType</key>
<string>Aqua</string>
```

### Complete Working Plist Examples

**com.airdash.plist (Menu Bar App):**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.airdash</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/USERNAME/Repos/airdash/airdash</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>LimitLoadToSessionType</key>
    <string>Aqua</string>
    <key>StandardOutPath</key>
    <string>/tmp/airdash-USERNAME.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/airdash-USERNAME.error.log</string>
</dict>
</plist>
```

**com.airdash.monitor.plist (Background Service):**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.airdash.monitor</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/USERNAME/Repos/airdash/airdash-monitor/airdash-monitor</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/airdash-monitor-USERNAME.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/airdash-monitor-USERNAME.error.log</string>
</dict>
</plist>
```

### Verifying Services Are Running

```bash
# Check launchctl status (should show PID and exit code 0)
launchctl list | grep airdash

# Example of healthy output:
# 7351    0    com.airdash
# 7352    0    com.airdash.monitor

# Example of failed output (exit code 78, no PID):
# -    78    com.airdash

# Check running processes
ps aux | grep airdash | grep -v grep

# Check logs for errors
tail -f /tmp/airdash-USERNAME.error.log
tail -f /tmp/airdash-monitor-USERNAME.error.log
```

### Validate Plist Syntax

```bash
plutil -lint ~/Library/LaunchAgents/com.airdash.plist
plutil -lint ~/Library/LaunchAgents/com.airdash.monitor.plist
```
