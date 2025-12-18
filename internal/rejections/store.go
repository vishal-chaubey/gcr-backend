package rejections

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gcr-backend/internal/schemagate"
)

// WriteRejection appends a rejection record to the durable rejections store.
func WriteRejection(ctx context.Context, envMeta map[string]string, rejection schemagate.Rejection) error {
	dir := "./data/rejections"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	fpath := filepath.Join(dir, fmt.Sprintf("rejections_%s.jsonl", time.Now().Format("2006-01-02")))
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	record := map[string]any{
		"scope":       rejection.Scope,
		"reason":      rejection.Reason,
		"transaction_id": envMeta["transaction_id"],
		"message_id":   envMeta["message_id"],
		"timestamp":    time.Now().UTC().Format(time.RFC3339Nano),
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	_, err = f.Write(append(data, '\n'))
	return err
}

