#!/bin/bash
# Restart AirDash services (menu bar app and monitor)

echo "Restarting AirDash services..."

# Restart menu bar app
echo "→ Restarting menu bar app..."
launchctl unload ~/Library/LaunchAgents/com.airdash.plist 2>/dev/null
launchctl load ~/Library/LaunchAgents/com.airdash.plist
echo "  ✓ Menu bar app restarted"

# Restart monitor service
echo "→ Restarting monitor service..."
launchctl unload ~/Library/LaunchAgents/com.airdash.monitor.plist 2>/dev/null
launchctl load ~/Library/LaunchAgents/com.airdash.monitor.plist
echo "  ✓ Monitor service restarted"

echo ""
echo "Services status:"
launchctl list | grep airdash

echo ""
echo "✓ All services restarted successfully!"
