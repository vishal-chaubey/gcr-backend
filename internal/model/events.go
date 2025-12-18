package model

// CatalogAccepted is the event emitted after Curated Writer commits to Hudi.
// It is published to Kafka topic.catalog.accepted and consumed by Projectors.
type CatalogAccepted struct {
	SellerID   string `json:"seller_id"`   // bpp_id
	City       string `json:"city"`
	Category   string `json:"category"`    // extracted from provider categories
	Timestamp  string `json:"timestamp"`   // commit timestamp (tC)
	ProviderID string `json:"provider_id"`
	Domain     string `json:"domain"`
}

// SearchRequest models a buyer /search call.
type SearchRequest struct {
	Context SearchContext `json:"context" validate:"required"`
	Message SearchMessage `json:"message" validate:"required"`
}

type SearchContext struct {
	Domain  string `json:"domain" validate:"required"`
	City    string `json:"city" validate:"required"`
	Action  string `json:"action" validate:"required,eq=search"`
	BapID   string `json:"bap_id" validate:"required"`
	BapURI  string `json:"bap_uri" validate:"required"`
}

type SearchMessage struct {
	Intent Intent `json:"intent" validate:"required"`
}

type Intent struct {
	Item ItemIntent `json:"item" validate:"required"`
}

type ItemIntent struct {
	Category CategoryRef `json:"category" validate:"required"`
}

type CategoryRef struct {
	ID string `json:"id" validate:"required"`
}

