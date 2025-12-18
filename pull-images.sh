#!/bin/bash
# Script to pull Docker images one at a time to avoid rate limits

set -e

echo "Pulling Docker images one at a time to avoid rate limits..."
echo "This may take a few minutes..."

# Pull images with delays between each
echo "1. Pulling Redis..."
docker pull redis/redis-stack-server:7.4.0-v0
sleep 5

echo "2. Pulling Zookeeper..."
docker pull public.ecr.aws/bitnami/zookeeper:3.9 --platform linux/amd64
sleep 5

echo "3. Pulling Kafka..."
docker pull public.ecr.aws/bitnami/kafka:3.7 --platform linux/amd64
sleep 5

echo "4. Pulling MinIO..."
docker pull minio/minio:latest
sleep 5

echo "5. Pulling Trino..."
docker pull trinodb/trino:435
sleep 5

echo "6. Pulling Spark..."
docker pull public.ecr.aws/bitnami/spark:3.5.4 --platform linux/amd64
sleep 5

echo "✓ All images pulled successfully!"
echo "You can now run: docker compose up"

