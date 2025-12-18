package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gcr-backend/internal/model"
)

// WriteProviderCatalog writes curated provider data in a format ready for Apache Hudi ingestion.
// This is a Phase-1 stub that writes JSONL files. In production, a Spark/Hudi job would:
// 1. Read these JSONL files (or from MinIO/S3)
// 2. Convert to Parquet format
// 3. Write to Hudi MoR (Merge-on-Read) tables with upsert semantics
// 4. Trino can then query these Hudi tables via SQL (SELECT * FROM hudi.default.providers)
func WriteProviderCatalog(_ context.Context, ctxMeta model.OnSearchContext, provider model.Provider) error {
	// Hudi preparation: Write JSONL files that Spark/Hudi will ingest into MoR tables.
	// Hudi MoR tables support upserts, time travel queries, and incremental processing.
	dir := "./data/hudi/providers"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	fpath := filepath.Join(dir, fmt.Sprintf("%s.jsonl", provider.ID))
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	record := map[string]any{
		"provider_id": provider.ID,
		"domain":      ctxMeta.Domain,
		"city":        ctxMeta.City,
		"bap_id":      ctxMeta.BapID,
		"bpp_id":      ctxMeta.BppID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
		"descriptor":  provider.Descriptor,
		"categories":  provider.Categories,
		"items":       provider.Items, // Include filtered items (only valid, non-duplicate items)
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	_, err = f.Write(append(data, '\n'))
	return err
}


