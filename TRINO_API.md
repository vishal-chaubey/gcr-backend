# Trino Query API Documentation

This document explains how to use the Trino Query API to query data from Hudi tables.

## Overview

Trino is used to query Hudi tables (data lake format) for analytics, reporting, and data exploration. The API provides REST endpoints to execute SQL queries against Trino.

**Current Status:**
- Trino is configured and running (port 8081)
- Hudi connector is configured
- API endpoints are available
- **Note:** Hudi tables need to be created from JSONL files (see setup below)

---

## API Endpoints

### Base URL
```
http://localhost:8080/api/trino
```

### 1. Health Check

Check if Trino is available and accessible.

**Endpoint:** `GET /api/trino/health`

**Response:**
```json
{
  "status": "ok",
  "message": "Trino is available"
}
```

**Example:**
```bash
curl http://localhost:8080/api/trino/health
```

---

### 2. Execute Custom SQL Query

Execute any SQL query against Trino.

**Endpoint:** `POST /api/trino/query`

**Request Body:**
```json
{
  "sql": "SELECT * FROM hudi.default.providers LIMIT 10"
}
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "provider_id": "10020084",
      "domain": "ONDC:RET11",
      "city": "std:020",
      "bpp_id": "webapi.magicpin.in/oms_partner/ondc",
      "timestamp": "2025-12-18T17:56:21.346478678Z",
      "descriptor": {...},
      "items": [...]
    }
  ],
  "stats": {
    "state": "FINISHED",
    "elapsedTime": "1.2s",
    "totalRows": 10,
    "totalBytes": 1024
  }
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/trino/query \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "SELECT provider_id, domain, city, COUNT(*) as item_count FROM hudi.default.providers GROUP BY provider_id, domain, city"
  }' | jq .
```

---

### 3. Get All Providers

Get a list of all providers with pagination.

**Endpoint:** `GET /api/trino/providers`

**Query Parameters:**
- `limit` (optional): Number of results (default: 100, max: 1000)
- `offset` (optional): Pagination offset (default: 0)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "provider_id": "10020084",
      "domain": "ONDC:RET11",
      "city": "std:020",
      "bpp_id": "webapi.magicpin.in/oms_partner/ondc",
      "timestamp": "2025-12-18T17:56:21.346478678Z",
      "provider_name": "Abhijeet",
      "items_count": 8999
    }
  ]
}
```

**Example:**
```bash
# Get first 50 providers
curl "http://localhost:8080/api/trino/providers?limit=50" | jq .

# Get next page
curl "http://localhost:8080/api/trino/providers?limit=50&offset=50" | jq .
```

---

### 4. Get Specific Provider

Get detailed information about a specific provider.

**Endpoint:** `GET /api/trino/providers/{provider_id}`

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "provider_id": "10020084",
      "domain": "ONDC:RET11",
      "city": "std:020",
      "bpp_id": "webapi.magicpin.in/oms_partner/ondc",
      "timestamp": "2025-12-18T17:56:21.346478678Z",
      "descriptor": {
        "name": "Abhijeet",
        "symbol": "...",
        "short_desc": "...",
        "long_desc": "...",
        "images": [...]
      },
      "categories": [...],
      "items": [...]
    }
  ]
}
```

**Example:**
```bash
curl http://localhost:8080/api/trino/providers/10020084 | jq .
```

---

### 5. Get Items

Get items with optional filters.

**Endpoint:** `GET /api/trino/items`

**Query Parameters:**
- `limit` (optional): Number of results (default: 100, max: 1000)
- `provider_id` (optional): Filter by provider ID
- `category_id` (optional): Filter by category ID
- `city` (optional): Filter by city

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "provider_id": "10020084",
      "city": "std:020",
      "domain": "ONDC:RET11",
      "item_id": "61407046",
      "item_name": "Margherita Semizza (Half Pizza)(Serves 1)",
      "category_id": "F&B",
      "price_value": "122.50",
      "price_currency": "INR"
    }
  ]
}
```

**Example:**
```bash
# Get all items
curl "http://localhost:8080/api/trino/items?limit=100" | jq .

# Get items for a specific provider
curl "http://localhost:8080/api/trino/items?provider_id=10020084&limit=50" | jq .

# Get items by category
curl "http://localhost:8080/api/trino/items?category_id=1&city=std:020" | jq .
```

---

### 6. Get Statistics

Get overall statistics about the data.

**Endpoint:** `GET /api/trino/stats`

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "total_providers": 5,
      "total_records": 5,
      "total_items": 45000,
      "total_domains": 1,
      "total_cities": 1,
      "latest_update": "2025-12-18T17:56:21.346478678Z"
    }
  ]
}
```

**Example:**
```bash
curl http://localhost:8080/api/trino/stats | jq .
```

---

## Setup Hudi Tables (Required)

Before using Trino API, Hudi tables need to be created from JSONL files. Currently, the system writes to JSONL files as a stub.

### Option 1: Using Spark (Recommended for Production)

```bash
# Submit Spark job to create Hudi tables from JSONL
docker exec -it gcr-backend-spark-1 spark-submit \
  --class org.apache.hudi.utilities.deltastreamer.HoodieDeltaStreamer \
  --packages org.apache.hudi:hudi-spark3.5-bundle_2.12:0.14.0 \
  --conf spark.serializer=org.apache.spark.serializer.KryoSerializer \
  /path/to/hudi-job.jar
```

### Option 2: Manual Table Creation (For Testing)

```bash
# Connect to Trino
docker exec -it gcr-backend-trino-1 trino

# Create table from JSONL files
CREATE TABLE hudi.default.providers (
  provider_id VARCHAR,
  domain VARCHAR,
  city VARCHAR,
  bap_id VARCHAR,
  bpp_id VARCHAR,
  timestamp VARCHAR,
  descriptor JSON,
  categories JSON,
  items JSON
)
WITH (
  format = 'JSON',
  external_location = 's3://gcr-data/hudi/providers/'
);
```

### Option 3: Query JSONL Directly (Current Workaround)

Since Hudi tables may not be set up yet, you can query JSONL files directly:

```bash
# View JSONL data
cat data/hudi/providers/*.jsonl | jq .

# Count items
cat data/hudi/providers/*.jsonl | jq '.items | length' | awk '{sum+=$1} END {print sum}'
```

---

## Example Queries

### Count Providers by City
```bash
curl -X POST http://localhost:8080/api/trino/query \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "SELECT city, COUNT(DISTINCT provider_id) as provider_count FROM hudi.default.providers GROUP BY city"
  }' | jq .
```

### Get Items by Price Range
```bash
curl -X POST http://localhost:8080/api/trino/query \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "SELECT provider_id, json_extract_scalar(item, \"$.id\") as item_id, json_extract_scalar(item, \"$.price.value\") as price FROM hudi.default.providers CROSS JOIN UNNEST(json_extract(items, \"$\")) AS t(item) WHERE CAST(json_extract_scalar(item, \"$.price.value\") AS DOUBLE) BETWEEN 100 AND 500 LIMIT 100"
  }' | jq .
```

### Get Latest Providers
```bash
curl -X POST http://localhost:8080/api/trino/query \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "SELECT provider_id, domain, city, timestamp FROM hudi.default.providers ORDER BY timestamp DESC LIMIT 10"
  }' | jq .
```

---

## Error Handling

### Trino Not Available
```json
{
  "status": "error",
  "message": "Trino health check failed: status 503"
}
```

### Invalid SQL Query
```json
{
  "success": false,
  "error": "trino error (status 400): line 1:1: mismatched input..."
}
```

### Table Not Found
```json
{
  "success": false,
  "error": "Table 'hudi.default.providers' does not exist"
}
```

---

## Configuration

### Environment Variables

Add to `.env`:
```bash
TRINO_URL=http://trino:8080
TRINO_USER=admin
```

### Trino Connection

- **URL:** `http://trino:8080` (internal) or `http://localhost:8081` (external)
- **Catalog:** `hudi`
- **Schema:** `default`
- **User:** `admin`

---

## Troubleshooting

### Check Trino Status
```bash
# Check if Trino is running
docker ps | grep trino

# Check Trino logs
docker compose logs trino --tail 50

# Test Trino connection
docker exec -it gcr-backend-trino-1 trino --execute "SHOW CATALOGS"
```

### Check Hudi Tables
```bash
# List tables
docker exec -it gcr-backend-trino-1 trino --execute "SHOW TABLES FROM hudi.default"

# Describe table
docker exec -it gcr-backend-trino-1 trino --execute "DESCRIBE hudi.default.providers"
```

### Common Issues

1. **Table doesn't exist**: Hudi tables need to be created from JSONL files
2. **Connection refused**: Check if Trino container is running
3. **Query timeout**: Increase timeout in client.go or simplify query
4. **No data**: Ensure JSONL files exist in `data/hudi/providers/`

---

## Next Steps

1. **Set up Hudi tables** from JSONL files using Spark
2. **Test queries** using the API endpoints
3. **Create indexes** for better query performance
4. **Set up monitoring** for query performance

---

## API Summary

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/trino/health` | GET | Health check |
| `/api/trino/query` | POST | Execute custom SQL |
| `/api/trino/providers` | GET | List providers |
| `/api/trino/providers/{id}` | GET | Get provider details |
| `/api/trino/items` | GET | List items with filters |
| `/api/trino/stats` | GET | Get statistics |

