package projections

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"

	"gcr-backend/internal/model"
)

// UpdateIndex updates the Redis Index (city:category → sellers) when a CatalogAccepted event arrives.
func UpdateIndex(ctx context.Context, rdb *redis.Client, evt model.CatalogAccepted) error {
	key := fmt.Sprintf("idx:%s:%s", evt.City, evt.Category)

	// redis/go-redis/v9: SAdd adds member to Redis Set. Used for Index: city:category → sellers.
	// Sets provide O(1) membership checks for fast discovery queries.
	if err := rdb.SAdd(ctx, key, evt.SellerID).Err(); err != nil {
		return err
	}

	// redis/go-redis/v9: ZAdd adds member to Redis Sorted Set (ZSET) with score.
	// Used for freshness tracking: sellers sorted by update timestamp.
	freshKey := fmt.Sprintf("freshness:%s:%s", evt.City, evt.Category)
	score := float64(0)
	if err := rdb.ZAdd(ctx, freshKey, redis.Z{Score: score, Member: evt.SellerID}).Err(); err != nil {
		return err
	}

	log.Printf("Index Projector: updated %s with seller %s", key, evt.SellerID)
	return nil
}

