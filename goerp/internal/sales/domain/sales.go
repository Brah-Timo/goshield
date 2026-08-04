package domain

import "time"

type Customer struct {
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
	CreditLimit  float64   `json:"credit_limit"`
	PaymentTerms int       `json:"payment_terms"`
	Currency     string    `json:"currency"`
	Balance      float64   `json:"balance"`
	IsActive     bool      `json:"is_active"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
}

type SalesOrder struct {
	ID             string           `json:"id"`
	TenantID       string           `json:"tenant_id"`
	OrderNumber    string           `json:"order_number"`
	CustomerID     string           `json:"customer_id"`
	CustomerName   string           `json:"customer_name,omitempty"`
	State          string           `json:"state"`
	OrderDate      string           `json:"order_date"`
	DeliveryDate   string           `json:"delivery_date"`
	Currency       string           `json:"currency"`
	Subtotal       float64          `json:"subtotal"`
	DiscountAmount float64          `json:"discount_amount"`
	TaxAmount      float64          `json:"tax_amount"`
	Total          float64          `json:"total"`
	AmountPaid     float64          `json:"amount_paid"`
	AmountDue      float64          `json:"amount_due"`
	Notes          string           `json:"notes"`
	Lines          []*SalesOrderLine `json:"lines,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
}

type SalesOrderLine struct {
	ID          string  `json:"id"`
	OrderID     string  `json:"order_id"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name,omitempty"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	DiscountPct float64 `json:"discount_pct"`
	TaxRate     float64 `json:"tax_rate"`
	Subtotal    float64 `json:"subtotal"`
	TaxAmount   float64 `json:"tax_amount"`
	Total       float64 `json:"total"`
}

type Invoice struct {
	ID            string         `json:"id"`
	TenantID      string         `json:"tenant_id"`
	InvoiceNumber string         `json:"invoice_number"`
	OrderID       string         `json:"order_id,omitempty"`
	CustomerID    string         `json:"customer_id"`
	CustomerName  string         `json:"customer_name,omitempty"`
	InvoiceType   string         `json:"invoice_type"`
	State         string         `json:"state"`
	InvoiceDate   string         `json:"invoice_date"`
	DueDate       string         `json:"due_date"`
	Currency      string         `json:"currency"`
	Subtotal      float64        `json:"subtotal"`
	TaxAmount     float64        `json:"tax_amount"`
	Total         float64        `json:"total"`
	AmountPaid    float64        `json:"amount_paid"`
	AmountDue     float64        `json:"amount_due"`
	Notes         string         `json:"notes"`
	Lines         []*InvoiceLine `json:"lines,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type InvoiceLine struct {
	ID          string  `json:"id"`
	InvoiceID   string  `json:"invoice_id"`
	ProductID   string  `json:"product_id"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TaxRate     float64 `json:"tax_rate"`
	Total       float64 `json:"total"`
}

type SalesFilter struct {
	TenantID   string
	CustomerID string
	State      string
	Search     string
	DateFrom   string
	DateTo     string
	Page       int
	PageSize   int
}
