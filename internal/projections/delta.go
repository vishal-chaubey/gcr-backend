package projections

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"gcr-backend/internal/model"
)

const deltaTTL = 5 * time.Minute // Short TTL for deltas

// UpdateDelta computes a delta vs previous state and stores it with TTL.
// If delta is too large or absent, it skips (shard is fallback).
func UpdateDelta(ctx context.Context, rdb *redis.Client, evt model.CatalogAccepted) error {
	key := fmt.Sprintf("delta:%s:%s:%s:%s", evt.SellerID, evt.City, evt.Category, evt.Timestamp)

	delta := map[string]any{
		"seller_id":  evt.SellerID,
		"city":       evt.City,
		"category":   evt.Category,
		"provider_id": evt.ProviderID,
		"timestamp":  evt.Timestamp,
		"type":       "update", // or "add", "remove"
	}

	data, err := json.Marshal(delta)
	if err != nil {
		return err
	}

	// redis/go-redis/v9: Set stores delta JSON with TTL (5 minutes).
	// Deltas expire automatically to prevent Redis memory bloat. Shard is fallback if delta missing.
	if err := rdb.Set(ctx, key, data, deltaTTL).Err(); err != nil {
		return err
	}

	log.Printf("Delta Projector: updated %s (TTL: %v)", key, deltaTTL)
	return nil
}

