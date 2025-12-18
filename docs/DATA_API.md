# Data Query API Documentation

This API allows you to query data from JSONL files (current storage) without requiring Trino/Hudi setup.

## Base URL
```
http://localhost:8080/api/data
```

---

## Endpoints

### 1. Get All Providers

**Endpoint:** `GET /api/data/providers`

**Query Parameters:**
- `limit` (optional): Number of results (default: 100, max: 1000)
- `offset` (optional): Pagination offset (default: 0)

**Example:**
```bash
curl "http://localhost:8080/api/data/providers?limit=50" | jq .

curl "http://localhost:8080/api/data/providers?limit=50&offset=50" | jq .
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
      "provider_name": "Abhijeet",
      "items_count": 8999
    }
  ]
}
```

---

### 2. Get Specific Provider

**Endpoint:** `GET /api/data/providers/{provider_id}`

**Example:**
```bash
curl http://localhost:8080/api/data/providers/10020084 | jq .
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
      "categories": [...],
      "items": [...]
    }
  ]
}
```

---

### 3. Get Items

**Endpoint:** `GET /api/data/items`

**Query Parameters:**
- `limit` (optional): Number of results (default: 100, max: 1000)
- `provider_id` (optional): Filter by provider ID
- `category_id` (optional): Filter by category ID
- `city` (optional): Filter by city

**Examples:**
```bash
# Get all items
curl "http://localhost:8080/api/data/items?limit=100" | jq .

# Get items for a specific provider
curl "http://localhost:8080/api/data/items?provider_id=10020084&limit=50" | jq .

# Get items by category and city
curl "http://localhost:8080/api/data/items?category_id=1&city=std:020" | jq .
```

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

---

### 4. Get Statistics

**Endpoint:** `GET /api/data/stats`

**Example:**
```bash
curl http://localhost:8080/api/data/stats | jq .
```

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

---

## Comparison: JSONL API vs Trino API

| Feature | JSONL API (`/api/data`) | Trino API (`/api/trino`) |
|---------|------------------------|--------------------------|
| **Status** | ✅ Works now | ⚠️ Requires Hudi tables |
| **Data Source** | JSONL files | Hudi tables via Trino |
| **Setup Required** | None | Hudi tables + Trino config |
| **Query Language** | REST API | SQL |
| **Performance** | Fast for small datasets | Optimized for large datasets |
| **Use Case** | Current data access | Analytics, complex queries |

---

## Quick Test

```bash
# Test all endpoints
curl http://localhost:8080/api/data/stats | jq .
curl "http://localhost:8080/api/data/providers?limit=5" | jq .
curl http://localhost:8080/api/data/providers/10020084 | jq .
curl "http://localhost:8080/api/data/items?limit=10" | jq .
```

---

## Notes

- **JSONL API** works immediately with current data files
- **Trino API** requires Hudi tables to be set up first
- Both APIs are available - use JSONL API for now, Trino API when tables are ready

