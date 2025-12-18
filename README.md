# GCR Backend - Global Catalog Repository

This is the Go implementation of the Global Catalog Repository (GCR) for ONDC, following the CQRS architecture with Kafka, Redis, Hudi, and Trino.

## Architecture Overview

The system implements the full sequence diagram flow:

1. **Ingest (Command/Write Side)**:
   - Edge receives compressed `/on_search` from Seller BPP
   - Publishes to Kafka `catalog.ingest`
   - SchemaGate validates providers/items (partial acceptance)
   - Curated Writer writes to Hudi (stub: JSONL files)
   - Publishes `CatalogAccepted` events to Kafka `catalog.accepted`

2. **Projections (Redis Read Models)**:
   - Index Projector: `idx:{city}:{category}` → sellers
   - Shard Projector: `shard:{seller}:{city}:cat:{category}` → full JSON
   - Delta Projector: `delta:{seller}:{city}:{cat}:{tC}` → diff (TTL)

3. **Read/Delivery (Query Side)**:
   - Discovery API: `/ondc/search` queries Index + Policy
   - Discovery API: `/ondc/on_search` returns shard JSON (overlay-first)
   - Push Fan-out: (future) commit-triggered delivery

## Components

- **Edge + Baseline Validation**: HTTP handler with gzip decompression
- **SchemaGate**: Provider/item validation with partial acceptance
- **Curated Writer**: Writes to Hudi (JSONL stub)
- **Projectors**: Index/Shard/Delta builders consuming from Kafka
- **Discovery API**: Buyer-facing `/search` and `/on_search` endpoints
- **Policy Service**: Buyer×Seller authorization
- **Redis Bloom Filter**: Duplicate detection for providers

## Prerequisites

- Docker & Docker Compose
- Go 1.22+ (for local development)

## Quick Start

### 1. Environment Configuration

The project uses a `.env` file for environment variables. A `.env` file is already created with default values. You can modify it if needed:

```bash
cd gcr-backend
# .env file is already created with defaults
# Edit .env if you need to change ports, credentials, etc.
```

Key environment variables:
- `GCR_HTTP_ADDR`: API server address (default: `:8080`)
- `REDIS_ADDR`: Redis connection string (default: `redis:6379`)
- `KAFKA_BROKER`: Kafka broker address (default: `kafka:9092`)
- `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD`: MinIO credentials
- See `.env` file for all available options

### 2. Start all services

```bash
cd gcr-backend
docker compose build
docker compose up
```

**Note:** If you encounter "toomanyrequests: Rate exceeded" errors from Docker Hub, you have two options:

**Option A: Pull images one at a time (recommended)**
```bash
./pull-images.sh
# Then run: docker compose up
```

**Option B: Wait and retry**
- Docker Hub rate limits: 100 pulls per 6 hours for anonymous users
- Wait 10-30 minutes and retry `docker compose up`
- Or create a free Docker Hub account and login: `docker login`

**Troubleshooting Docker Desktop Issues:**

If you get "error creating temporary lease" or `docker ps` hangs:

1. **Quick fix - Restart Docker Desktop:**
   ```bash
   ./fix-docker.sh
   ```

2. **Manual restart:**
   - Quit Docker Desktop completely (right-click tray icon → Quit)
   - Wait 10 seconds
   - Start Docker Desktop again
   - Wait 30-60 seconds for it to fully start
   - Test with: `docker ps`

3. **If restart doesn't work - Reset Docker VM (WARNING: deletes all containers/volumes):**
   ```bash
   # Quit Docker Desktop first
   rm -rf ~/Library/Containers/com.docker.docker/Data/vms/0/data/Docker.raw
   # Then restart Docker Desktop
   ```

This starts:
- **API** (port 8080): Edge + Discovery endpoints
- **Redis** (port 6379): Bloom filter + read models
- **Kafka** (port 9092): `catalog.ingest` and `catalog.accepted` topics
- **Zookeeper** (port 2181): Kafka coordination
- **MinIO** (ports 9000/9001): S3-compatible storage for Hudi
- **Trino** (port 8081): Query engine for Hudi tables
- **Spark** (ports 7077/8082): Spark master for Hudi processing jobs (RPC: 7077, Web UI: 8082)

### 3. Health check

```bash
curl http://localhost:8080/health
# -> {"status":"ok"}
```

### 4. Send `/on_search` (ingest)

**Plain JSON:**
```bash
curl -X POST http://localhost:8080/ondc/on_search \
  -H "Content-Type: application/json" \
  --data-binary @../request_13_undefined_1729964994274.json \
  | gunzip
```

**GZIP-compressed (as per SNP requirement):**
```bash
gzip -c ../request_13_undefined_1729964994274.json > /tmp/on_search.gz

curl -X POST http://localhost:8080/ondc/on_search \
  -H "Content-Type: application/json" \
  -H "Content-Encoding: gzip" \
  --data-binary @/tmp/on_search.gz \
  | gunzip
```

**Expected response (gzip-compressed):**
```json
{
  "providers": 1,
  "duration_ms": 12
}
```

### 5. Query `/search` (discovery)

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
  }' | gunzip
```

### 6. Get `/on_search` shard (read)

```bash
curl "http://localhost:8080/ondc/on_search?seller_id=webapi.magicpin.in/oms_partner/ondc&city=std:020&category=1" \
  | gunzip
```

## Data Flow

### Ingest Flow (Sequence Diagram Steps 1-10)

1. **Seller BPP** → `POST /ondc/on_search` (gzip-compressed)
2. **Edge** decompresses, validates ONDC envelope
3. **Edge** → publishes to Kafka `catalog.ingest`
4. **SchemaGate** consumes, validates providers/items
5. **Rejections Store** writes rejected scopes
6. **Curated Writer** writes valid rows to Hudi (JSONL)
7. **Curated Writer** → publishes `CatalogAccepted` to `catalog.accepted`
8. **Index Projector** → updates Redis `idx:{city}:{category}`
9. **Shard Projector** → updates Redis `shard:{seller}:{city}:cat:{category}`
10. **Delta Projector** → updates Redis `delta:{seller}:{city}:{cat}:{tC}` (TTL)

### Read Flow (Sequence Diagram Steps 11-20)

1. **Buyer BAP** → `POST /ondc/search` {domain, city, category}
2. **Discovery** queries Redis Index `idx:{city}:{category}`
3. **Policy Service** filters allowed sellers
4. **Discovery** → returns seller list
5. **Buyer BAP** → `GET /ondc/on_search?seller_id=...&city=...&category=...`
6. **Discovery** reads Redis Shard (overlay-first if exists)
7. **Discovery** → returns gzip-compressed `/on_search` JSON

## Data Locations

- **Hudi stub**: `./data/hudi/providers/{provider_id}.jsonl`
- **Rejections**: `./data/rejections/rejections_{date}.jsonl`
- **Redis keys**:
  - Index: `idx:{city}:{category}`
  - Shard: `shard:{seller}:{city}:cat:{category}`
  - Delta: `delta:{seller}:{city}:{cat}:{tC}`
  - Policy: `policy:{buyer}:{seller}:{domain}:{city}`
  - Bloom: `gcr:providers` (RedisBloom filter)

## Kafka Topics

- **`catalog.ingest`**: Raw `/on_search` envelopes from Edge
- **`catalog.accepted`**: `CatalogAccepted` events after Hudi commit

## Testing

### Check Redis Index

```bash
docker exec -it gcr-backend-redis-1 redis-cli
> SMEMBERS idx:std:020:1
```

### Check Redis Shard

```bash
docker exec -it gcr-backend-redis-1 redis-cli
> GET shard:webapi.magicpin.in/oms_partner/ondc:std:020:cat:1
```

### Check Kafka Topics

```bash
docker exec -it gcr-backend-kafka-1 kafka-topics.sh --list --bootstrap-server localhost:9092
```

### View Hudi Stub Data

```bash
cat data/hudi/providers/*.jsonl
```

## Trino Query (Future)

Once Hudi tables are created from JSONL files, query via Trino:

```bash
docker exec -it gcr-backend-trino-1 trino
```

```sql
SHOW CATALOGS;
SHOW SCHEMAS FROM hudi;
SELECT * FROM hudi.default.providers LIMIT 10;
```

## Development

### Local Go Run (without Docker)

```bash
cd gcr-backend
go mod tidy
GCR_HTTP_ADDR=":8080" REDIS_ADDR="localhost:6379" KAFKA_BROKER="localhost:9092" go run ./cmd/gcr-api
```

(Requires Redis and Kafka running locally)

## Architecture Notes

- **CQRS**: Write (ingest) and Read (discovery) are separated
- **Event-driven**: Kafka topics connect components
- **Partial acceptance**: Bad providers don't block good ones
- **GZIP compression**: Request and response support compression
- **Redis Bloom**: Fast duplicate detection
- **Policy Service**: Buyer×Seller authorization (stub)
- **Hudi MoR**: Master Catalog Store (stub: JSONL, ready for Spark/Hudi job)

## Next Steps

- [ ] Implement full Hudi MoR writes (Spark job)
- [ ] Add Push Fan-out worker for commit-triggered delivery
- [ ] Implement buyer handshake for unknown sellers
- [ ] Add overlay shard support
- [ ] Add subscription registry
- [ ] Add observability (OTel traces/metrics)
