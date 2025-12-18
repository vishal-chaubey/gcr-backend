package bloom

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	client *redis.Client
	once   sync.Once
)

const (
	// bloomKey is the RedisBloom filter key for provider uniqueness.
	bloomKey = "gcr:providers"
	// bloomItemsKey is the RedisBloom filter key for item deduplication.
	bloomItemsKey = "gcr:items"
)

// Init sets up the Redis client and ensures the Bloom filter exists.
// It is safe to call multiple times.
func Init() {
	once.Do(func() {
		addr := getenv("REDIS_ADDR", "redis:6379")
		client = redis.NewClient(&redis.Options{
			Addr: addr,
		})

		// RedisBloom (redis/go-redis/v9): Reserve Bloom filter with error rate 0.001 and capacity 1M.
		// BF.RESERVE creates a probabilistic data structure for duplicate detection.
		// This uses RedisBloom module (via redis-stack-server) - not standard Redis commands.
		ctx := context.Background()
		if err := client.Do(ctx, "BF.RESERVE", bloomKey, 0.001, 1_000_000).Err(); err != nil {
			log.Printf("bloom: reserve providers (may already exist): %v", err)
		}
		// Reserve Bloom filter for items with larger capacity (10M) to handle high item volume
		if err := client.Do(ctx, "BF.RESERVE", bloomItemsKey, 0.001, 10_000_000).Err(); err != nil {
			log.Printf("bloom: reserve items (may already exist): %v", err)
		}
	})
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// SeenProvider returns true if this provider key looks like a duplicate
// according to the Bloom filter. It also adds the key if not seen before.
func SeenProvider(ctx context.Context, providerKey string) bool {
	if client == nil {
		return false
	}
	// RedisBloom (redis/go-redis/v9): BF.ADD adds item to Bloom filter.
	// Returns 1 if item was new, 0 if it probably existed (false positive possible).
	// Uses RedisBloom module commands via client.Do().
	res := client.Do(ctx, "BF.ADD", bloomKey, providerKey)
	if res.Err() != nil {
		log.Printf("bloom: BF.ADD error: %v", res.Err())
		return false
	}
	// BF.ADD can return either int (0/1) or bool (true/false) depending on Redis version
	// Try int first, then bool
	val, err := res.Int()
	if err != nil {
		// Try as bool instead
		boolVal, boolErr := res.Bool()
		if boolErr != nil {
			log.Printf("bloom: BF.ADD type error (not int or bool): %v", err)
			return false
		}
		// Bool: true means item was new (not seen), false means probably existed
		return !boolVal
	}
	// Int: 1 means item was new (not seen), 0 means probably existed
	return val == 0
}

// SeenItem returns true if this item key looks like a duplicate
// according to the Bloom filter. It also adds the key if not seen before.
// Item key format: "domain:city:provider_id:item_id"
func SeenItem(ctx context.Context, itemKey string) bool {
	if client == nil {
		return false
	}
	// RedisBloom (redis/go-redis/v9): BF.ADD adds item to Bloom filter.
	// Returns 1 if item was new, 0 if it probably existed (false positive possible).
	res := client.Do(ctx, "BF.ADD", bloomItemsKey, itemKey)
	if res.Err() != nil {
		log.Printf("bloom: BF.ADD item error: %v", res.Err())
		return false
	}
	// BF.ADD can return either int (0/1) or bool (true/false) depending on Redis version
	// Try int first, then bool
	val, err := res.Int()
	if err != nil {
		// Try as bool instead
		boolVal, boolErr := res.Bool()
		if boolErr != nil {
			log.Printf("bloom: BF.ADD item type error (not int or bool): %v", err)
			return false
		}
		// Bool: true means item was new (not seen), false means probably existed
		return !boolVal
	}
	// Int: 1 means item was new (not seen), 0 means probably existed
	return val == 0
}


