package model

// OnSearchEnvelope models the top-level ONDC on_search payload.
type OnSearchEnvelope struct {
	Context OnSearchContext `json:"context" validate:"required"`
	Message OnSearchMessage `json:"message" validate:"required"`
}

type OnSearchContext struct {
	Domain        string `json:"domain" validate:"required"`
	Country       string `json:"country" validate:"required"`
	City          string `json:"city" validate:"required"`
	Action        string `json:"action" validate:"required,eq=on_search"`
	CoreVersion   string `json:"core_version" validate:"required"`
	BapID         string `json:"bap_id" validate:"required"`
	BapURI        string `json:"bap_uri" validate:"required"`
	BppID         string `json:"bpp_id" validate:"required"`
	BppURI        string `json:"bpp_uri" validate:"required"`
	TransactionID string `json:"transaction_id" validate:"required"`
	MessageID     string `json:"message_id" validate:"required"`
	Timestamp     string `json:"timestamp" validate:"required"`
}

type OnSearchMessage struct {
	Catalog Catalog `json:"catalog" validate:"required"`
}

type Catalog struct {
	BPPDescriptor   BPPDescriptor  `json:"bpp/descriptor" validate:"required"`
	BPPFulfillments []Fulfillment  `json:"bpp/fulfillments" validate:"dive"`
	BPPProviders    []Provider     `json:"bpp/providers" validate:"dive"`
}

type BPPDescriptor struct {
	Name      string   `json:"name" validate:"required"`
	Symbol    string   `json:"symbol" validate:"omitempty,url"`
	ShortDesc string   `json:"short_desc" validate:"required"`
	LongDesc  string   `json:"long_desc" validate:"required"`
	Images    []string `json:"images" validate:"dive,required"`
}

type Fulfillment struct {
	ID   string `json:"id" validate:"required"`
	Type string `json:"type" validate:"required"`
}

type Provider struct {
	ID         string            `json:"id" validate:"required"`
	Time       *ProviderTime     `json:"time,omitempty"`
	Descriptor ProviderDescriptor `json:"descriptor" validate:"required"`
	Categories []Category        `json:"categories" validate:"dive"`
	Items      []Item            `json:"items,omitempty" validate:"dive"`
}

type Item struct {
	ID            string         `json:"id" validate:"required"`
	Descriptor    ItemDescriptor `json:"descriptor" validate:"required"`
	Price         ItemPrice      `json:"price" validate:"required"`
	Quantity      *ItemQuantity  `json:"quantity,omitempty"`
	CategoryID    string         `json:"category_id" validate:"required"`
	CategoryIDs   []string       `json:"category_ids,omitempty"`
	FulfillmentID string         `json:"fulfillment_id,omitempty"`
	LocationID    string         `json:"location_id,omitempty"`
	Time          *ItemTime      `json:"time,omitempty"`
}

type ItemDescriptor struct {
	Name      string   `json:"name" validate:"required"`
	Symbol    string   `json:"symbol,omitempty"`
	ShortDesc string   `json:"short_desc,omitempty"`
	LongDesc  string   `json:"long_desc,omitempty"`
	Images    []string `json:"images,omitempty" validate:"dive"`
}

type ItemPrice struct {
	Currency     string `json:"currency" validate:"required"`
	Value        string `json:"value" validate:"required"`
	MaximumValue string `json:"maximum_value,omitempty"`
}

type ItemQuantity struct {
	Available *ItemQuantityAvailable `json:"available,omitempty"`
	Maximum   *ItemQuantityAvailable `json:"maximum,omitempty"`
	Unitized  *ItemQuantityUnitized  `json:"unitized,omitempty"`
}

type ItemQuantityAvailable struct {
	Count string `json:"count" validate:"required"`
	Unit  string `json:"unit,omitempty"`
}

type ItemQuantityUnitized struct {
	Measure ItemMeasure `json:"measure" validate:"required"`
}

type ItemMeasure struct {
	Value string `json:"value" validate:"required"`
	Unit  string `json:"unit" validate:"required"`
}

type ItemTime struct {
	Label     string `json:"label" validate:"required"`
	Timestamp string `json:"timestamp" validate:"required"`
}

type ProviderTime struct {
	Label     string `json:"label" validate:"required"`
	Timestamp string `json:"timestamp" validate:"required"`
}

type ProviderDescriptor struct {
	Name      string   `json:"name" validate:"required"`
	Symbol    string   `json:"symbol" validate:"omitempty,url"`
	ShortDesc string   `json:"short_desc" validate:"required"`
	LongDesc  string   `json:"long_desc" validate:"required"`
	Images    []string `json:"images" validate:"dive"`
}

type Category struct {
	ID               string            `json:"id" validate:"required"`
	ParentCategoryID string            `json:"parent_category_id"`
	Descriptor       CategoryDescriptor `json:"descriptor" validate:"required"`
}

type CategoryDescriptor struct {
	Name      string `json:"name" validate:"required"`
	ShortDesc string `json:"short_desc" validate:"required"`
	LongDesc  string `json:"long_desc" validate:"required"`
}


