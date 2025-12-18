# Quick Start Guide

## Terminal 1: Start Services

```bash
cd /Users/vishalkumarchaubey/Desktop/GCR/gcr-backend
docker compose up -d
```

Wait 10-15 seconds for services to initialize, then check status:

```bash
docker compose ps
```

## Terminal 2: Monitor Logs (Your Mac Bash)

Open a **NEW terminal window** on your Mac and run:

```bash
cd /Users/vishalkumarchaubey/Desktop/GCR/gcr-backend

# Option 1: Monitor all API logs
docker compose logs -f api

# Option 2: Monitor only validation/processing (recommended)
./monitor.sh validation

# Option 3: Monitor everything
./monitor.sh all
```

## Terminal 3: Run Tests

Open **another terminal** and run:

```bash
cd /Users/vishalkumarchaubey/Desktop/GCR/gcr-backend

# Run full test flow
./test-flow.sh

# Or test individual endpoints
curl http://localhost:8080/health
```

## What to Watch in Terminal 2

You'll see:
- ✅ **Provider validation**: "SchemaGate: rejected provider..." or providers being accepted
- ✅ **Item validation**: "SchemaGate: rejected item..." or items being processed
- ✅ **Duplicate detection**: "SchemaGate: duplicate item..., skipping"
- ✅ **Kafka processing**: Messages being published/consumed
- ✅ **Redis updates**: Index and shard updates
- ✅ **Processing stats**: Duration and counts

## Stop Services

```bash
docker compose down
```

## Restart After Code Changes

```bash
docker compose build api
docker compose up -d api
```

