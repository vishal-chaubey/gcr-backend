package policy

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// PolicyStatus represents the authorization status for a buyer×seller×domain×city combination.
type PolicyStatus string

const (
	PolicyUnknown PolicyStatus = "unknown"
	PolicyAllowed PolicyStatus = "allowed"
	PolicyDenied  PolicyStatus = "denied"
)

// Service provides buyer/seller authorization checks.
type Service struct {
	rdb *redis.Client
}

// NewService creates a new Policy Service backed by Redis.
// Uses redis/go-redis/v9 for buyer×seller authorization caching.
func NewService() *Service {
	// redis/go-redis/v9: NewClient creates Redis client for Policy Service.
	// Stores policy status (allowed/denied/unknown) for fast authorization checks.
	rdb := redis.NewClient(&redis.Options{
		Addr: getenv("REDIS_ADDR", "redis:6379"),
	})
	return &Service{rdb: rdb}
}

// CheckPolicy returns the policy status for {buyer, seller, domain, city}.
func (s *Service) CheckPolicy(ctx context.Context, buyerID, sellerID, domain, city string) (PolicyStatus, error) {
	key := fmt.Sprintf("policy:%s:%s:%s:%s", buyerID, sellerID, domain, city)
	// redis/go-redis/v9: Get retrieves policy status from Redis.
	// redis.Nil indicates key doesn't exist (unknown policy).
	val, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return PolicyUnknown, nil
	}
	if err != nil {
		return PolicyUnknown, err
	}

	switch val {
	case "allowed":
		return PolicyAllowed, nil
	case "denied":
		return PolicyDenied, nil
	default:
		return PolicyUnknown, nil
	}
}

// SetPolicy sets the policy status for {buyer, seller, domain, city}.
func (s *Service) SetPolicy(ctx context.Context, buyerID, sellerID, domain, city string, status PolicyStatus) error {
	key := fmt.Sprintf("policy:%s:%s:%s:%s", buyerID, sellerID, domain, city)
	// redis/go-redis/v9: Set stores policy status (allowed/denied) in Redis.
	// TTL=0 means no expiration. Used for buyer×seller authorization caching.
	return s.rdb.Set(ctx, key, string(status), 0).Err()
}

