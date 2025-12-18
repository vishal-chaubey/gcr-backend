#!/bin/bash
# Script to fix Docker Desktop containerd issues

echo "=== Docker Desktop Troubleshooting ==="
echo ""

echo "Step 1: Checking Docker Desktop status..."
if pgrep -f "Docker Desktop" > /dev/null; then
    echo "✓ Docker Desktop is running"
else
    echo "✗ Docker Desktop is not running"
    echo "Please start Docker Desktop from Applications"
    exit 1
fi

echo ""
echo "Step 2: Attempting to restart Docker Desktop..."
echo "This will quit and restart Docker Desktop..."

# Quit Docker Desktop
osascript -e 'quit app "Docker"'
sleep 5

# Wait a bit
echo "Waiting 10 seconds..."
sleep 10

# Start Docker Desktop
open -a Docker
echo "Docker Desktop is starting..."
echo "Please wait 30-60 seconds for Docker Desktop to fully start"
echo ""

echo "Step 3: After Docker Desktop starts, run:"
echo "  docker ps"
echo ""
echo "If 'docker ps' works, then run:"
echo "  docker compose up"
echo ""

echo "=== Alternative: If restart doesn't work ==="
echo "1. Quit Docker Desktop completely"
echo "2. Run: rm -rf ~/Library/Containers/com.docker.docker/Data/vms/0/data/Docker.raw"
echo "3. Restart Docker Desktop (it will recreate the VM)"
echo ""
echo "WARNING: This will delete all Docker containers and volumes!"

