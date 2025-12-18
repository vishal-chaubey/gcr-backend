# How to View All Data in GCR Backend

This guide shows you how to inspect data stored in Hudi, Kafka, Redis, and other storage systems.

---

## 📁 1. Hudi Data (JSONL Files)

### View All Provider Files
```bash
cd /Users/vishalkumarchaubey/Desktop/GCR/gcr-backend

# List all provider files
ls -lh data/hudi/providers/

# Count how many providers
ls data/hudi/providers/ | wc -l
```

### View Specific Provider Data
```bash
# View a specific provider's data
cat data/hudi/providers/10020084.jsonl | jq .

# View first provider (pretty formatted)
cat data/hudi/providers/10020084.jsonl | jq . | head -50

# View all providers
cat data/hudi/providers/*.jsonl | jq .

# Count items in a provider
cat data/hudi/providers/10020084.jsonl | jq '.items | length'
```

### View Provider Summary
```bash
# Show provider IDs and item counts
for file in data/hudi/providers/*.jsonl; do
  echo "Provider: $(basename $file .jsonl)"
  echo "Items: $(cat $file | jq '.items | length')"
  echo "---"
done
```

### View All Data in Table Format
```bash
# Using jq to create a summary table
cat data/hudi/providers/*.jsonl | jq -r '
  "Provider ID: " + .provider_id,
  "Domain: " + .domain,
  "City: " + .city,
  "Items Count: " + (.items | length | tostring),
  "---"
'
```

---

## 🔴 2. Redis Data

### Connect to Redis CLI
```bash
# Enter Redis CLI
docker exec -it gcr-backend-redis-1 redis-cli

# Or run commands directly
docker exec gcr-backend-redis-1 redis-cli <command>
```

### View All Keys
```bash
# List all keys
docker exec gcr-backend-redis-1 redis-cli KEYS "*"

# Count total keys
docker exec gcr-backend-redis-1 redis-cli DBSIZE

# List keys by pattern
docker exec gcr-backend-redis-1 redis-cli KEYS "idx:*"
docker exec gcr-backend-redis-1 redis-cli KEYS "shard:*"
docker exec gcr-backend-redis-1 redis-cli KEYS "delta:*"
docker exec gcr-backend-redis-1 redis-cli KEYS "policy:*"
```

### View Index Data (Seller Lists)
```bash
# View all indexes
docker exec gcr-backend-redis-1 redis-cli KEYS "idx:*"

# View sellers for a specific city+category
docker exec gcr-backend-redis-1 redis-cli SMEMBERS "idx:std:020:1"

# View all indexes with their seller counts
docker exec gcr-backend-redis-1 redis-cli --scan --pattern "idx:*" | while read key; do
  echo "Index: $key"
  docker exec gcr-backend-redis-1 redis-cli SCARD "$key"
  docker exec gcr-backend-redis-1 redis-cli SMEMBERS "$key"
  echo "---"
done
```

### View Shard Data (Full Catalogs)
```bash
# List all shards
docker exec gcr-backend-redis-1 redis-cli KEYS "shard:*"

# View a specific shard (full catalog JSON)
docker exec gcr-backend-redis-1 redis-cli GET "shard:webapi.magicpin.in/oms_partner/ondc:std:020:cat:1" | jq .

# View shard with pretty formatting
docker exec gcr-backend-redis-1 redis-cli GET "shard:webapi.magicpin.in/oms_partner/ondc:std:020:cat:1" | python3 -m json.tool

# View all shards
docker exec gcr-backend-redis-1 redis-cli --scan --pattern "shard:*" | while read key; do
  echo "Shard: $key"
  docker exec gcr-backend-redis-1 redis-cli GET "$key" | jq . | head -20
  echo "---"
done
```

### View Delta Data (Changes)
```bash
# List all deltas
docker exec gcr-backend-redis-1 redis-cli KEYS "delta:*"

# View a specific delta
docker exec gcr-backend-redis-1 redis-cli GET "delta:webapi.magicpin.in/oms_partner/ondc:std:020:cat:1:2025-12-18T..." | jq .

# Check TTL (time to live) on deltas
docker exec gcr-backend-redis-1 redis-cli --scan --pattern "delta:*" | while read key; do
  ttl=$(docker exec gcr-backend-redis-1 redis-cli TTL "$key")
  echo "Delta: $key (TTL: ${ttl}s)"
done
```

### View Policy Data (Authorization)
```bash
# List all policies
docker exec gcr-backend-redis-1 redis-cli KEYS "policy:*"

# View a specific policy
docker exec gcr-backend-redis-1 redis-cli GET "policy:buyer-backend.himira.co.in:webapi.magicpin.in/oms_partner/ondc:ONDC:RET11:std:020"
```

### View Bloom Filter Data
```bash
# Check if Bloom filters exist
docker exec gcr-backend-redis-1 redis-cli BF.INFO "gcr:providers"
docker exec gcr-backend-redis-1 redis-cli BF.INFO "gcr:items"

# Check if a specific item exists (example)
docker exec gcr-backend-redis-1 redis-cli BF.EXISTS "gcr:items" "ONDC:RET11:std:020:10020084:61407046"
```

### Export All Redis Data
```bash
# Export all keys and values to a file
docker exec gcr-backend-redis-1 redis-cli --scan --pattern "*" | while read key; do
  echo "Key: $key"
  docker exec gcr-backend-redis-1 redis-cli GET "$key" 2>/dev/null || \
  docker exec gcr-backend-redis-1 redis-cli SMEMBERS "$key" 2>/dev/null || \
  docker exec gcr-backend-redis-1 redis-cli TYPE "$key"
  echo "---"
done > redis_export.txt
```

---

## 📨 3. Kafka Data

### List All Topics
```bash
# List all Kafka topics
docker exec gcr-backend-kafka-1 kafka-topics.sh \
  --list \
  --bootstrap-server localhost:9092
```

### View Topic Details
```bash
# View details of a specific topic
docker exec gcr-backend-kafka-1 kafka-topics.sh \
  --describe \
  --topic catalog.ingest \
  --bootstrap-server localhost:9092

docker exec gcr-backend-kafka-1 kafka-topics.sh \
  --describe \
  --topic catalog.accepted \
  --bootstrap-server localhost:9092
```

### View Messages in a Topic
```bash
# View messages from catalog.ingest (last 10)
docker exec gcr-backend-kafka-1 kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic catalog.ingest \
  --from-beginning \
  --max-messages 10

# View messages from catalog.accepted
docker exec gcr-backend-kafka-1 kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic catalog.accepted \
  --from-beginning \
  --max-messages 10

# View messages with pretty formatting (using jq)
docker exec gcr-backend-kafka-1 kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic catalog.accepted \
  --from-beginning \
  --max-messages 5 | jq .
```

### View Topic Statistics
```bash
# Get message count in a topic
docker exec gcr-backend-kafka-1 kafka-run-class.sh \
  kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic catalog.ingest

# View consumer groups
docker exec gcr-backend-kafka-1 kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --list

# View consumer group details
docker exec gcr-backend-kafka-1 kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --group schemagate-group \
  --describe
```

### Monitor Kafka in Real-Time
```bash
# Watch messages as they arrive
docker exec -it gcr-backend-kafka-1 kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic catalog.ingest \
  --from-beginning

# Watch with formatting
docker exec -it gcr-backend-kafka-1 kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic catalog.accepted \
  --from-beginning | jq .
```

---

## 📋 4. Rejection Data

### View Rejection Files
```bash
# List all rejection files
ls -lh data/rejections/

# View rejections for today
cat data/rejections/rejections_$(date +%Y-%m-%d).jsonl | jq .

# View all rejections
cat data/rejections/*.jsonl | jq .

# Count rejections by type
cat data/rejections/*.jsonl | jq -r '.scope' | sort | uniq -c

# View rejection reasons
cat data/rejections/*.jsonl | jq -r '"\(.scope): \(.reason)"'
```

### View Rejection Summary
```bash
# Summary of rejections
echo "Total Rejections: $(cat data/rejections/*.jsonl 2>/dev/null | wc -l)"
echo ""
echo "By Scope:"
cat data/rejections/*.jsonl 2>/dev/null | jq -r '.scope' | cut -d: -f1 | sort | uniq -c
echo ""
echo "By Reason:"
cat data/rejections/*.jsonl 2>/dev/null | jq -r '.reason' | sort | uniq -c
```

---

## 🔍 5. Comprehensive Data View Script

Create a script to view everything at once:

```bash
#!/bin/bash
# view-all-data.sh - View all data in GCR Backend

echo "=== GCR Backend Data Overview ==="
echo ""

echo "=== HUDI DATA (Providers) ==="
echo "Provider Files:"
ls -lh data/hudi/providers/ 2>/dev/null | tail -n +2 | wc -l
echo "files"
echo ""
echo "Sample Provider:"
cat data/hudi/providers/*.jsonl 2>/dev/null | head -1 | jq '{provider_id, domain, city, items_count: (.items | length)}' 2>/dev/null
echo ""

echo "=== REDIS DATA ==="
echo "Total Keys: $(docker exec gcr-backend-redis-1 redis-cli DBSIZE 2>/dev/null)"
echo ""
echo "Indexes (idx:*):"
docker exec gcr-backend-redis-1 redis-cli --scan --pattern "idx:*" 2>/dev/null | wc -l
echo "keys"
echo ""
echo "Shards (shard:*):"
docker exec gcr-backend-redis-1 redis-cli --scan --pattern "shard:*" 2>/dev/null | wc -l
echo "keys"
echo ""
echo "Deltas (delta:*):"
docker exec gcr-backend-redis-1 redis-cli --scan --pattern "delta:*" 2>/dev/null | wc -l
echo "keys"
echo ""

echo "=== KAFKA TOPICS ==="
docker exec gcr-backend-kafka-1 kafka-topics.sh --list --bootstrap-server localhost:9092 2>/dev/null
echo ""

echo "=== REJECTIONS ==="
echo "Total Rejections: $(cat data/rejections/*.jsonl 2>/dev/null | wc -l)"
echo ""

echo "=== BLOOM FILTERS ==="
docker exec gcr-backend-redis-1 redis-cli BF.INFO "gcr:providers" 2>/dev/null | grep -E "Capacity|Size"
docker exec gcr-backend-redis-1 redis-cli BF.INFO "gcr:items" 2>/dev/null | grep -E "Capacity|Size"
```

Save as `view-all-data.sh` and run:
```bash
chmod +x view-all-data.sh
./view-all-data.sh
```

---

## 🎯 Quick Reference Commands

### View Everything at Once
```bash
# Hudi
ls -lh data/hudi/providers/ && cat data/hudi/providers/*.jsonl | jq . | head -20

# Redis - All keys
docker exec gcr-backend-redis-1 redis-cli KEYS "*"

# Redis - Indexes
docker exec gcr-backend-redis-1 redis-cli --scan --pattern "idx:*" | xargs -I {} sh -c 'echo "{}:" && docker exec gcr-backend-redis-1 redis-cli SMEMBERS {}'

# Redis - Shards
docker exec gcr-backend-redis-1 redis-cli --scan --pattern "shard:*" | head -1 | xargs -I {} docker exec gcr-backend-redis-1 redis-cli GET {} | jq .

# Kafka
docker exec gcr-backend-kafka-1 kafka-topics.sh --list --bootstrap-server localhost:9092
```

### Interactive Redis Browser
```bash
# Enter Redis CLI for interactive exploration
docker exec -it gcr-backend-redis-1 redis-cli

# Then use commands:
# KEYS *                    # List all keys
# GET <key>                 # Get value
# SMEMBERS <key>           # Get set members
# TYPE <key>               # Get key type
# TTL <key>                # Get time to live
# BF.INFO gcr:items        # Bloom filter info
```

---

## 📊 Data Location Summary

| Storage | Location | How to View |
|---------|----------|-------------|
| **Hudi** | `data/hudi/providers/*.jsonl` | `cat data/hudi/providers/*.jsonl \| jq .` |
| **Redis Index** | Redis key `idx:*` | `redis-cli SMEMBERS idx:std:020:1` |
| **Redis Shard** | Redis key `shard:*` | `redis-cli GET shard:... \| jq .` |
| **Redis Delta** | Redis key `delta:*` | `redis-cli GET delta:... \| jq .` |
| **Kafka Ingest** | Topic `catalog.ingest` | `kafka-console-consumer.sh --topic catalog.ingest` |
| **Kafka Accepted** | Topic `catalog.accepted` | `kafka-console-consumer.sh --topic catalog.accepted` |
| **Rejections** | `data/rejections/*.jsonl` | `cat data/rejections/*.jsonl \| jq .` |
| **Bloom Filters** | Redis `gcr:providers`, `gcr:items` | `redis-cli BF.INFO gcr:items` |

---

## 🔧 Troubleshooting

### If Redis CLI doesn't work:
```bash
# Check if Redis container is running
docker ps | grep redis

# Restart Redis if needed
docker compose restart redis
```

### If Kafka commands don't work:
```bash
# Check if Kafka is running
docker ps | grep kafka

# Check Kafka logs
docker compose logs kafka --tail 20
```

### If data files are missing:
```bash
# Check if data directory exists
ls -la data/

# Run a test to generate data
./test-flow.sh
```

---

## 💡 Pro Tips

1. **Use jq for pretty JSON**: Always pipe JSON through `jq .` for readability
2. **Use grep to filter**: `redis-cli KEYS "*" | grep "idx:"` to find specific keys
3. **Export data**: Redirect output to files for analysis: `redis-cli GET key > data.json`
4. **Monitor in real-time**: Use `watch` command to monitor changes: `watch -n 1 'redis-cli DBSIZE'`

