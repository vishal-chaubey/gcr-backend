package jsonl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// QueryService provides querying capabilities for JSONL files
type QueryService struct {
	dataDir string
}

// NewQueryService creates a new JSONL query service
func NewQueryService() *QueryService {
	// Use /app/data in Docker container, ./data for local development
	dataDir := getEnv("DATA_DIR", "/app/data/hudi/providers")
	// Fallback to ./data if /app/data doesn't exist (local dev)
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		dataDir = "./data/hudi/providers"
	}
	return &QueryService{
		dataDir: dataDir,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Provider represents a provider from JSONL
type Provider struct {
	ProviderID string                 `json:"provider_id"`
	Domain     string                 `json:"domain"`
	City       string                 `json:"city"`
	BapID      string                 `json:"bap_id"`
	BppID      string                 `json:"bpp_id"`
	Timestamp  string                 `json:"timestamp"`
	Descriptor map[string]interface{} `json:"descriptor"`
	Categories []interface{}          `json:"categories"`
	Items      []interface{}          `json:"items"`
}

// GetAllProviders returns all providers from JSONL files
func (s *QueryService) GetAllProviders(ctx context.Context, limit, offset int) ([]Provider, error) {
	// Try multiple possible paths
	possibleDirs := []string{
		s.dataDir,
		"/app/data/hudi/providers",
		"./data/hudi/providers",
		"data/hudi/providers",
	}

	var files []string
	var err error
	
	for _, dir := range possibleDirs {
		pattern := filepath.Join(dir, "*.jsonl")
		files, err = filepath.Glob(pattern)
		if err == nil && len(files) > 0 {
			s.dataDir = dir
			break
		}
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	
	if len(files) == 0 {
		return []Provider{}, nil // Return empty, not error
	}

	providers := []Provider{}
	count := 0
	skipped := 0

	for _, file := range files {
		if count >= limit+offset {
			break
		}

		// Read file line by line (JSONL format - one JSON object per line)
		fileData, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Split by newlines and process each line
		lines := strings.Split(string(fileData), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if count >= limit+offset {
				break
			}

			var provider Provider
			if err := json.Unmarshal([]byte(line), &provider); err != nil {
				continue
			}

			if skipped < offset {
				skipped++
				continue
			}

			if count < limit {
				providers = append(providers, provider)
				count++
			}
		}
	}

	return providers, nil
}

// GetProvider returns a specific provider by ID
func (s *QueryService) GetProvider(ctx context.Context, providerID string) (*Provider, error) {
	file := filepath.Join(s.dataDir, fmt.Sprintf("%s.jsonl", providerID))
	
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// JSONL format - get the last line (most recent record)
	lines := strings.Split(string(data), "\n")
	var lastLine string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			lastLine = line
			break
		}
	}

	if lastLine == "" {
		return nil, fmt.Errorf("provider file is empty")
	}

	var provider Provider
	if err := json.Unmarshal([]byte(lastLine), &provider); err != nil {
		return nil, fmt.Errorf("failed to parse provider: %w", err)
	}

	return &provider, nil
}

// GetItems returns items with optional filters
func (s *QueryService) GetItems(ctx context.Context, providerID, categoryID, city string, limit int) ([]map[string]interface{}, error) {
	files, err := filepath.Glob(filepath.Join(s.dataDir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	items := []map[string]interface{}{}
	count := 0

	for _, file := range files {
		if count >= limit {
			break
		}

		fileData, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Process each line in JSONL file
		lines := strings.Split(string(fileData), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if count >= limit {
				break
			}

			var provider Provider
			if err := json.Unmarshal([]byte(line), &provider); err != nil {
				continue
			}

			// Apply filters
			if providerID != "" && provider.ProviderID != providerID {
				continue
			}
			if city != "" && provider.City != city {
				continue
			}

			// Extract items
			for _, itemRaw := range provider.Items {
				item, ok := itemRaw.(map[string]interface{})
				if !ok {
					continue
				}

				// Filter by category if specified
				if categoryID != "" {
					itemCatID, _ := item["category_id"].(string)
					if itemCatID != categoryID {
						continue
					}
				}

				itemMap := map[string]interface{}{
					"provider_id":    provider.ProviderID,
					"city":           provider.City,
					"domain":         provider.Domain,
					"item_id":        item["id"],
					"item_name":      getNestedString(item, "descriptor", "name"),
					"category_id":    item["category_id"],
					"price_value":    getNestedString(item, "price", "value"),
					"price_currency": getNestedString(item, "price", "currency"),
				}

				items = append(items, itemMap)
				count++

				if count >= limit {
					break
				}
			}
		}
	}

	return items, nil
}

// GetStats returns statistics about the data
func (s *QueryService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	files, err := filepath.Glob(filepath.Join(s.dataDir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	stats := map[string]interface{}{
		"total_providers": int(0),
		"total_records":   int(0),
		"total_items":     int(0),
		"total_domains":   make(map[string]bool),
		"total_cities":    make(map[string]bool),
		"latest_update":   "",
	}

	for _, file := range files {
		fileData, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Process each line (JSONL format)
		lines := strings.Split(string(fileData), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			var provider Provider
			if err := json.Unmarshal([]byte(line), &provider); err != nil {
				continue
			}

			stats["total_providers"] = stats["total_providers"].(int) + 1
			stats["total_records"] = stats["total_records"].(int) + 1
			stats["total_items"] = stats["total_items"].(int) + len(provider.Items)

			stats["total_domains"].(map[string]bool)[provider.Domain] = true
			stats["total_cities"].(map[string]bool)[provider.City] = true

			if provider.Timestamp > stats["latest_update"].(string) {
				stats["latest_update"] = provider.Timestamp
			}
		}
	}

	// Convert sets to counts
	stats["total_domains"] = len(stats["total_domains"].(map[string]bool))
	stats["total_cities"] = len(stats["total_cities"].(map[string]bool))

	return stats, nil
}

// Helper function to get nested string value
func getNestedString(m map[string]interface{}, keys ...string) string {
	current := interface{}(m)
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return ""
		}
	}
	if str, ok := current.(string); ok {
		return str
	}
	return ""
}

