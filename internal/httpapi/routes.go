package httpapi

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"

	"gcr-backend/internal/kstream"
	"gcr-backend/internal/model"
	"gcr-backend/internal/processing"
)

// go-playground/validator/v10: Struct validator for ONDC payload schema validation.
var validate = validator.New()

// RegisterRoutes wires HTTP routes (Edge/ingest side only).
// gorilla/mux: Router provides method-based routing and URL pattern matching.
func RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/health", healthHandler).Methods(http.MethodGet)
	r.HandleFunc("/ondc/on_search", onSearchHandler).Methods(http.MethodPost) // Seller ingest
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// onSearchHandler accepts large ONDC on_search payloads from sellers (Edge/ingest side).
// It validates, then fans out work to a parallel processing pipeline so that
// even large catalogs complete in a few seconds.
func onSearchHandler(w http.ResponseWriter, r *http.Request) {
	var payload model.OnSearchEnvelope
	reader := io.Reader(r.Body)
	if enc := r.Header.Get("Content-Encoding"); strings.EqualFold(enc, "gzip") {
		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "failed to decompress gzip body", http.StatusBadRequest)
			return
		}
		defer gr.Close()
		reader = gr
	}

	if err := json.NewDecoder(reader).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// go-playground/validator/v10: Struct validates ONDC envelope against struct tags.
	// Checks required fields, format (URLs, enums), and nested validation rules.
	if err := validate.Struct(payload); err != nil {
		http.Error(w, "schema validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Edge: Publish to Kafka catalog.ingest (as per sequence diagram)
	if err := kstream.PublishOnSearchIngest(ctx, &payload); err != nil {
		log.Printf("Edge: failed to publish to Kafka: %v", err)
		// Continue anyway (fire-and-forget per architecture)
	}

	// Also process inline for immediate response (stats)
	stats, err := processing.ProcessOnSearch(ctx, &payload)
	if err != nil {
		log.Printf("on_search processing error: %v", err)
		http.Error(w, "processing failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Always respond gzip-compressed per SNP requirement.
	w.Header().Set("Content-Encoding", "gzip")

	gw := gzip.NewWriter(w)
	defer gw.Close()
	_ = json.NewEncoder(gw).Encode(stats)
}


