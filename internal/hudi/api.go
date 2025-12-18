package hudi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"gcr-backend/internal/jsonl"
)

// Service provides Hudi data query API
type Service struct {
	queryService *jsonl.QueryService
}

// NewService creates a new Hudi service
func NewService() *Service {
	return &Service{
		queryService: jsonl.NewQueryService(),
	}
}

// RegisterRoutes registers Hudi API routes
func (s *Service) RegisterRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/hudi").Subrouter()
	api.HandleFunc("/health", s.HealthCheck).Methods("GET")
	api.HandleFunc("/providers", s.GetProviders).Methods("GET")
	api.HandleFunc("/providers/{provider_id}", s.GetProvider).Methods("GET")
	api.HandleFunc("/items", s.GetItems).Methods("GET")
	api.HandleFunc("/stats", s.GetStats).Methods("GET")
	api.HandleFunc("/provider/{provider_id}/items", s.GetProviderItems).Methods("GET")
}

// HealthCheckResponse represents health check response
type HealthCheckResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// HealthCheck checks if Hudi data is available
func (s *Service) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	stats, err := s.queryService.GetStats(ctx)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HealthCheckResponse{
			Status:  "error",
			Message: fmt.Sprintf("Hudi data unavailable: %v", err),
		})
		return
	}

	totalProviders := 0
	totalItems := 0
	if tp, ok := stats["total_providers"].(int); ok {
		totalProviders = tp
	}
	if ti, ok := stats["total_items"].(int); ok {
		totalItems = ti
	}

	json.NewEncoder(w).Encode(HealthCheckResponse{
		Status:  "ok",
		Message: fmt.Sprintf("Hudi data available: %d providers, %d items", totalProviders, totalItems),
	})
}

// GetProviders returns all providers from Hudi
func (s *Service) GetProviders(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	city := r.URL.Query().Get("city")
	domain := r.URL.Query().Get("domain")

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	providers, err := s.queryService.GetAllProviders(ctx, limit, offset)
	if err != nil {
		log.Printf("Hudi query error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Apply filters
	filtered := []jsonl.Provider{}
	for _, p := range providers {
		if city != "" && p.City != city {
			continue
		}
		if domain != "" && p.Domain != domain {
			continue
		}
		filtered = append(filtered, p)
	}

	// Convert to response format
	result := []map[string]interface{}{}
	for _, p := range filtered {
		itemCount := len(p.Items)
		providerName := ""
		if desc, ok := p.Descriptor["name"].(string); ok {
			providerName = desc
		}

		result = append(result, map[string]interface{}{
			"provider_id":   p.ProviderID,
			"domain":        p.Domain,
			"city":          p.City,
			"bpp_id":        p.BppID,
			"bap_id":        p.BapID,
			"timestamp":     p.Timestamp,
			"provider_name": providerName,
			"items_count":   itemCount,
			"categories":    p.Categories,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"data":       result,
		"count":      len(result),
		"limit":      limit,
		"offset":     offset,
		"has_more":   len(result) == limit,
	})
}

// GetProvider returns a specific provider by ID
func (s *Service) GetProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerID := vars["provider_id"]

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	provider, err := s.queryService.GetProvider(ctx, providerID)
	if err != nil {
		log.Printf("Hudi query error: %v", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Provider not found",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    provider,
	})
}

// GetItems returns items from Hudi with optional filters
func (s *Service) GetItems(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	providerID := r.URL.Query().Get("provider_id")
	categoryID := r.URL.Query().Get("category_id")
	city := r.URL.Query().Get("city")

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	items, err := s.queryService.GetItems(ctx, providerID, categoryID, city, limit)
	if err != nil {
		log.Printf("Hudi query error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"data":     items,
		"count":    len(items),
		"limit":    limit,
		"has_more": len(items) == limit,
	})
}

// GetProviderItems returns all items for a specific provider
func (s *Service) GetProviderItems(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerID := vars["provider_id"]

	limit := 1000
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 10000 {
			limit = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	items, err := s.queryService.GetItems(ctx, providerID, "", "", limit)
	if err != nil {
		log.Printf("Hudi query error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"data":        items,
		"count":       len(items),
		"provider_id": providerID,
	})
}

// GetStats returns statistics about Hudi data
func (s *Service) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	stats, err := s.queryService.GetStats(ctx)
	if err != nil {
		log.Printf("Hudi query error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}

