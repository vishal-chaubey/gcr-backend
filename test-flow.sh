#!/bin/bash
# Quick test script for GCR backend flow

# Don't exit on errors, continue to show all results
set +e

API="http://localhost:8080"
PAYLOAD="../request_13_undefined_1729964994274.json"

echo "=== 1. Health Check ==="
curl -s "$API/health" | jq .
echo ""

echo "=== 2. Send /on_search (ingest) - Plain JSON ==="
time curl -s -X POST "$API/ondc/on_search" \
  -H "Content-Type: application/json" \
  --data-binary @"$PAYLOAD" \
  | gunzip | jq .
echo ""

echo "=== 3. Wait for Kafka processing (2s) ==="
sleep 2

echo "=== 4. Query /search (discovery) ==="
curl -s -X POST "$API/ondc/search" \
  -H "Content-Type: application/json" \
  -d '{
    "context": {
      "domain": "ONDC:RET11",
      "city": "std:020",
      "action": "search",
      "bap_id": "buyer-backend.himira.co.in",
      "bap_uri": "https://premium-lion-especially.ngrok-free.app"
    },
    "message": {
      "intent": {
        "item": {
          "category": {
            "id": "1"
          }
        }
      }
    }
  }' | (gunzip 2>/dev/null || cat) | jq .
echo ""

echo "=== 5. Get /on_search shard (read) ==="
curl -s "$API/ondc/on_search?seller_id=webapi.magicpin.in/oms_partner/ondc&city=std:020&category=1" \
  | (gunzip 2>/dev/null || cat) | jq . 2>/dev/null || echo "No data or invalid response"
echo ""

echo "=== 6. Check Redis Index ==="
docker exec gcr-backend-redis-1 redis-cli SMEMBERS idx:std:020:1
echo ""

echo "=== 7. Check Redis Shard ==="
docker exec gcr-backend-redis-1 redis-cli GET shard:webapi.magicpin.in/oms_partner/ondc:std:020:cat:1 | jq . 2>/dev/null || echo "No shard data found"
echo ""

echo "=== Flow test complete ==="

