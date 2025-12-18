#!/bin/bash
# Quick test if Docker is responding

echo "Testing Docker connection..."
echo ""

# Try with a timeout using background process
docker ps > /tmp/docker-test.log 2>&1 &
DOCKER_PID=$!

# Wait max 5 seconds
sleep 5

if kill -0 $DOCKER_PID 2>/dev/null; then
    echo "✗ Docker is STUCK (docker ps still running after 5 seconds)"
    echo "Run: ./fix-stuck-docker.sh"
    kill $DOCKER_PID 2>/dev/null || true
    exit 1
else
    if [ -f /tmp/docker-test.log ]; then
        echo "✓ Docker is responding!"
        cat /tmp/docker-test.log
        exit 0
    else
        echo "✗ Docker test failed"
        exit 1
    fi
fi
