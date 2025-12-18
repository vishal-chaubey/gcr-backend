package trino

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Client handles Trino queries
type Client struct {
	baseURL    string
	httpClient *http.Client
	user       string
}

// QueryResult represents a Trino query result
type QueryResult struct {
	Columns []Column `json:"columns"`
	Data    []Row    `json:"data"`
	Stats   Stats    `json:"stats"`
}

// Column represents a column in the result
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Row represents a row of data
type Row []interface{}

// Stats represents query statistics
type Stats struct {
	State           string        `json:"state"`
	ElapsedTime     time.Duration `json:"elapsedTime"`
	QueuedTime      time.Duration `json:"queuedTime"`
	ExecutionTime   time.Duration `json:"executionTime"`
	TotalRows       int64         `json:"totalRows"`
	TotalBytes      int64         `json:"totalBytes"`
	CompletedSplits int64         `json:"completedSplits"`
}

// NewClient creates a new Trino client
func NewClient() *Client {
	trinoURL := getEnv("TRINO_URL", "http://trino:8080")
	user := getEnv("TRINO_USER", "admin")

	return &Client{
		baseURL: trinoURL,
		user:    user,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteQuery executes a SQL query and returns results
func (c *Client) ExecuteQuery(ctx context.Context, sql string) (*QueryResult, error) {
	// Create query request
	queryURL := fmt.Sprintf("%s/v1/statement", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "POST", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Trino-User", c.user)
	req.Header.Set("X-Trino-Catalog", "hudi")
	req.Header.Set("X-Trino-Schema", "default")
	req.URL.RawQuery = url.Values{"query": {sql}}.Encode()

	// Execute query
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("trino error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Query executes a SQL query and returns JSON results
func (c *Client) Query(ctx context.Context, sql string) ([]map[string]interface{}, error) {
	result, err := c.ExecuteQuery(ctx, sql)
	if err != nil {
		return nil, err
	}

	// Convert rows to maps
	rows := make([]map[string]interface{}, 0, len(result.Data))
	for _, row := range result.Data {
		rowMap := make(map[string]interface{})
		for i, col := range result.Columns {
			if i < len(row) {
				rowMap[col.Name] = row[i]
			}
		}
		rows = append(rows, rowMap)
	}

	return rows, nil
}

// HealthCheck checks if Trino is available
func (c *Client) HealthCheck(ctx context.Context) error {
	healthURL := fmt.Sprintf("%s/v1/info", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("trino health check failed: status %d", resp.StatusCode)
	}

	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

