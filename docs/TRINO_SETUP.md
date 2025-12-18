# Trino Setup and Troubleshooting

## Current Issue

Trino is configured but the Hudi connector requires proper setup. The connector is failing to initialize because Hudi tables haven't been created yet.

## Quick Fix: Use JSONL Query API Instead

Since Hudi tables aren't set up yet, you can query JSONL files directly using a simpler approach.

### Option 1: Query JSONL Files Directly (Current Data)

The data is stored in JSONL files. You can query them directly:

```bash
# View all providers
cat data/hudi/providers/*.jsonl | jq .

# Count items per provider
for file in data/hudi/providers/*.jsonl; do
  echo "$(basename $file .jsonl): $(cat $file | jq '.items | length')"
done

# Search for specific provider
cat data/hudi/providers/*.jsonl | jq 'select(.provider_id == "10020084")'

# Get all items
cat data/hudi/providers/*.jsonl | jq -r '.items[] | {id: .id, name: .descriptor.name, price: .price.value}'
```

### Option 2: Fix Trino Configuration

Trino needs a working connector. For now, let's use a simpler connector or disable Hudi until tables are set up.

**Temporary Fix - Use Memory Connector for Testing:**

1. Create a memory connector config:
```bash
cat > trino/catalog/memory.properties << EOF
connector.name=memory
EOF
```

2. Restart Trino:
```bash
docker compose restart trino
```

3. Test with memory connector:
```bash
curl -X POST http://localhost:8080/api/trino/query \
  -H "Content-Type: application/json" \
  -d '{"sql": "SHOW CATALOGS"}' | jq .
```

### Option 3: Set Up Hudi Tables Properly

To use Hudi connector, you need to:

1. **Create Hudi tables from JSONL files** using Spark
2. **Store in MinIO/S3** (configured in docker-compose)
3. **Configure Trino to read from MinIO**

This requires:
- Spark job to convert JSONL → Hudi Parquet
- MinIO bucket setup
- Proper Hudi table schema

## Current Status

- ✅ Trino container is configured
- ✅ Trino API endpoints are ready
- ❌ Hudi connector needs table setup
- ✅ JSONL files contain the data

## Recommended Approach

**For now, use the JSONL files directly** or create a simple JSONL query API that doesn't require Trino.

Would you like me to create a JSONL query API that works with the current data files?

