package domain

import "time"

type Supplier struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	CompanyName  string    `json:"company_name"`
	TaxID        string    `json:"tax_id"`
	Address      string    `json:"address"`
	City         string    `json:"city"`
	Country      string    `json:"country"`
	PaymentTerms int       `json:"payment_terms"`
	Currency     string    `json:"currency"`
	Balance      float64   `json:"balance"`
	Rating       int       `json:"rating"`
	IsActive     bool      `json:"is_active"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
}

type PurchaseOrder struct {
	ID           string               `json:"id"`
	TenantID     string               `json:"tenant_id"`
	PONumber     string               `json:"po_number"`
	SupplierID   string               `json:"supplier_id"`
	SupplierName string               `json:"supplier_name,omitempty"`
	State        string               `json:"state"`
	OrderDate    string               `json:"order_date"`
	ExpectedDate string               `json:"expected_date"`
	Currency     string               `json:"currency"`
	Subtotal     float64              `json:"subtotal"`
	TaxAmount    float64              `json:"tax_amount"`
	Total        float64              `json:"total"`
	AmountPaid   float64              `json:"amount_paid"`
	AmountDue    float64              `json:"amount_due"`
	Notes        string               `json:"notes"`
	Lines        []*PurchaseOrderLine `json:"lines,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
}

type PurchaseOrderLine struct {
	ID          string  `json:"id"`
	POID        string  `json:"po_id"`
	ProductID   string  `json:"product_id"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitCost    float64 `json:"unit_cost"`
	TaxRate     float64 `json:"tax_rate"`
	Total       float64 `json:"total"`
	QtyReceived float64 `json:"qty_received"`
}

type PurchaseFilter struct {
	TenantID   string
	SupplierID string
	State      string
	Page       int
}
