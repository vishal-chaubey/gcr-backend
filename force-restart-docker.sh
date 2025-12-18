#!/bin/bash
# Force restart Docker Desktop when docker ps is stuck

set -e

echo "=== Force Restart Docker Desktop ==="
echo ""

echo "Step 1: Killing all Docker processes..."
# Kill Docker Desktop processes
pkill -9 "Docker Desktop" 2>/dev/null || true
pkill -9 "com.docker.backend" 2>/dev/null || true
pkill -9 "com.docker.virtualization" 2>/dev/null || true
pkill -9 "com.docker.dev-envs" 2>/dev/null || true
sleep 3

echo "Step 2: Cleaning up Docker socket..."
# Remove potentially corrupted socket
rm -f ~/.docker/run/docker.sock 2>/dev/null || true
rm -f /var/run/docker.sock 2>/dev/null || true

echo "Step 3: Waiting 5 seconds..."
sleep 5

echo "Step 4: Starting Docker Desktop..."
open -a Docker

echo ""
echo "✓ Docker Desktop is starting..."
echo ""
echo "IMPORTANT: Wait 60-90 seconds for Docker Desktop to fully initialize"
echo "You'll see the Docker icon in the menu bar turn green when ready"
echo ""
echo "Then test with:"
echo "  timeout 5 docker ps"
echo ""
echo "If that works, run:"
echo "  docker compose up"

