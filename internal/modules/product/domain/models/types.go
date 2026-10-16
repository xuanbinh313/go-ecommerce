package models

import (
	"encoding/json"
	"time"
)

type Product struct {
	ID            string           `json:"id"`
	Title         string           `json:"title"`
	Handle        string           `json:"handle"`
	Subtitle      *string          `json:"subtitle"`
	Description   *string          `json:"description"`
	IsGiftcard    bool             `json:"is_giftcard"`
	Status        string           `json:"status"`
	Thumbnail     *string          `json:"thumbnail"`
	Weight        *float64         `json:"weight"`
	Length        *float64         `json:"length"`
	Height        *float64         `json:"height"`
	Width         *float64         `json:"width"`
	OriginCountry *string          `json:"origin_country"`
	HSCode        *string          `json:"hs_code"`
	MIDCode       *string          `json:"mid_code"`
	Material      *string          `json:"material"`
	TypeID        *string          `json:"type_id"`
	CollectionID  *string          `json:"collection_id"`
	Discountable  bool             `json:"discountable"`
	ExternalID    *string          `json:"external_id"`
	Metadata      json.RawMessage  `json:"metadata"`
	Options       []ProductOption  `json:"options,omitempty"`
	Variants      []ProductVariant `json:"variants,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	DeletedAt     *time.Time       `json:"deleted_at,omitempty"`
}

type ProductType struct {
	ID         string          `json:"id"`
	Value      string          `json:"value"`
	ExternalID *string         `json:"external_id"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	DeletedAt  *time.Time      `json:"deleted_at,omitempty"`
}

type ProductTypeInput struct {
	Value      string          `json:"value"`
	ExternalID *string         `json:"external_id"`
	Metadata   json.RawMessage `json:"metadata"`
}

type ProductCatalogFilter struct {
	IDs, Values, ExternalIDs        []string
	Query                           string
	CreatedAt, UpdatedAt, DeletedAt map[string]string
}

type ProductTypeProductReference struct {
	ProductID string
	TypeID    string
}

type ProductTag struct {
	ID         string          `json:"id"`
	Value      string          `json:"value"`
	ExternalID *string         `json:"external_id"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	DeletedAt  *time.Time      `json:"deleted_at,omitempty"`
}

type ProductTagInput struct {
	Value      string          `json:"value"`
	ExternalID *string         `json:"external_id"`
	Metadata   json.RawMessage `json:"metadata"`
}

type ProductCategory struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	Handle           string          `json:"handle"`
	MPath            string          `json:"mpath"`
	IsActive         bool            `json:"is_active"`
	IsInternal       bool            `json:"is_internal"`
	Rank             float64         `json:"rank"`
	ExternalID       *string         `json:"external_id"`
	Metadata         json.RawMessage `json:"metadata"`
	ParentCategoryID *string         `json:"parent_category_id"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        *time.Time      `json:"deleted_at,omitempty"`
}

type StoreProductCategoryFilter struct {
	IDs, Names, Descriptions, Handles, ParentCategoryIDs, ExternalIDs []string
	Query                                                             string
	CreatedAt, UpdatedAt                                              map[string]string
}

type ProductCategoryInput struct {
	Name             string          `json:"name"`
	Description      *string         `json:"description,omitempty"`
	Handle           *string         `json:"handle,omitempty"`
	IsActive         *bool           `json:"is_active,omitempty"`
	IsInternal       *bool           `json:"is_internal,omitempty"`
	Rank             *float64        `json:"rank,omitempty"`
	ExternalID       *string         `json:"external_id,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	ParentCategoryID *string         `json:"parent_category_id,omitempty"`
}

type ProductCollection struct {
	ID         string          `json:"id"`
	Title      string          `json:"title"`
	Handle     string          `json:"handle"`
	ExternalID *string         `json:"external_id"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	DeletedAt  *time.Time      `json:"deleted_at,omitempty"`
}
type ProductCollectionInput struct {
	Title      string          `json:"title"`
	Handle     *string         `json:"handle,omitempty"`
	ExternalID *string         `json:"external_id"`
	Metadata   json.RawMessage `json:"metadata"`
}
type ProductCollectionFilter struct {
	IDs, Titles, Handles, ExternalIDs []string
	Query                             string
	CreatedAt, UpdatedAt              map[string]string
	Order                             string
}
type ProductOption struct {
	ID          string               `json:"id"`
	ProductID   string               `json:"product_id,omitempty"`
	Title       string               `json:"title"`
	IsExclusive bool                 `json:"is_exclusive"`
	Metadata    json.RawMessage      `json:"metadata"`
	Values      []ProductOptionValue `json:"values,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	DeletedAt   *time.Time           `json:"deleted_at,omitempty"`
}

type ProductOptionFilter struct {
	IDs                             []string
	Titles                          []string
	Query                           string
	IsExclusive                     *bool
	CreatedAt, UpdatedAt, DeletedAt map[string]string
}

type CreateProductOptionInput struct {
	Title       string          `json:"title"`
	Values      []string        `json:"values"`
	Ranks       map[string]int  `json:"ranks,omitempty"`
	IsExclusive *bool           `json:"is_exclusive,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}
type ProductOptionValue struct {
	ID       string          `json:"id"`
	Value    string          `json:"value"`
	OptionID *string         `json:"option_id"`
	Rank     *int            `json:"rank"`
	Metadata json.RawMessage `json:"metadata"`
}
type ProductVariant struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	SKU             *string                `json:"sku"`
	Barcode         *string                `json:"barcode"`
	EAN             *string                `json:"ean"`
	UPC             *string                `json:"upc"`
	AllowBackorder  bool                   `json:"allow_backorder"`
	ManageInventory bool                   `json:"manage_inventory"`
	HSCode          *string                `json:"hs_code"`
	OriginCountry   *string                `json:"origin_country"`
	MIDCode         *string                `json:"mid_code"`
	Material        *string                `json:"material"`
	Weight          *float64               `json:"weight"`
	Length          *float64               `json:"length"`
	Height          *float64               `json:"height"`
	Width           *float64               `json:"width"`
	Metadata        json.RawMessage        `json:"metadata"`
	VariantRank     *int                   `json:"variant_rank"`
	Thumbnail       *string                `json:"thumbnail"`
	ProductID       *string                `json:"product_id"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	DeletedAt       *time.Time             `json:"deleted_at,omitempty"`
	Options         []ProductOptionValue   `json:"options,omitempty"`
	Product         *ProductVariantProduct `json:"product,omitempty"`
}

type ProductVariantProduct struct {
	ID     string  `json:"id"`
	TypeID *string `json:"type_id"`
}

type ProductInput struct {
	ID                *string                 `json:"id,omitempty"`
	Title             string                  `json:"title"`
	Handle            *string                 `json:"handle,omitempty"`
	Subtitle          *string                 `json:"subtitle,omitempty"`
	Description       *string                 `json:"description,omitempty"`
	IsGiftcard        *bool                   `json:"is_giftcard,omitempty"`
	Status            *string                 `json:"status,omitempty"`
	Thumbnail         *string                 `json:"thumbnail,omitempty"`
	Weight            *float64                `json:"weight,omitempty"`
	Length            *float64                `json:"length,omitempty"`
	Height            *float64                `json:"height,omitempty"`
	Width             *float64                `json:"width,omitempty"`
	OriginCountry     *string                 `json:"origin_country,omitempty"`
	HSCode            *string                 `json:"hs_code,omitempty"`
	MIDCode           *string                 `json:"mid_code,omitempty"`
	Material          *string                 `json:"material,omitempty"`
	TypeID            *string                 `json:"type_id,omitempty"`
	CollectionID      *string                 `json:"collection_id,omitempty"`
	Discountable      *bool                   `json:"discountable,omitempty"`
	ExternalID        *string                 `json:"external_id,omitempty"`
	Metadata          json.RawMessage         `json:"metadata,omitempty"`
	Options           []ProductOptionInput    `json:"options"`
	Variants          []ProductVariantInput   `json:"variants,omitempty"`
	SalesChannels     []SalesChannelReference `json:"sales_channels,omitempty"`
	ShippingProfileID *string                 `json:"shipping_profile_id,omitempty"`
}
type SalesChannelReference struct {
	ID string `json:"id"`
}
type ProductOptionInput struct {
	ID          *string         `json:"id,omitempty"`
	Title       string          `json:"title,omitempty"`
	IsExclusive *bool           `json:"is_exclusive,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	Values      []string        `json:"values,omitempty"`
	ValueIDs    []string        `json:"value_ids,omitempty"`
}
type ProductVariantInput struct {
	ID              *string           `json:"id,omitempty"`
	ProductID       string            `json:"product_id"`
	Title           string            `json:"title"`
	SKU             *string           `json:"sku,omitempty"`
	Barcode         *string           `json:"barcode,omitempty"`
	EAN             *string           `json:"ean,omitempty"`
	UPC             *string           `json:"upc,omitempty"`
	AllowBackorder  *bool             `json:"allow_backorder,omitempty"`
	ManageInventory *bool             `json:"manage_inventory,omitempty"`
	HSCode          *string           `json:"hs_code,omitempty"`
	OriginCountry   *string           `json:"origin_country,omitempty"`
	MIDCode         *string           `json:"mid_code,omitempty"`
	Material        *string           `json:"material,omitempty"`
	Weight          *float64          `json:"weight,omitempty"`
	Length          *float64          `json:"length,omitempty"`
	Height          *float64          `json:"height,omitempty"`
	Width           *float64          `json:"width,omitempty"`
	Metadata        json.RawMessage   `json:"metadata,omitempty"`
	VariantRank     *int              `json:"variant_rank,omitempty"`
	Thumbnail       *string           `json:"thumbnail,omitempty"`
	Options         map[string]string `json:"options,omitempty"`
}

type ProductVariantFilter struct {
	IDs                             []string
	ProductIDs                      []string
	SKUs                            []string
	EANs                            []string
	UPCs                            []string
	Barcodes                        []string
	Query                           string
	ManageInventory                 *bool
	AllowBackorder                  *bool
	OptionValue                     *string
	OptionID                        *string
	SalesChannelIDs                 []string
	PublishedOnly                   bool
	CreatedAt, UpdatedAt, DeletedAt map[string]string
}
type CreateProductsInput struct {
	Products       []ProductInput `json:"products"`
	AdditionalData map[string]any `json:"additional_data,omitempty"`
}
