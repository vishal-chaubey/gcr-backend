# API Endpoints Summary

## Available APIs

### 1. **JSONL Data API** (`/api/data/*`) - ✅ **WORKING NOW**

Queries JSONL files directly - works immediately with current data.

#### Endpoints:

```bash
# Get statistics
curl http://localhost:8080/api/data/stats | jq .

# Get all providers (with pagination)
curl "http://localhost:8080/api/data/providers?limit=10&offset=0" | jq .

# Get specific provider
curl http://localhost:8080/api/data/providers/10020084 | jq .

# Get items (with filters)
curl "http://localhost:8080/api/data/items?provider_id=10020084&limit=50" | jq .
curl "http://localhost:8080/api/data/items?category_id=1&city=std:020" | jq .
```

**Status:** ✅ Fully functional, returns data from JSONL files

---

### 2. **Trino Query API** (`/api/trino/*`) - ⚠️ **Requires Setup**

Queries Hudi tables via Trino - requires Hudi tables to be created first.

#### Endpoints:

```bash
# Health check
curl http://localhost:8080/api/trino/health | jq .

# Execute custom SQL
curl -X POST http://localhost:8080/api/trino/query \
  -H "Content-Type: application/json" \
  -d '{"sql": "SHOW CATALOGS"}' | jq .

# Get providers
curl "http://localhost:8080/api/trino/providers?limit=10" | jq .

# Get items
curl "http://localhost:8080/api/trino/items?limit=50" | jq .
```

**Status:** ⚠️ API ready, but Trino needs Hudi tables configured

**Current Issue:** Trino container can't resolve hostname (needs network fix or Hudi setup)

---

### 3. **ONDC Discovery API** (`/ondc/*`)

For ONDC protocol endpoints.

```bash
# Health check
curl http://localhost:8080/health

# Search (buyer)
curl -X POST http://localhost:8080/ondc/search \
  -H "Content-Type: application/json" \
  -d '{...}' | gunzip | jq .

# On Search (seller ingest)
curl -X POST http://localhost:8080/ondc/on_search \
  -H "Content-Type: application/json" \
  --data-binary @request.json | gunzip | jq .
```

---

## Quick Test All APIs

```bash
# JSONL API (works now)
curl http://localhost:8080/api/data/stats | jq .
curl "http://localhost:8080/api/data/providers?limit=5" | jq .

# Trino API (needs setup)
curl http://localhost:8080/api/trino/health | jq .

# ONDC API
curl http://localhost:8080/health
```

---

## Data Sources

| API | Data Source | Status |
|-----|-------------|--------|
| `/api/data/*` | JSONL files (`data/hudi/providers/*.jsonl`) | ✅ Working |
| `/api/trino/*` | Hudi tables via Trino | ⚠️ Needs setup |
| `/ondc/*` | Redis (Index/Shard) | ✅ Working |

---

## Recommendation

**Use `/api/data/*` endpoints** for now - they work immediately with your current data files!

