#!/bin/bash
# Fix stuck Docker Desktop when docker ps hangs

echo "=== Fixing Stuck Docker Desktop ==="
echo ""

# Check if Docker Desktop is running
if ! pgrep -f "Docker Desktop" > /dev/null; then
    echo "Docker Desktop is not running. Starting it..."
    open -a Docker
    echo "Wait 60 seconds for Docker to start, then run this script again."
    exit 0
fi

echo "Step 1: Force quitting Docker Desktop..."
osascript -e 'quit app "Docker"' 2>/dev/null || true
sleep 3

echo "Step 2: Killing stuck Docker processes..."
pkill -9 "com.docker.backend" 2>/dev/null || true
pkill -9 "com.docker.virtualization" 2>/dev/null || true
pkill -9 "com.docker.dev-envs" 2>/dev/null || true
pkill -9 "containerd" 2>/dev/null || true
sleep 2

echo "Step 3: Cleaning Docker sockets..."
rm -f ~/.docker/run/docker.sock 2>/dev/null || true
rm -f /var/run/docker.sock 2>/dev/null || true

echo "Step 4: Waiting 10 seconds..."
sleep 10

echo "Step 5: Starting Docker Desktop..."
open -a Docker

echo ""
echo "✓ Docker Desktop is restarting..."
echo ""
echo "CRITICAL: Wait 60-90 seconds for Docker Desktop to fully initialize"
echo "Watch the Docker icon in menu bar - it should turn green when ready"
echo ""
echo "To test if Docker is working:"
echo "  1. Wait until menu bar icon is green"
echo "  2. Run: docker ps"
echo "  3. If it still hangs, run this script again with --reset flag"
echo ""
echo "If still stuck after restart, run:"
echo "  ./fix-stuck-docker.sh --reset"
echo ""
echo "WARNING: --reset will delete all containers and volumes!"

# Handle --reset flag
if [ "$1" == "--reset" ]; then
    echo ""
    echo "=== RESET MODE: Deleting Docker VM ==="
    read -p "This will delete ALL containers and volumes. Continue? (yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
        echo "Cancelled."
        exit 0
    fi
    
    echo "Quitting Docker Desktop..."
    osascript -e 'quit app "Docker"' 2>/dev/null || true
    sleep 5
    
    echo "Deleting Docker VM file..."
    rm -rf ~/Library/Containers/com.docker.docker/Data/vms/0/data/Docker.raw
    echo "✓ Docker VM deleted"
    
    echo "Starting Docker Desktop (will recreate VM)..."
    open -a Docker
    echo ""
    echo "Wait 2-3 minutes for Docker to recreate the VM"
    echo "Then test with: docker ps"
fi

