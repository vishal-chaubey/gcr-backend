package projections

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"

	"gcr-backend/internal/model"
)

// UpdateShard builds a ready-to-send /on_search JSON for seller×city×category
// and stores it in Redis as a Full Shard.
func UpdateShard(ctx context.Context, rdb *redis.Client, evt model.CatalogAccepted) error {
	key := fmt.Sprintf("shard:%s:%s:cat:%s", evt.SellerID, evt.City, evt.Category)

	shard := map[string]any{
		"seller_id":  evt.SellerID,
		"city":       evt.City,
		"category":   evt.Category,
		"provider_id": evt.ProviderID,
		"timestamp":  evt.Timestamp,
	}

	data, err := json.Marshal(shard)
	if err != nil {
		return err
	}

	// redis/go-redis/v9: Set stores ready-to-send /on_search JSON as string value.
	// TTL=0 means no expiration. Used for Full Shard: seller×city×category snapshot.
	if err := rdb.Set(ctx, key, data, 0).Err(); err != nil {
		return err
	}

	log.Printf("Shard Projector: updated %s", key)
	return nil
}

