package trino

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Service provides Trino query API
type Service struct {
	client *Client
}

// NewService creates a new Trino service
func NewService() *Service {
	return &Service{
		client: NewClient(),
	}
}

// RegisterRoutes registers Trino API routes
func (s *Service) RegisterRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/trino").Subrouter()
	api.HandleFunc("/health", s.HealthCheck).Methods("GET")
	api.HandleFunc("/query", s.ExecuteQuery).Methods("POST")
	api.HandleFunc("/providers", s.GetProviders).Methods("GET")
	api.HandleFunc("/providers/{provider_id}", s.GetProvider).Methods("GET")
	api.HandleFunc("/items", s.GetItems).Methods("GET")
	api.HandleFunc("/stats", s.GetStats).Methods("GET")
}

// HealthCheckResponse represents health check response
type HealthCheckResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// HealthCheck checks Trino connectivity
func (s *Service) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.client.HealthCheck(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HealthCheckResponse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(HealthCheckResponse{
		Status:  "ok",
		Message: "Trino is available",
	})
}

// QueryRequest represents a SQL query request
type QueryRequest struct {
	SQL string `json:"sql"`
}

// QueryResponse represents a query response
type QueryResponse struct {
	Success bool                   `json:"success"`
	Data    []map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Stats   *Stats                 `json:"stats,omitempty"`
}

// ExecuteQuery executes a custom SQL query
func (s *Service) ExecuteQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(QueryResponse{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	if req.SQL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(QueryResponse{
			Success: false,
			Error:   "SQL query is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := s.client.ExecuteQuery(ctx, req.SQL)
	if err != nil {
		log.Printf("Trino query error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Convert to map format
	data := make([]map[string]interface{}, 0, len(result.Data))
	for _, row := range result.Data {
		rowMap := make(map[string]interface{})
		for i, col := range result.Columns {
			if i < len(row) {
				rowMap[col.Name] = row[i]
			}
		}
		data = append(data, rowMap)
	}

	json.NewEncoder(w).Encode(QueryResponse{
		Success: true,
		Data:    data,
		Stats:   &result.Stats,
	})
}

// GetProviders returns all providers
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

	sql := fmt.Sprintf(`
		SELECT 
			provider_id,
			domain,
			city,
			bpp_id,
			timestamp,
			descriptor->>'name' as provider_name,
			json_array_length(items) as items_count
		FROM hudi.default.providers
		ORDER BY timestamp DESC
		LIMIT %d OFFSET %d
	`, limit, offset)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	data, err := s.client.Query(ctx, sql)
	if err != nil {
		log.Printf("Trino query error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(QueryResponse{
		Success: true,
		Data:    data,
	})
}

// GetProvider returns a specific provider by ID
func (s *Service) GetProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerID := vars["provider_id"]

	sql := fmt.Sprintf(`
		SELECT *
		FROM hudi.default.providers
		WHERE provider_id = '%s'
		ORDER BY timestamp DESC
		LIMIT 1
	`, providerID)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	data, err := s.client.Query(ctx, sql)
	if err != nil {
		log.Printf("Trino query error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if len(data) == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(QueryResponse{
			Success: false,
			Error:   "Provider not found",
		})
		return
	}

	json.NewEncoder(w).Encode(QueryResponse{
		Success: true,
		Data:    data,
	})
}

// GetItems returns items with optional filters
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

	// Build WHERE clause
	where := []string{}
	if providerID != "" {
		where = append(where, fmt.Sprintf("provider_id = '%s'", providerID))
	}
	if categoryID != "" {
		where = append(where, fmt.Sprintf("json_extract_scalar(item, '$.category_id') = '%s'", categoryID))
	}
	if city != "" {
		where = append(where, fmt.Sprintf("city = '%s'", city))
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	sql := fmt.Sprintf(`
		SELECT 
			provider_id,
			city,
			domain,
			json_extract_scalar(item, '$.id') as item_id,
			json_extract_scalar(item, '$.descriptor.name') as item_name,
			json_extract_scalar(item, '$.category_id') as category_id,
			json_extract_scalar(item, '$.price.value') as price_value,
			json_extract_scalar(item, '$.price.currency') as price_currency
		FROM hudi.default.providers
		CROSS JOIN UNNEST(json_extract(items, '$')) AS t(item)
		%s
		LIMIT %d
	`, whereClause, limit)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	data, err := s.client.Query(ctx, sql)
	if err != nil {
		log.Printf("Trino query error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(QueryResponse{
		Success: true,
		Data:    data,
	})
}

// GetStats returns statistics about the data
func (s *Service) GetStats(w http.ResponseWriter, r *http.Request) {
	sql := `
		SELECT 
			COUNT(DISTINCT provider_id) as total_providers,
			COUNT(*) as total_records,
			SUM(json_array_length(items)) as total_items,
			COUNT(DISTINCT domain) as total_domains,
			COUNT(DISTINCT city) as total_cities,
			MAX(timestamp) as latest_update
		FROM hudi.default.providers
	`

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	data, err := s.client.Query(ctx, sql)
	if err != nil {
		log.Printf("Trino query error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(QueryResponse{
		Success: true,
		Data:    data,
	})
}

