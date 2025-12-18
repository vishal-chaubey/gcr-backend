package projections

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/redis/go-redis/v9"

	"gcr-backend/internal/kstream"
	"gcr-backend/internal/model"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ConsumeAcceptedTopic runs all three projectors (Index, Shard, Delta) that
// consume from catalog.accepted and update Redis read models.
func ConsumeAcceptedTopic(ctx context.Context) error {
	// redis/go-redis/v9: NewClient creates a Redis client connection.
	// Used to store read-optimized projections (Index, Shard, Delta) for fast buyer queries.
	rdb := redis.NewClient(&redis.Options{
		Addr: getenv("REDIS_ADDR", "redis:6379"),
	})
	defer rdb.Close()

	reader := kstream.KafkaReader("catalog.accepted", "projectors-group")
	defer reader.Close()

	log.Println("Projectors: consuming from catalog.accepted")

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			return err
		}

		var evt model.CatalogAccepted
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			log.Printf("Projectors: failed to unmarshal: %v", err)
			continue
		}

		// Update Index (city:category → sellers)
		if err := UpdateIndex(ctx, rdb, evt); err != nil {
			log.Printf("Index Projector error: %v", err)
		}

		// Update Full Shard (seller×city×category snapshot)
		if err := UpdateShard(ctx, rdb, evt); err != nil {
			log.Printf("Shard Projector error: %v", err)
		}

		// Update Delta (short-TTL diff)
		if err := UpdateDelta(ctx, rdb, evt); err != nil {
			log.Printf("Delta Projector error: %v", err)
		}
	}
}

