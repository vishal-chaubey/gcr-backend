package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"gcr-backend/internal/bloom"
	"gcr-backend/internal/discovery"
	"gcr-backend/internal/httpapi"
	"gcr-backend/internal/kstream"
	"gcr-backend/internal/projections"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialise Redis Bloom filter (idempotent).
	bloom.Init()

	// Start Kafka consumers in background goroutines
	go func() {
		log.Println("Starting SchemaGate consumer...")
		if err := kstream.ConsumeIngestTopic(ctx); err != nil {
			log.Printf("SchemaGate consumer error: %v", err)
		}
	}()

	go func() {
		log.Println("Starting Projectors consumer...")
		if err := projections.ConsumeAcceptedTopic(ctx); err != nil {
			log.Printf("Projectors consumer error: %v", err)
		}
	}()

	// Give consumers time to connect
	time.Sleep(2 * time.Second)

	// Setup HTTP routes
	r := mux.NewRouter()
	httpapi.RegisterRoutes(r) // Edge + ingest side

	// Discovery API (read side)
	disc := discovery.NewService()
	disc.RegisterRoutes(r)

	addr := getEnv("GCR_HTTP_ADDR", ":8080")
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("Shutting down...")
		cancel()
		_ = server.Shutdown(context.Background())
	}()

	log.Printf("GCR API listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}


