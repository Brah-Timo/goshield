package domain

import "time"

// ─── Product ──────────────────────────────────────────────────────────────────

type Product struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	SKU            string            `json:"sku"`
	Name           map[string]string `json:"name"`
	NameEn         string            `json:"name_en,omitempty"`
	Description    string            `json:"description"`
	CategoryID     string            `json:"category_id"`
	CategoryName   string            `json:"category_name,omitempty"`
	UOM            string            `json:"uom"`
	BasePrice      float64           `json:"base_price"`
	CostPrice      float64           `json:"cost_price"`
	SalePrice      float64           `json:"sale_price"`
	TaxRate        float64           `json:"tax_rate"`
	Barcode        string            `json:"barcode"`
	QRCode         string            `json:"qr_code"`
	TrackInventory bool              `json:"track_inventory"`
	TrackBatch     bool              `json:"track_batch"`
	TrackSerial    bool              `json:"track_serial"`
	HasExpiry      bool              `json:"has_expiry"`
	MinStockLevel  float64           `json:"min_stock_level"`
	ReorderPoint   float64           `json:"reorder_point"`
	CurrentStock   float64           `json:"current_stock,omitempty"`
	ReservedQty    float64           `json:"reserved_qty,omitempty"`
	AvailableQty   float64           `json:"available_qty,omitempty"`
	StockValue     float64           `json:"stock_value,omitempty"`
	IsActive       bool              `json:"is_active"`
	ImageURL       string            `json:"image_url"`
	Tags           []string          `json:"tags,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// ─── Product Category ─────────────────────────────────────────────────────────

type ProductCategory struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	ParentID     string    `json:"parent_id"`
	ParentName   string    `json:"parent_name,omitempty"`
	Description  string    `json:"description"`
	ProductCount int       `json:"product_count,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ─── Product Variant ──────────────────────────────────────────────────────────

type ProductVariant struct {
	ID            string            `json:"id"`
	ProductID     string            `json:"product_id"`
	ProductName   string            `json:"product_name,omitempty"`
	SKU           string            `json:"sku"`
	Attributes    map[string]string `json:"attributes"`
	PriceModifier float64           `json:"price_modifier"`
	StockQty      float64           `json:"stock_qty"`
	Barcode       string            `json:"barcode"`
	IsActive      bool              `json:"is_active"`
	CreatedAt     time.Time         `json:"created_at"`
}

// ─── Warehouse ────────────────────────────────────────────────────────────────

type Warehouse struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Address     string    `json:"address"`
	City        string    `json:"city"`
	Country     string    `json:"country"`
	ManagerID   string    `json:"manager_id"`
	ManagerName string    `json:"manager_name,omitempty"`
	IsActive    bool      `json:"is_active"`
	LocationCount int     `json:"location_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ─── Stock Location ───────────────────────────────────────────────────────────

type StockLocation struct {
	ID           string    `json:"id"`
	WarehouseID  string    `json:"warehouse_id"`
	WarehouseName string   `json:"warehouse_name,omitempty"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	LocationType string    `json:"location_type"`
	ParentID     string    `json:"parent_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// ─── Stock Move ───────────────────────────────────────────────────────────────

type StockMove struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	ProductID        string     `json:"product_id"`
	ProductName      string     `json:"product_name,omitempty"`
	ProductSKU       string     `json:"product_sku,omitempty"`
	VariantID        string     `json:"variant_id"`
	FromLocationID   string     `json:"from_location_id"`
	FromLocationName string     `json:"from_location_name,omitempty"`
	ToLocationID     string     `json:"to_location_id"`
	ToLocationName   string     `json:"to_location_name,omitempty"`
	Quantity         float64    `json:"quantity"`
	UnitCost         float64    `json:"unit_cost"`
	TotalCost        float64    `json:"total_cost,omitempty"`
	BatchNumber      string     `json:"batch_number"`
	SerialNumber     string     `json:"serial_number"`
	ExpiryDate       *time.Time `json:"expiry_date,omitempty"`
	MoveType         string     `json:"move_type"`
	Reference        string     `json:"reference"`
	State            string     `json:"state"`
	Notes            string     `json:"notes"`
	CreatedBy        string     `json:"created_by"`
	CreatedAt        time.Time  `json:"created_at"`
	DoneAt           *time.Time `json:"done_at,omitempty"`
}

// ─── Batch ────────────────────────────────────────────────────────────────────

type Batch struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	ProductID       string     `json:"product_id"`
	ProductName     string     `json:"product_name,omitempty"`
	BatchNumber     string     `json:"batch_number"`
	ManufactureDate *time.Time `json:"manufacture_date,omitempty"`
	ExpiryDate      *time.Time `json:"expiry_date,omitempty"`
	QtyReceived     float64    `json:"qty_received"`
	QtyRemaining    float64    `json:"qty_remaining"`
	SupplierID      string     `json:"supplier_id"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ─── Stock Summary ────────────────────────────────────────────────────────────

type StockSummary struct {
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	SKU          string  `json:"sku"`
	CategoryName string  `json:"category_name"`
	UOM          string  `json:"uom"`
	TotalStock   float64 `json:"total_stock"`
	ReservedQty  float64 `json:"reserved_qty"`
	AvailableQty float64 `json:"available_qty"`
	UnitCost     float64 `json:"unit_cost"`
	StockValue   float64 `json:"stock_value"`
	MinLevel     float64 `json:"min_level"`
	ReorderPoint float64 `json:"reorder_point"`
	IsLow        bool    `json:"is_low"`
	NeedsReorder bool    `json:"needs_reorder"`
}

// ─── Inventory Stats ──────────────────────────────────────────────────────────

type InventoryStats struct {
	TotalProducts   int     `json:"total_products"`
	ActiveProducts  int     `json:"active_products"`
	TotalCategories int     `json:"total_categories"`
	TotalWarehouses int     `json:"total_warehouses"`
	LowStockCount   int     `json:"low_stock_count"`
	OutOfStock      int     `json:"out_of_stock"`
	TotalStockValue float64 `json:"total_stock_value"`
	MovesThisMonth  int     `json:"moves_this_month"`
	IncomingValue   float64 `json:"incoming_value"`
	OutgoingValue   float64 `json:"outgoing_value"`
}

// ─── Adjustment ───────────────────────────────────────────────────────────────

type StockAdjustment struct {
	ProductID  string  `json:"product_id"`
	LocationID string  `json:"location_id"`
	Quantity   float64 `json:"quantity"`
	Reason     string  `json:"reason"`
	Notes      string  `json:"notes"`
}

// ─── Filters ──────────────────────────────────────────────────────────────────

type ListProductsFilter struct {
	TenantID   string
	CategoryID string
	Search     string
	IsActive   *bool
	LowStock   bool
	Page       int
	PageSize   int
	SortBy     string
	SortDir    string
}

type ListStockMovesFilter struct {
	TenantID   string
	ProductID  string
	MoveType   string
	State      string
	DateFrom   string
	DateTo     string
	Page       int
	PageSize   int
}

// ─── Move type constants ──────────────────────────────────────────────────────

const (
	MoveTypeIn       = "in"
	MoveTypeOut      = "out"
	MoveTypeTransfer = "transfer"
	MoveTypeAdjust   = "adjust"
	MoveTypeReturn   = "return"
	MoveTypeScrap    = "scrap"

	MoveStateDraft    = "draft"
	MoveStateDone     = "done"
	MoveStateCancelled = "cancelled"

	LocationTypeInternal = "internal"
	LocationTypeSupplier = "supplier"
	LocationTypeCustomer = "customer"
	LocationTypeVirtual  = "virtual"
)
