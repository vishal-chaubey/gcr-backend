#!/bin/bash
# Quick monitoring script for GCR Backend
# Usage: ./monitor.sh [api|kafka|redis|all]

SERVICE=${1:-api}

case $SERVICE in
  api)
    echo "=== Monitoring API Logs (Schema Validation & Processing) ==="
    echo "Press Ctrl+C to stop"
    echo ""
    docker compose logs -f api
    ;;
  kafka)
    echo "=== Monitoring Kafka Logs ==="
    echo "Press Ctrl+C to stop"
    echo ""
    docker compose logs -f kafka
    ;;
  redis)
    echo "=== Monitoring Redis Logs ==="
    echo "Press Ctrl+C to stop"
    echo ""
    docker compose logs -f redis
    ;;
  all)
    echo "=== Monitoring All Services ==="
    echo "Press Ctrl+C to stop"
    echo ""
    docker compose logs -f
    ;;
  validation)
    echo "=== Monitoring Schema Validation (Provider & Item) ==="
    echo "Press Ctrl+C to stop"
    echo ""
    docker compose logs -f api | grep -E "(SchemaGate|Provider|Item|rejected|duplicate|valid)"
    ;;
  processing)
    echo "=== Monitoring Processing Pipeline ==="
    echo "Press Ctrl+C to stop"
    echo ""
    docker compose logs -f api | grep -E "(SchemaGate|Curated|Projector|Index|Shard|Kafka)"
    ;;
  *)
    echo "Usage: $0 [api|kafka|redis|all|validation|processing]"
    echo ""
    echo "Options:"
    echo "  api         - Monitor API logs (default)"
    echo "  kafka       - Monitor Kafka logs"
    echo "  redis       - Monitor Redis logs"
    echo "  all         - Monitor all services"
    echo "  validation  - Monitor schema validation only"
    echo "  processing  - Monitor processing pipeline only"
    exit 1
    ;;
esac

