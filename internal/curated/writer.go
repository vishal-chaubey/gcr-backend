package curated

import (
	"context"
	"time"

	"gcr-backend/internal/model"
	"gcr-backend/internal/storage"
)

// WriteValidProviders writes curated provider rows to Hudi (stub) and returns
// CatalogAccepted events for each provider+category combination.
func WriteValidProviders(ctx context.Context, env *model.OnSearchEnvelope, providers []model.Provider) ([]model.CatalogAccepted, error) {
	events := []model.CatalogAccepted{}
	tC := time.Now().UTC().Format(time.RFC3339Nano)

	for _, provider := range providers {
		// Write to Hudi stub (JSONL)
		if err := storage.WriteProviderCatalog(ctx, env.Context, provider); err != nil {
			continue // skip on error, but continue with others
		}

		// Extract categories and emit one event per category
		for _, cat := range provider.Categories {
			events = append(events, model.CatalogAccepted{
				SellerID:   env.Context.BppID,
				City:       env.Context.City,
				Category:   cat.ID,
				Timestamp:  tC,
				ProviderID: provider.ID,
				Domain:     env.Context.Domain,
			})
		}
	}

	return events, nil
}

