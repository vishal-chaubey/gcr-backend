package processing

import (
	"context"
	"sync"
	"time"

	"gcr-backend/internal/bloom"
	"gcr-backend/internal/kstream"
	"gcr-backend/internal/model"
	"gcr-backend/internal/storage"
)

// OnSearchStats is returned to the client so we can see throughput and latency.
type OnSearchStats struct {
	Providers int64 `json:"providers"`
	// DurationMillis is the end-to-end processing time for this on_search call.
	DurationMillis int64 `json:"duration_ms"`
}

// ProcessOnSearch validates business rules (delegated to storage) and fans
// out provider-level work in parallel so large catalogs complete quickly.
func ProcessOnSearch(ctx context.Context, env *model.OnSearchEnvelope) (*OnSearchStats, error) {
	start := time.Now()

	// Fire-and-forget publish to Kafka so multiple on_search calls are durably
	// captured on topic.catalog.on_search.ingest.
	_ = kstream.PublishOnSearchIngest(ctx, env)

	providers := env.Message.Catalog.BPPProviders
	if len(providers) == 0 {
		return &OnSearchStats{Providers: 0, DurationMillis: 0}, nil
	}

	workerCount := calcWorkerCount(len(providers))
	jobs := make(chan model.Provider)
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				_ = bloom.SeenProvider(ctx, env.Context.Domain+":"+env.Context.City+":"+p.ID)
				_ = storage.WriteProviderCatalog(ctx, env.Context, p)
			}
		}()
	}

	for _, p := range providers {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil, ctx.Err()
		case jobs <- p:
		}
	}

	close(jobs)
	wg.Wait()

	return &OnSearchStats{
		Providers:      int64(len(providers)),
		DurationMillis: time.Since(start).Milliseconds(),
	}, nil
}

func calcWorkerCount(n int) int {
	if n <= 0 {
		return 1
	}
	if n < 4 {
		return n
	}
	if n > 16 {
		return 16
	}
	return n
}


