package discovery

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"

	"gcr-backend/internal/model"
	"gcr-backend/internal/policy"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Service provides Discovery/Publisher functionality for buyer /search and /on_search.
type Service struct {
	rdb    *redis.Client
	policy *policy.Service
}

// NewService creates a new Discovery Service.
func NewService() *Service {
	rdb := redis.NewClient(&redis.Options{
		Addr: getenv("REDIS_ADDR", "redis:6379"),
	})
	return &Service{
		rdb:    rdb,
		policy: policy.NewService(),
	}
}

// RegisterRoutes wires Discovery API routes.
// gorilla/mux: Router handles buyer-facing /search and /on_search endpoints.
func (s *Service) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/ondc/search", s.searchHandler).Methods("POST")
	r.HandleFunc("/ondc/on_search", s.onSearchReadHandler).Methods("GET")
}

// searchHandler handles buyer /search requests.
// It queries Redis Index for candidates, checks Policy, and returns seller list.
func (s *Service) searchHandler(w http.ResponseWriter, r *http.Request) {
	var req model.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	city := req.Context.City
	category := req.Message.Intent.Item.Category.ID
	buyerID := req.Context.BapID
	domain := req.Context.Domain

	// redis/go-redis/v9: SMembers retrieves all members from Redis Set.
	// Returns list of seller IDs for given city:category from Index projection.
	indexKey := fmt.Sprintf("idx:%s:%s", city, category)
	sellers, err := s.rdb.SMembers(ctx, indexKey).Result()
	if err != nil {
		http.Error(w, "index lookup failed", http.StatusInternalServerError)
		return
	}

	// Filter by Policy (allowed only)
	allowedSellers := []string{}
	for _, sellerID := range sellers {
		status, err := s.policy.CheckPolicy(ctx, buyerID, sellerID, domain, city)
		if err != nil {
			log.Printf("Policy check error: %v", err)
			continue
		}
		if status == policy.PolicyAllowed {
			allowedSellers = append(allowedSellers, sellerID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "gzip")
	gw := gzip.NewWriter(w)
	defer gw.Close()
	_ = json.NewEncoder(gw).Encode(map[string]any{
		"sellers": allowedSellers,
		"city":    city,
		"category": category,
	})
}

// onSearchReadHandler returns a ready-to-send /on_search JSON for a specific seller.
// It reads from Redis Shard (overlay-first if exists).
func (s *Service) onSearchReadHandler(w http.ResponseWriter, r *http.Request) {
	sellerID := r.URL.Query().Get("seller_id")
	city := r.URL.Query().Get("city")
	category := r.URL.Query().Get("category")
	buyerID := r.URL.Query().Get("buyer_id") // optional, for overlay lookup

	if sellerID == "" || city == "" || category == "" {
		http.Error(w, "missing required params: seller_id, city, category", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// redis/go-redis/v9: Get retrieves string value from Redis.
	// First tries overlay shard (buyer-specific), then falls back to base shard.
	var shardKey string
	if buyerID != "" {
		shardKey = fmt.Sprintf("overlay:%s:%s:%s:cat:%s", buyerID, sellerID, city, category)
		val, err := s.rdb.Get(ctx, shardKey).Result()
		if err == nil {
			// Overlay found, return it
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")
			gw := gzip.NewWriter(w)
			defer gw.Close()
			_, _ = gw.Write([]byte(val))
			return
		}
	}

	// redis/go-redis/v9: Get retrieves base shard JSON (seller×city×category).
	// This is the ready-to-send /on_search payload stored by Shard Projector.
	shardKey = fmt.Sprintf("shard:%s:%s:cat:%s", sellerID, city, category)
	val, err := s.rdb.Get(ctx, shardKey).Result()
	if err != nil {
		http.Error(w, "shard not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "gzip")
	gw := gzip.NewWriter(w)
	defer gw.Close()
	_, _ = gw.Write([]byte(val))
}

