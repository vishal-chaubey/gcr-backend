# GCR Backend - Architecture & Code Summary

## 🎯 What This System Does

The **Global Catalog Repository (GCR) Backend** is a high-performance catalog ingestion and discovery system for ONDC (Open Network for Digital Commerce). It receives product catalogs from sellers, validates them, stores them, and makes them searchable for buyers.

**In Simple Terms:**
- **Sellers** send their product catalogs → System validates and stores them
- **Buyers** search for products → System returns matching sellers and products
- Handles **thousands of concurrent requests** with **millions of items**

---

## 🏗️ System Architecture

### High-Level Flow

```
Seller BPP → [Edge API] → [Kafka] → [SchemaGate] → [Curated Writer] → [Hudi Storage]
                                                      ↓
                                              [Projectors] → [Redis] → [Discovery API] → Buyer BAP
```

### Key Principles

1. **CQRS (Command Query Responsibility Segregation)**
   - **Write Side (Command)**: Ingest catalogs, validate, store
   - **Read Side (Query)**: Fast search and retrieval from Redis

2. **Event-Driven Architecture**
   - Uses Kafka for asynchronous message processing
   - Components communicate via events

3. **Partial Acceptance**
   - Bad providers don't block good ones
   - Bad items don't block good items in the same provider

---

## 📦 Core Components

### 1. **Edge API** (`internal/httpapi/routes.go`)
**What it does:**
- Receives `/on_search` requests from seller BPPs
- Handles gzip compression/decompression
- Validates ONDC envelope structure
- Publishes to Kafka `catalog.ingest` topic

**Key Features:**
- Supports compressed payloads (up to 100MB)
- Fast response (fire-and-forget to Kafka)

### 2. **SchemaGate** (`internal/schemagate/validator.go`)
**What it does:**
- Validates provider and item schemas
- Implements **parallel processing** for high throughput
- Uses **Bloom filter** for item deduplication
- Discards invalid providers/items, keeps valid ones

**Validation Rules:**
- **Provider Level**: If provider schema is invalid → discard entire provider
- **Item Level**: If item schema is invalid → discard only that item (keep provider)
- **Deduplication**: If item already exists → skip it (don't reject)

**Performance:**
- Processes providers in parallel (16 workers)
- Processes items in batches of 100 (32 workers)
- Can handle 50 providers × 500 items each = 25,000 items efficiently

### 3. **Curated Writer** (`internal/curated/writer.go`)
**What it does:**
- Writes validated providers to Hudi storage (currently JSONL stub)
- Creates `CatalogAccepted` events for each provider+category
- Publishes events to Kafka `catalog.accepted` topic

**Output:**
- JSONL files: `data/hudi/providers/{provider_id}.jsonl`
- Contains only valid, non-duplicate items

### 4. **Projectors** (`internal/projections/`)
**What they do:**
- Consume `CatalogAccepted` events from Kafka
- Build read-optimized views in Redis

**Three Types:**

#### a) **Index Projector** (`index.go`)
- Creates: `idx:{city}:{category}` → Set of seller IDs
- Used for: Fast discovery of sellers by location and category

#### b) **Shard Projector** (`shard.go`)
- Creates: `shard:{seller}:{city}:cat:{category}` → Full catalog JSON
- Used for: Returning complete catalog data to buyers

#### c) **Delta Projector** (`delta.go`)
- Creates: `delta:{seller}:{city}:{cat}:{tC}` → Change diff
- Used for: Incremental updates (with TTL)

### 5. **Discovery API** (`internal/discovery/api.go`)
**What it does:**
- Handles buyer search requests (`/ondc/search`)
- Queries Redis Index to find sellers
- Applies policy filters (buyer×seller authorization)
- Returns seller list

**Endpoints:**
- `POST /ondc/search` - Find sellers by category/city
- `GET /ondc/on_search` - Get full catalog for a seller

### 6. **Bloom Filter** (`internal/bloom/bloom.go`)
**What it does:**
- Fast duplicate detection for items
- Uses RedisBloom module
- Two filters:
  - `gcr:providers` - Provider deduplication (1M capacity)
  - `gcr:items` - Item deduplication (10M capacity)

**How it works:**
- Item key: `domain:city:provider_id:item_id`
- Returns `true` if item probably exists (may have false positives)
- O(1) lookup time, very fast

### 7. **Storage** (`internal/storage/hudi_stub.go`)
**What it does:**
- Writes provider data to JSONL files (Phase 1 stub)
- In production: Would write to Apache Hudi (data lake format)
- Hudi enables: Time travel queries, incremental processing, upserts

---

## 🔄 Data Flow (Step by Step)

### **Ingest Flow** (Seller → Storage)

1. **Seller sends catalog** → `POST /ondc/on_search` (gzip compressed, up to 100MB)
2. **Edge API** → Decompresses, validates envelope, publishes to Kafka `catalog.ingest`
3. **SchemaGate Consumer** → 
   - Validates providers in parallel (16 workers)
   - For each valid provider:
     - Validates items in parallel batches (32 workers, 100 items/batch)
     - Checks Bloom filter for duplicates
     - Keeps only valid, non-duplicate items
4. **Curated Writer** → 
   - Writes to Hudi (JSONL files)
   - Publishes `CatalogAccepted` events to Kafka `catalog.accepted`
5. **Projectors** → 
   - Index: Updates `idx:{city}:{category}` with seller IDs
   - Shard: Updates `shard:{seller}:{city}:cat:{category}` with full JSON
   - Delta: Updates `delta:{seller}:{city}:{cat}:{tC}` with changes

### **Discovery Flow** (Buyer → Results)

1. **Buyer searches** → `POST /ondc/search` {domain, city, category}
2. **Discovery API** → 
   - Queries Redis Index: `idx:{city}:{category}`
   - Gets list of seller IDs
   - Applies policy filters (authorization)
3. **Returns** → List of matching sellers
4. **Buyer requests catalog** → `GET /ondc/on_search?seller_id=...&city=...&category=...`
5. **Discovery API** → 
   - Reads Redis Shard: `shard:{seller}:{city}:cat:{category}`
   - Returns gzip-compressed catalog JSON

---

## 🚀 Performance Optimizations

### 1. **Parallel Processing**
- **Providers**: 16 parallel workers
- **Items**: 32 workers processing batches of 100
- Handles large catalogs (50 providers × 500 items) efficiently

### 2. **Bloom Filter Deduplication**
- O(1) duplicate detection
- No database lookups needed
- 10M item capacity

### 3. **Kafka for Async Processing**
- Non-blocking ingestion
- Handles bursts of traffic
- 100MB message size support

### 4. **Redis for Fast Reads**
- In-memory storage for hot data
- Pre-computed indexes and shards
- Sub-millisecond query times

### 5. **CQRS Pattern**
- Write and read paths separated
- Read path optimized for speed
- Write path optimized for validation and storage

---

## 📊 Data Structures

### **Redis Keys**

```
# Index: Fast seller lookup
idx:{city}:{category} → Set of seller IDs

# Shard: Full catalog data
shard:{seller}:{city}:cat:{category} → JSON string

# Delta: Incremental changes
delta:{seller}:{city}:{cat}:{tC} → JSON diff (with TTL)

# Policy: Authorization
policy:{buyer}:{seller}:{domain}:{city} → Allow/Deny

# Bloom Filters
gcr:providers → Bloom filter (1M capacity)
gcr:items → Bloom filter (10M capacity)
```

### **Kafka Topics**

```
catalog.ingest → Raw on_search envelopes from Edge
catalog.accepted → CatalogAccepted events after validation
```

### **Storage Files**

```
data/hudi/providers/{provider_id}.jsonl → Provider data (Hudi stub)
data/rejections/rejections_{date}.jsonl → Rejected scopes with reasons
```

---

## 🔍 Key Code Files Explained

### **Main Entry Point**
- `cmd/gcr-api/main.go`
  - Starts HTTP server
  - Initializes Bloom filters
  - Starts Kafka consumers (SchemaGate, Projectors)
  - Registers routes

### **Validation Logic**
- `internal/schemagate/validator.go`
  - `ProcessCatalog()` - Main validation orchestrator
  - `processProvider()` - Validates provider schema
  - `processItemsParallel()` - Parallel item validation
  - `processItemBatch()` - Batch processing with Bloom filter

### **Kafka Integration**
- `internal/kstream/producer.go` - Publishes to Kafka
- `internal/kstream/consumer.go` - Consumes from Kafka

### **Data Models**
- `internal/model/ondc.go` - ONDC schema structures (Provider, Item, etc.)
- `internal/model/events.go` - Event structures (CatalogAccepted)

---

## 🎯 Business Logic

### **What Gets Rejected?**

1. **Provider Rejected** (entire provider discarded):
   - Missing provider.id
   - Missing provider.descriptor.name
   - Empty categories array

2. **Item Rejected** (only that item discarded):
   - Missing item.id
   - Missing item.descriptor.name
   - Missing item.category_id
   - Missing item.price.currency or item.price.value

3. **Item Skipped** (not rejected, just skipped):
   - Item already exists (Bloom filter detects duplicate)

### **What Gets Accepted?**

- Providers with valid schema
- Items with valid schema
- Items that are not duplicates
- Result: Clean, validated, deduplicated catalog

---

## 🛠️ Technology Stack

- **Language**: Go 1.22+
- **Message Queue**: Kafka (via segmentio/kafka-go)
- **Cache/Storage**: Redis (with RedisBloom module)
- **Data Lake**: Apache Hudi (stub: JSONL, production: Parquet)
- **Query Engine**: Trino (for Hudi tables)
- **Processing**: Spark (for Hudi jobs)
- **Storage**: MinIO (S3-compatible for Hudi)
- **Containerization**: Docker & Docker Compose

---

## 📈 Scalability Features

1. **Horizontal Scaling**: Stateless API, can run multiple instances
2. **Kafka Consumer Groups**: Multiple workers can process in parallel
3. **Redis Clustering**: Can scale Redis for larger datasets
4. **Batch Processing**: Items processed in optimized batches
5. **Async Processing**: Non-blocking ingestion pipeline

---

## 🔐 Security & Compliance

- **Policy Service**: Buyer×Seller authorization
- **Schema Validation**: Ensures data quality
- **Rejection Tracking**: All rejections logged for audit
- **TTL on Deltas**: Automatic cleanup of temporary data

---

## 📝 Summary

**In One Sentence:**
This system receives large product catalogs from sellers, validates them in parallel (discarding invalid providers/items and duplicates), stores them efficiently, and provides fast search capabilities for buyers using Redis indexes and pre-computed shards.

**Key Strengths:**
- ✅ Handles large payloads (100MB+)
- ✅ Parallel processing for high throughput
- ✅ Smart deduplication with Bloom filters
- ✅ Fast reads via Redis
- ✅ Partial acceptance (bad data doesn't block good data)
- ✅ Event-driven, scalable architecture

