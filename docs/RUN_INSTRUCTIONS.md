# GCR Backend - Run Instructions

## Prerequisites
- Docker Desktop running
- Terminal access (bash/zsh)

## Step 1: Start All Services

Open a terminal and navigate to the project directory:

```bash
cd /Users/vishalkumarchaubey/Desktop/GCR/gcr-backend
```

Start all services:

```bash
docker compose up -d
```

This will start:
- API (port 8080)
- Redis (port 6379)
- Kafka (port 9092)
- Zookeeper (port 2181)
- MinIO (ports 9000/9001)
- Spark (ports 7077/8082)

## Step 2: Check Service Status

Verify all services are running:

```bash
docker compose ps
```

All services should show "Up" status.

## Step 3: Monitor Logs (In Separate Terminal)

Open a **NEW terminal window** (on your Mac) and run:

### Option A: Monitor All Services
```bash
cd /Users/vishalkumarchaubey/Desktop/GCR/gcr-backend
docker compose logs -f
```

### Option B: Monitor Only API (Recommended)
```bash
cd /Users/vishalkumarchaubey/Desktop/GCR/gcr-backend
docker compose logs -f api
```

### Option C: Monitor Specific Service
```bash
# Kafka logs
docker compose logs -f kafka

# Redis logs
docker compose logs -f redis

# All services with timestamps
docker compose logs -f --timestamps
```

## Step 4: Test the System

In your **original terminal** (or another terminal), run the test flow:

```bash
cd /Users/vishalkumarchaubey/Desktop/GCR/gcr-backend
./test-flow.sh
```

Or test manually:

### Health Check
```bash
curl http://localhost:8080/health
```

### Send on_search (Ingest)
```bash
curl -X POST http://localhost:8080/ondc/on_search \
  -H "Content-Type: application/json" \
  --data-binary @../request_13_undefined_1729964994274.json \
  | gunzip | jq .
```

### Query Search (Discovery)
```bash
curl -X POST http://localhost:8080/ondc/search \
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
  }' | gunzip | jq .
```

## Step 5: Monitor Processing in Real-Time

### Watch API Logs (Schema Validation & Processing)
```bash
# In a separate terminal
docker compose logs -f api | grep -E "(SchemaGate|Provider|Item|rejected|duplicate|valid)"
```

### Watch Kafka Processing
```bash
docker compose logs -f api | grep -E "(Kafka|publish|consume)"
```

### Watch Redis Operations
```bash
docker compose logs -f api | grep -E "(Redis|Bloom|BF\.ADD)"
```

### Watch All Processing Steps
```bash
docker compose logs -f api | grep -E "(SchemaGate|Curated|Projector|Index|Shard)"
```

## Step 6: Check Data

### Check Redis Index
```bash
docker exec gcr-backend-redis-1 redis-cli SMEMBERS idx:std:020:1
```

### Check Redis Shard
```bash
docker exec gcr-backend-redis-1 redis-cli GET "shard:webapi.magicpin.in/oms_partner/ondc:std:020:cat:1"
```

### Check Hudi Data Files
```bash
ls -lh data/hudi/providers/
cat data/hudi/providers/*.jsonl | head -5
```

### Check Rejections
```bash
ls -lh data/rejections/
cat data/rejections/*.jsonl | head -10
```

## Step 7: Stop Services

When done:

```bash
docker compose down
```

To stop and remove volumes (clean slate):

```bash
docker compose down -v
```

## Useful Commands

### Restart API Only (After Code Changes)
```bash
docker compose build api
docker compose up -d api
```

### View Last 100 Lines of API Logs
```bash
docker compose logs --tail 100 api
```

### Follow Logs with Timestamps
```bash
docker compose logs -f --timestamps api
```

### Check Container Resource Usage
```bash
docker stats
```

### Access Redis CLI
```bash
docker exec -it gcr-backend-redis-1 redis-cli
```

### Access Kafka Topics
```bash
docker exec -it gcr-backend-kafka-1 kafka-topics.sh --list --bootstrap-server localhost:9092
```

## Monitoring Schema Validation

To see detailed schema validation output:

```bash
# Watch provider validations
docker compose logs -f api | grep "provider"

# Watch item validations
docker compose logs -f api | grep "item"

# Watch rejections
docker compose logs -f api | grep "rejected"

# Watch duplicates
docker compose logs -f api | grep "duplicate"
```

## Performance Monitoring

### Watch Processing Time
```bash
docker compose logs -f api | grep -E "(duration|ms|seconds)"
```

### Count Processed Items
```bash
docker compose logs api | grep "SchemaGate: duplicate item" | wc -l
docker compose logs api | grep "SchemaGate: rejected item" | wc -l
```

## Troubleshooting

### If services won't start:
```bash
docker compose down -v
docker compose up -d
```

### If API has errors:
```bash
docker compose logs api --tail 50
docker compose restart api
```

### Check Kafka connectivity:
```bash
docker compose logs kafka --tail 20
docker compose logs api | grep -i kafka
```

