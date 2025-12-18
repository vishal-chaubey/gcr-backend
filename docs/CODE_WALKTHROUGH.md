# Code Walkthrough - What Happens When You Send a Catalog

This document explains **exactly what the code does** when a seller sends a product catalog.

---

## 📥 Step 1: Seller Sends Catalog

**File:** `internal/httpapi/routes.go`

**What happens:**
1. Seller sends `POST /ondc/on_search` with a large JSON file (can be 100MB, gzip compressed)
2. Code receives the request at `POST /ondc/on_search` endpoint
3. Decompresses if gzip-compressed
4. Validates basic ONDC envelope structure
5. **Immediately publishes to Kafka** (doesn't wait for processing)
6. Returns response: `{"providers": 5, "duration_ms": 10}`

**Key Code:**
```go
// Receives request, decompresses, validates
// Then publishes to Kafka topic "catalog.ingest"
kstream.PublishOnSearchIngest(ctx, env)
```

**Why fast?** It's "fire-and-forget" - doesn't wait for validation.

---

## 🔍 Step 2: SchemaGate Validates (Parallel Processing)

**File:** `internal/kstream/consumer.go` → `internal/schemagate/validator.go`

**What happens:**
1. **SchemaGate consumer** reads from Kafka `catalog.ingest` topic
2. Calls `schemagate.ProcessCatalog()` to validate

**Inside ProcessCatalog():**

### 2a. Provider Validation (16 parallel workers)
```go
// For each provider in the catalog:
for each provider {
    if provider.id is missing → REJECT entire provider
    if provider.descriptor.name is missing → REJECT entire provider
    if provider.categories is empty → REJECT entire provider
    else → ACCEPT provider (continue to items)
}
```

**Key Point:** If provider fails → **entire provider discarded** (all items lost)

### 2b. Item Validation (32 workers, batches of 100)
```go
// For each item in valid providers:
for each item {
    // Validate schema
    if item.id is missing → REJECT only this item
    if item.descriptor.name is missing → REJECT only this item
    if item.category_id is missing → REJECT only this item
    if item.price is missing → REJECT only this item
    
    // Check for duplicates using Bloom filter
    itemKey = "domain:city:provider_id:item_id"
    if bloom.SeenItem(itemKey) → SKIP (duplicate, already in DB)
    
    else → ACCEPT item (add to valid items list)
}
```

**Key Points:**
- If item fails → **only that item discarded** (other items in provider kept)
- If duplicate → **item skipped** (not rejected, just not added again)
- **Parallel processing:** 32 workers process items in batches of 100

**Result:** Provider with only valid, non-duplicate items

---

## 💾 Step 3: Curated Writer Stores Data

**File:** `internal/curated/writer.go` → `internal/storage/hudi_stub.go`

**What happens:**
1. For each valid provider (with filtered items):
   - Writes to file: `data/hudi/providers/{provider_id}.jsonl`
   - File contains: provider info + only valid items
2. Creates `CatalogAccepted` event for each provider+category
3. Publishes events to Kafka `catalog.accepted` topic

**Key Code:**
```go
// Write provider with filtered items
storage.WriteProviderCatalog(ctx, env.Context, provider)

// Create event
CatalogAccepted {
    SellerID: "webapi.magicpin.in/oms_partner/ondc",
    City: "std:020",
    Category: "1",
    ProviderID: "10020084"
}
```

---

## 📊 Step 4: Projectors Build Read Views

**File:** `internal/projections/consumer.go`

**What happens:**
1. **Projectors consumer** reads from Kafka `catalog.accepted` topic
2. For each `CatalogAccepted` event:

### 4a. Index Projector (`internal/projections/index.go`)
```go
// Updates Redis: idx:{city}:{category}
// Example: idx:std:020:1 → Set of seller IDs
redis.SADD("idx:std:020:1", "webapi.magicpin.in/oms_partner/ondc")
```
**Purpose:** Fast lookup - "Which sellers have category 1 in city std:020?"

### 4b. Shard Projector (`internal/projections/shard.go`)
```go
// Updates Redis: shard:{seller}:{city}:cat:{category}
// Example: shard:webapi.magicpin.in/oms_partner/ondc:std:020:cat:1
// Stores: Full catalog JSON for this seller+city+category
redis.SET("shard:...", fullCatalogJSON)
```
**Purpose:** Fast retrieval - "Get full catalog for seller X in city Y, category Z"

### 4c. Delta Projector (`internal/projections/delta.go`)
```go
// Updates Redis: delta:{seller}:{city}:{cat}:{timestamp}
// Stores: What changed (diff)
redis.SET("delta:...", changeDiff, TTL=24h)
```
**Purpose:** Incremental updates - "What changed since last time?"

---

## 🔎 Step 5: Buyer Searches

**File:** `internal/discovery/api.go`

### 5a. Buyer sends search request
```http
POST /ondc/search
{
  "context": {
    "city": "std:020",
    "category": "1"
  }
}
```

**What happens:**
1. Queries Redis Index: `idx:std:020:1`
2. Gets list of seller IDs: `["seller1", "seller2", ...]`
3. Applies policy filter (buyer×seller authorization)
4. Returns seller list

### 5b. Buyer requests catalog
```http
GET /ondc/on_search?seller_id=...&city=std:020&category=1
```

**What happens:**
1. Reads Redis Shard: `shard:{seller}:std:020:cat:1`
2. Gets full catalog JSON
3. Returns gzip-compressed response

---

## 🔑 Key Functions Explained

### `bloom.SeenItem(ctx, itemKey)`
**File:** `internal/bloom/bloom.go`

**What it does:**
- Checks RedisBloom filter: "Have I seen this item before?"
- Item key format: `"domain:city:provider_id:item_id"`
- Returns `true` if probably duplicate (may have false positives)
- Returns `false` if definitely new
- **O(1) operation** - very fast, no database lookup

**Why Bloom filter?**
- Fast duplicate detection without querying database
- Handles millions of items efficiently
- Small memory footprint

### `schemagate.ProcessCatalog(ctx, env)`
**File:** `internal/schemagate/validator.go`

**What it does:**
1. Creates worker pool (16 workers for providers)
2. Each worker validates providers in parallel
3. For valid providers, creates item worker pool (32 workers)
4. Each item worker validates items in batches of 100
5. Uses Bloom filter to check duplicates
6. Returns: valid providers (with filtered items) + rejection list

**Performance:**
- Can process 50 providers × 500 items = 25,000 items efficiently
- Parallel processing ensures fast validation

### `projections.ConsumeAcceptedTopic(ctx)`
**File:** `internal/projections/consumer.go`

**What it does:**
1. Reads `CatalogAccepted` events from Kafka
2. For each event:
   - Updates Index (seller list)
   - Updates Shard (full catalog)
   - Updates Delta (changes)
3. All updates go to Redis for fast reads

---

## 📈 Performance Numbers

**What the code can handle:**
- **Message size:** Up to 100MB (Kafka configured)
- **Providers:** 16 parallel workers
- **Items:** 32 workers, 100 items per batch
- **Throughput:** Thousands of concurrent requests
- **Response time:** 
  - Ingest: ~10-20ms (fire-and-forget)
  - Search: <100ms (Redis lookup)
  - Catalog retrieval: <100ms (Redis shard)

---

## 🎯 Summary Flow Diagram

```
Seller → Edge API → Kafka → SchemaGate → Curated Writer → Hudi
                                    ↓
                              (valid providers
                               with filtered items)
                                    ↓
                            CatalogAccepted Event
                                    ↓
                            Kafka catalog.accepted
                                    ↓
                            Projectors → Redis
                                    ↓
                            (Index + Shard + Delta)
                                    ↓
                            Discovery API → Buyer
```

---

## 🔍 Debugging: What to Look For

**In logs, you'll see:**
- `SchemaGate: rejected provider X: reason` → Provider validation failed
- `SchemaGate: rejected item X: reason` → Item validation failed
- `SchemaGate: duplicate item X, skipping` → Item already exists
- `Curated Writer: wrote provider X` → Provider stored
- `Index: updated idx:std:020:1` → Redis index updated
- `Shard: updated shard:...` → Redis shard updated

**Check Redis:**
```bash
# See which sellers are indexed
redis-cli SMEMBERS idx:std:020:1

# See full catalog for a seller
redis-cli GET shard:webapi.magicpin.in/oms_partner/ondc:std:020:cat:1
```

---

## 💡 Key Design Decisions

1. **Why parallel processing?**
   - Large catalogs (50 providers × 500 items) would be slow sequentially
   - Parallel processing reduces time from minutes to seconds

2. **Why Bloom filter for duplicates?**
   - Database lookup for each item would be too slow
   - Bloom filter is O(1) and handles millions of items

3. **Why separate provider/item validation?**
   - Provider invalid → discard all (makes sense)
   - Item invalid → discard only that item (partial acceptance)
   - Better data quality, less waste

4. **Why Kafka?**
   - Async processing (doesn't block seller)
   - Handles bursts of traffic
   - Decouples components

5. **Why Redis for reads?**
   - Sub-millisecond lookups
   - Pre-computed indexes and shards
   - Fast buyer experience

---

This is what the code does. Every request goes through these steps, optimized for speed and scale.

