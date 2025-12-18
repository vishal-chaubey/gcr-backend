package schemagate

import (
	"context"
	"log"
	"sync"

	"gcr-backend/internal/bloom"
	"gcr-backend/internal/model"
)

// ValidateProvider performs provider-level validation.
// If provider is invalid, entire provider is discarded.
func ValidateProvider(ctx context.Context, provider model.Provider, ctxMeta model.OnSearchContext) (valid bool, rejectReason string) {
	// Basic validation: provider must have ID and descriptor
	if provider.ID == "" {
		return false, "provider.id missing"
	}
	if provider.Descriptor.Name == "" {
		return false, "provider.descriptor.name missing"
	}
	if len(provider.Categories) == 0 {
		return false, "provider.categories empty"
	}
	return true, ""
}

// ValidateItem performs item-level validation.
// If item is invalid, only that item is discarded (provider continues).
func ValidateItem(ctx context.Context, item model.Item) (valid bool, rejectReason string) {
	if item.ID == "" {
		return false, "item.id missing"
	}
	if item.Descriptor.Name == "" {
		return false, "item.descriptor.name missing"
	}
	if item.CategoryID == "" {
		return false, "item.category_id missing"
	}
	if item.Price.Currency == "" {
		return false, "item.price.currency missing"
	}
	if item.Price.Value == "" {
		return false, "item.price.value missing"
	}
	// Validate quantity if present
	if item.Quantity != nil && item.Quantity.Available != nil {
		if item.Quantity.Available.Count == "" {
			return false, "item.quantity.available.count missing"
		}
	}
	return true, ""
}

// ProcessCatalog validates all providers and items with parallel processing.
// - Provider-level: if invalid, discard entire provider
// - Item-level: if invalid, discard only that item
// - Item deduplication: use Bloom filter to skip duplicate items
func ProcessCatalog(ctx context.Context, env *model.OnSearchEnvelope) (validProviders []model.Provider, rejections []Rejection) {
	validProviders = []model.Provider{}
	rejections = []Rejection{}

	providers := env.Message.Catalog.BPPProviders
	if len(providers) == 0 {
		return validProviders, rejections
	}

	// Parallel processing configuration
	maxWorkers := 16 // Optimal for most systems
	if len(providers) < maxWorkers {
		maxWorkers = len(providers)
	}

	// Channel for provider processing jobs
	providerJobs := make(chan model.Provider, len(providers))
	results := make(chan providerResult, len(providers))

	// Start worker goroutines for provider processing
	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for provider := range providerJobs {
				result := processProvider(ctx, provider, env.Context)
				results <- result
			}
		}()
	}

	// Send all providers to workers
	go func() {
		for _, provider := range providers {
			providerJobs <- provider
		}
		close(providerJobs)
	}()

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		if result.valid {
			validProviders = append(validProviders, result.provider)
		} else {
			rejections = append(rejections, result.rejections...)
		}
	}

	return validProviders, rejections
}

type providerResult struct {
	provider  model.Provider
	valid     bool
	rejections []Rejection
}

// processProvider validates a single provider and its items with parallel processing
func processProvider(ctx context.Context, provider model.Provider, ctxMeta model.OnSearchContext) providerResult {
	// Step 1: Validate provider-level schema
	valid, reason := ValidateProvider(ctx, provider, ctxMeta)
	if !valid {
		return providerResult{
			provider:  provider,
			valid:     false,
			rejections: []Rejection{
				{
					Scope:  "provider:" + provider.ID,
					Reason: reason,
				},
			},
		}
	}

	// Step 2: Process items in parallel (if provider has items)
	if len(provider.Items) == 0 {
		// Provider is valid but has no items - still accept it
		return providerResult{
			provider:  provider,
			valid:     true,
			rejections: []Rejection{},
		}
	}

	// Process items in parallel batches
	validItems := processItemsParallel(ctx, provider.Items, ctxMeta, provider.ID)
	
	// Create new provider with only valid items
	provider.Items = validItems.items
	
	// Combine rejections
	rejections := validItems.rejections

	return providerResult{
		provider:  provider,
		valid:     true,
		rejections: rejections,
	}
}

type itemsResult struct {
	items     []model.Item
	rejections []Rejection
}

// processItemsParallel processes items in parallel batches for optimal performance
func processItemsParallel(ctx context.Context, items []model.Item, ctxMeta model.OnSearchContext, providerID string) itemsResult {
	if len(items) == 0 {
		return itemsResult{
			items:     []model.Item{},
			rejections: []Rejection{},
		}
	}

	// Optimal batch size: process 100 items per worker for good balance
	batchSize := 100
	maxWorkers := 32 // Higher for items since they're lighter
	if len(items) < maxWorkers*batchSize {
		maxWorkers = (len(items) + batchSize - 1) / batchSize
	}

	// Channel for item batches
	itemBatches := make(chan []model.Item, maxWorkers)
	results := make(chan itemsResult, maxWorkers)

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range itemBatches {
				result := processItemBatch(ctx, batch, ctxMeta, providerID)
				results <- result
			}
		}()
	}

	// Send item batches to workers
	go func() {
		for i := 0; i < len(items); i += batchSize {
			end := i + batchSize
			if end > len(items) {
				end = len(items)
			}
			itemBatches <- items[i:end]
		}
		close(itemBatches)
	}()

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	allValidItems := []model.Item{}
	allRejections := []Rejection{}
	for result := range results {
		allValidItems = append(allValidItems, result.items...)
		allRejections = append(allRejections, result.rejections...)
	}

	return itemsResult{
		items:     allValidItems,
		rejections: allRejections,
	}
}

// processItemBatch processes a batch of items with validation and deduplication
func processItemBatch(ctx context.Context, items []model.Item, ctxMeta model.OnSearchContext, providerID string) itemsResult {
	validItems := []model.Item{}
	rejections := []Rejection{}

	for _, item := range items {
		// Step 1: Validate item schema
		valid, reason := ValidateItem(ctx, item)
		if !valid {
			rejections = append(rejections, Rejection{
				Scope:  "item:" + providerID + ":" + item.ID,
				Reason: reason,
			})
			log.Printf("SchemaGate: rejected item %s in provider %s: %s", item.ID, providerID, reason)
			continue
		}

		// Step 2: Check for duplicates using Bloom filter
		// Item key format: "domain:city:provider_id:item_id"
		itemKey := ctxMeta.Domain + ":" + ctxMeta.City + ":" + providerID + ":" + item.ID
		if bloom.SeenItem(ctx, itemKey) {
			// Item is a duplicate, skip it but don't reject (it's already in DB)
			log.Printf("SchemaGate: duplicate item %s in provider %s, skipping", item.ID, providerID)
			continue
		}

		// Item is valid and not a duplicate
		validItems = append(validItems, item)
	}

	return itemsResult{
		items:     validItems,
		rejections: rejections,
	}
}

// Rejection records a rejected scope (provider/item) with reason.
type Rejection struct {
	Scope  string `json:"scope"`  // e.g., "provider:10020084" or "item:12345"
	Reason string `json:"reason"` // e.g., "provider.descriptor.name missing"
}
