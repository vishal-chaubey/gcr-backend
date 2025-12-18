#!/bin/bash
# view-all-data.sh - View all data in GCR Backend

echo "=== GCR Backend Data Overview ==="
echo ""

echo "=== HUDI DATA (Providers) ==="
if [ -d "data/hudi/providers" ]; then
  file_count=$(ls -1 data/hudi/providers/*.jsonl 2>/dev/null | wc -l)
  echo "Provider Files: $file_count"
  if [ "$file_count" -gt 0 ]; then
    echo ""
    echo "Sample Provider:"
    cat data/hudi/providers/*.jsonl 2>/dev/null | head -1 | jq '{provider_id, domain, city, items_count: (.items | length)}' 2>/dev/null || echo "No valid JSON found"
  fi
else
  echo "No provider data directory found"
fi
echo ""

echo "=== REDIS DATA ==="
if docker ps | grep -q redis; then
  total_keys=$(docker exec gcr-backend-redis-1 redis-cli DBSIZE 2>/dev/null || echo "0")
  echo "Total Keys: $total_keys"
  echo ""
  
  idx_count=$(docker exec gcr-backend-redis-1 redis-cli --scan --pattern "idx:*" 2>/dev/null | wc -l)
  echo "Indexes (idx:*): $idx_count keys"
  
  shard_count=$(docker exec gcr-backend-redis-1 redis-cli --scan --pattern "shard:*" 2>/dev/null | wc -l)
  echo "Shards (shard:*): $shard_count keys"
  
  delta_count=$(docker exec gcr-backend-redis-1 redis-cli --scan --pattern "delta:*" 2>/dev/null | wc -l)
  echo "Deltas (delta:*): $delta_count keys"
  
  policy_count=$(docker exec gcr-backend-redis-1 redis-cli --scan --pattern "policy:*" 2>/dev/null | wc -l)
  echo "Policies (policy:*): $policy_count keys"
  
  echo ""
  echo "Sample Index (idx:std:020:1):"
  docker exec gcr-backend-redis-1 redis-cli SMEMBERS "idx:std:020:1" 2>/dev/null | head -5 || echo "No data"
else
  echo "Redis container not running"
fi
echo ""

echo "=== KAFKA TOPICS ==="
if docker ps | grep -q kafka; then
  docker exec gcr-backend-kafka-1 kafka-topics.sh --list --bootstrap-server localhost:9092 2>/dev/null || echo "Cannot connect to Kafka"
else
  echo "Kafka container not running"
fi
echo ""

echo "=== REJECTIONS ==="
if [ -d "data/rejections" ]; then
  rejection_count=$(cat data/rejections/*.jsonl 2>/dev/null | wc -l)
  echo "Total Rejections: $rejection_count"
  if [ "$rejection_count" -gt 0 ]; then
    echo ""
    echo "Rejection Summary:"
    cat data/rejections/*.jsonl 2>/dev/null | jq -r '.scope' | cut -d: -f1 | sort | uniq -c 2>/dev/null || echo "No valid rejection data"
  fi
else
  echo "No rejections directory found"
fi
echo ""

echo "=== BLOOM FILTERS ==="
if docker ps | grep -q redis; then
  echo "Providers Bloom Filter:"
  docker exec gcr-backend-redis-1 redis-cli BF.INFO "gcr:providers" 2>/dev/null | grep -E "Capacity|Size" || echo "Not found"
  echo ""
  echo "Items Bloom Filter:"
  docker exec gcr-backend-redis-1 redis-cli BF.INFO "gcr:items" 2>/dev/null | grep -E "Capacity|Size" || echo "Not found"
else
  echo "Redis container not running"
fi
echo ""

echo "=== DONE ==="

