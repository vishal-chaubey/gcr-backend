package jsonl

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// RegisterRoutes registers JSONL query API routes
func RegisterRoutes(r *mux.Router) {
	service := NewQueryService()
	api := r.PathPrefix("/api/data").Subrouter()

	api.HandleFunc("/providers", service.GetProvidersHandler).Methods("GET")
	api.HandleFunc("/providers/{provider_id}", service.GetProviderHandler).Methods("GET")
	api.HandleFunc("/items", service.GetItemsHandler).Methods("GET")
	api.HandleFunc("/stats", service.GetStatsHandler).Methods("GET")
}

// GetProvidersHandler handles GET /api/data/providers
func (s *QueryService) GetProvidersHandler(w http.ResponseWriter, r *http.Request) {
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

	providers, err := s.GetAllProviders(r.Context(), limit, offset)
	if err != nil {
		log.Printf("Error getting providers: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Convert to response format
	result := []map[string]interface{}{}
	for _, p := range providers {
		itemCount := len(p.Items)

		providerName := ""
		if desc, ok := p.Descriptor["name"].(string); ok {
			providerName = desc
		}

		result = append(result, map[string]interface{}{
			"provider_id":  p.ProviderID,
			"domain":       p.Domain,
			"city":         p.City,
			"bpp_id":       p.BppID,
			"timestamp":    p.Timestamp,
			"provider_name": providerName,
			"items_count":  itemCount,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

// GetProviderHandler handles GET /api/data/providers/{provider_id}
func (s *QueryService) GetProviderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerID := vars["provider_id"]

	provider, err := s.GetProvider(r.Context(), providerID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Provider not found",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    []Provider{*provider},
	})
}

// GetItemsHandler handles GET /api/data/items
func (s *QueryService) GetItemsHandler(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	providerID := r.URL.Query().Get("provider_id")
	categoryID := r.URL.Query().Get("category_id")
	city := r.URL.Query().Get("city")

	items, err := s.GetItems(r.Context(), providerID, categoryID, city, limit)
	if err != nil {
		log.Printf("Error getting items: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    items,
	})
}

// GetStatsHandler handles GET /api/data/stats
func (s *QueryService) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := s.GetStats(r.Context())
	if err != nil {
		log.Printf("Error getting stats: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    []map[string]interface{}{stats},
	})
}

