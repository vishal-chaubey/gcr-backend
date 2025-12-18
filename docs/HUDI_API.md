# Hudi Data API Documentation

This API provides direct access to Hudi data stored in JSONL files. This is a dedicated API specifically for querying Hudi data.

## Base URL
```
http://localhost:8080/api/hudi
```

---

## Endpoints

### 1. Health Check

Check if Hudi data is available and get a quick summary.

**Endpoint:** `GET /api/hudi/health`

**Response:**
```json
{
  "status": "ok",
  "message": "Hudi data available: 60 providers, 92936 items"
}
```

**Example:**
```bash
curl http://localhost:8080/api/hudi/health | jq .
```

---

### 2. Get Statistics

Get overall statistics about the Hudi data.

**Endpoint:** `GET /api/hudi/stats`

**Response:**
```json
{
  "success": true,
  "data": {
    "total_providers": 60,
    "total_records": 60,
    "total_items": 92936,
    "total_domains": 1,
    "total_cities": 1,
    "latest_update": "2025-12-18T19:11:18.876966883Z"
  }
}
```

**Example:**
```bash
curl http://localhost:8080/api/hudi/stats | jq .
```

---

### 3. Get All Providers

Get a list of all providers from Hudi data with optional filtering.

**Endpoint:** `GET /api/hudi/providers`

**Query Parameters:**
- `limit` (optional): Number of results (default: 100, max: 1000)
- `offset` (optional): Pagination offset (default: 0)
- `city` (optional): Filter by city (e.g., `std:020`)
- `domain` (optional): Filter by domain (e.g., `ONDC:RET11`)

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
      "bap_id": "buyer-backend.himira.co.in",
      "timestamp": "2025-12-18T19:11:18.876966883Z",
      "provider_name": "Abhijeet",
      "items_count": 1548,
      "categories": [...]
    }
  ],
  "count": 1,
  "limit": 100,
  "offset": 0,
  "has_more": false
}
```

**Examples:**
```bash
# Get first 10 providers
curl "http://localhost:8080/api/hudi/providers?limit=10" | jq .

# Get providers with pagination
curl "http://localhost:8080/api/hudi/providers?limit=50&offset=50" | jq .

# Filter by city
curl "http://localhost:8080/api/hudi/providers?city=std:020" | jq .

# Filter by domain
curl "http://localhost:8080/api/hudi/providers?domain=ONDC:RET11" | jq .
```

---

### 4. Get Provider by ID

Get detailed information about a specific provider including all its items and categories.

**Endpoint:** `GET /api/hudi/providers/{provider_id}`

**Response:**
```json
{
  "success": true,
  "data": {
    "provider_id": "10020084",
    "domain": "ONDC:RET11",
    "city": "std:020",
    "bap_id": "buyer-backend.himira.co.in",
    "bpp_id": "webapi.magicpin.in/oms_partner/ondc",
    "timestamp": "2025-12-18T19:11:18.876966883Z",
    "descriptor": {
      "name": "Abhijeet",
      ...
    },
    "categories": [...],
    "items": [...]
  }
}
```

**Example:**
```bash
curl http://localhost:8080/api/hudi/providers/10020084 | jq .
```

---

### 5. Get Items

Get items from Hudi data with optional filters.

**Endpoint:** `GET /api/hudi/items`

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
  ],
  "count": 1,
  "limit": 100,
  "has_more": false
}
```

**Examples:**
```bash
# Get all items (first 100)
curl "http://localhost:8080/api/hudi/items?limit=100" | jq .

# Get items for a specific provider
curl "http://localhost:8080/api/hudi/items?provider_id=10020084&limit=50" | jq .

# Get items by category
curl "http://localhost:8080/api/hudi/items?category_id=F&B&limit=20" | jq .

# Get items by city
curl "http://localhost:8080/api/hudi/items?city=std:020&limit=50" | jq .
```

---

### 6. Get Provider Items

Get all items for a specific provider (convenience endpoint).

**Endpoint:** `GET /api/hudi/provider/{provider_id}/items`

**Query Parameters:**
- `limit` (optional): Number of results (default: 1000, max: 10000)

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
  ],
  "count": 1548,
  "provider_id": "10020084"
}
```

**Example:**
```bash
# Get all items for provider 10020084
curl "http://localhost:8080/api/hudi/provider/10020084/items" | jq .

# Get first 10 items
curl "http://localhost:8080/api/hudi/provider/10020084/items?limit=10" | jq .
```

---

## Data Source

This API reads directly from JSONL files stored in `data/hudi/providers/*.jsonl`. These files represent the Hudi data structure and contain:

- **Provider information**: ID, domain, city, BAP/BPP IDs, timestamp
- **Descriptor**: Provider name and details
- **Categories**: List of categories the provider serves
- **Items**: List of items (products/services) with details like price, category, etc.

Each JSONL file is named after the provider ID (e.g., `10020084.jsonl`) and contains one JSON object per line, representing different versions/updates of the provider data.

---

## Comparison with Other APIs

### Hudi API (`/api/hudi/*`) vs JSONL API (`/api/data/*`)
- **Hudi API**: Dedicated API specifically for Hudi data, provides more detailed provider information
- **JSONL API**: General-purpose API for querying JSONL files, simpler response format

### Hudi API vs Trino API (`/api/trino/*`)
- **Hudi API**: Reads directly from JSONL files (works immediately)
- **Trino API**: Requires Hudi tables to be set up in Trino (for SQL queries)

---

## Notes

- All timestamps are in UTC (RFC3339Nano format)
- Items are filtered during ingestion (only valid, non-duplicate items are stored)
- Provider data is stored in JSONL format (one JSON object per line)
- The API supports pagination for large result sets
- Filtering is done in-memory after reading from files (for performance, consider using Trino API once Hudi tables are set up)

