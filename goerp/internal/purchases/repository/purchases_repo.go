package repository

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/goerp/goerp/internal/purchases/domain"
	"github.com/goerp/goerp/internal/shared/database"
)

type PurchasesRepository struct{ db *database.DB }

func NewPurchasesRepository(db *database.DB) *PurchasesRepository {
	return &PurchasesRepository{db: db}
}

// ---- helpers ----------------------------------------------------------------

func ptNullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func ptParseTime(s string) time.Time {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05+00:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ---- ListSuppliers ----------------------------------------------------------

func (r *PurchasesRepository) ListSuppliers(tenantID string, page int) ([]*domain.Supplier, int, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * 50

	rows, err := r.db.Query(`
		SELECT id, tenant_id, name, COALESCE(email,''), COALESCE(phone,''),
		       COALESCE(company_name,''), COALESCE(tax_id,''), COALESCE(address,''),
		       COALESCE(city,''), COALESCE(country,''), payment_terms, currency,
		       balance, rating, is_active, COALESCE(notes,''), created_at
		FROM suppliers
		WHERE tenant_id=?
		ORDER BY name
		LIMIT 50 OFFSET ?`, tenantID, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*domain.Supplier
	for rows.Next() {
		s := &domain.Supplier{}
		var isActive int
		var createdStr string
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.Email, &s.Phone,
			&s.CompanyName, &s.TaxID, &s.Address, &s.City, &s.Country,
			&s.PaymentTerms, &s.Currency, &s.Balance, &s.Rating,
			&isActive, &s.Notes, &createdStr,
		); err != nil {
			return nil, 0, err
		}
		s.IsActive = isActive == 1
		s.CreatedAt = ptParseTime(createdStr)
		list = append(list, s)
	}

	var total int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM suppliers WHERE tenant_id=?`, tenantID).Scan(&total)
	return list, total, nil
}

// ---- CreateSupplier ---------------------------------------------------------

func (r *PurchasesRepository) CreateSupplier(s *domain.Supplier) error {
	s.ID = uuid.New().String()
	s.CreatedAt = time.Now()
	if s.Currency == "" {
		s.Currency = "USD"
	}
	if s.PaymentTerms == 0 {
		s.PaymentTerms = 30
	}

	_, err := r.db.Exec(`
		INSERT INTO suppliers
		  (id, tenant_id, name, email, phone, company_name, tax_id, address,
		   city, country, payment_terms, currency, balance, rating, is_active, notes, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,0,0,1,?,?)`,
		s.ID, s.TenantID, s.Name, s.Email, s.Phone,
		s.CompanyName, s.TaxID, s.Address, s.City, s.Country,
		s.PaymentTerms, s.Currency, s.Notes,
		s.CreatedAt.Format(time.RFC3339),
	)
	return err
}

// ---- GetSupplier ------------------------------------------------------------

func (r *PurchasesRepository) GetSupplier(id, tenantID string) (*domain.Supplier, error) {
	s := &domain.Supplier{}
	var isActive int
	var createdStr string
	err := r.db.QueryRow(`
		SELECT id, tenant_id, name, COALESCE(email,''), COALESCE(phone,''),
		       COALESCE(company_name,''), COALESCE(tax_id,''), COALESCE(address,''),
		       COALESCE(city,''), COALESCE(country,''), payment_terms, currency,
		       balance, rating, is_active, COALESCE(notes,''), created_at
		FROM suppliers WHERE id=? AND tenant_id=?`, id, tenantID).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Email, &s.Phone,
		&s.CompanyName, &s.TaxID, &s.Address, &s.City, &s.Country,
		&s.PaymentTerms, &s.Currency, &s.Balance, &s.Rating,
		&isActive, &s.Notes, &createdStr,
	)
	if err != nil {
		return nil, err
	}
	s.IsActive = isActive == 1
	s.CreatedAt = ptParseTime(createdStr)
	return s, nil
}

// ---- ListPurchaseOrders -----------------------------------------------------

func (r *PurchasesRepository) ListPurchaseOrders(f domain.PurchaseFilter) ([]*domain.PurchaseOrder, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	offset := (f.Page - 1) * 50

	rows, err := r.db.Query(`
		SELECT po.id, po.tenant_id, po.po_number,
		       COALESCE(po.supplier_id,''), COALESCE(s.name,''),
		       po.state, po.order_date, COALESCE(po.expected_date,''),
		       po.currency, po.subtotal, po.tax_amount, po.total,
		       po.amount_paid, po.amount_due, COALESCE(po.notes,''), po.created_at
		FROM purchase_orders po
		LEFT JOIN suppliers s ON po.supplier_id = s.id
		WHERE po.tenant_id=?
		ORDER BY po.created_at DESC
		LIMIT 50 OFFSET ?`, f.TenantID, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*domain.PurchaseOrder
	for rows.Next() {
		o := &domain.PurchaseOrder{}
		var createdStr string
		if err := rows.Scan(
			&o.ID, &o.TenantID, &o.PONumber,
			&o.SupplierID, &o.SupplierName,
			&o.State, &o.OrderDate, &o.ExpectedDate,
			&o.Currency, &o.Subtotal, &o.TaxAmount, &o.Total,
			&o.AmountPaid, &o.AmountDue, &o.Notes, &createdStr,
		); err != nil {
			return nil, 0, err
		}
		o.CreatedAt = ptParseTime(createdStr)
		list = append(list, o)
	}

	var total int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM purchase_orders WHERE tenant_id=?`, f.TenantID).Scan(&total)
	return list, total, nil
}

// ---- CreatePurchaseOrder ----------------------------------------------------

func (r *PurchasesRepository) CreatePurchaseOrder(o *domain.PurchaseOrder) error {
	o.ID = uuid.New().String()
	o.CreatedAt = time.Now()
	if o.State == "" {
		o.State = "draft"
	}
	if o.Currency == "" {
		o.Currency = "USD"
	}

	var count int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM purchase_orders WHERE tenant_id=?`, o.TenantID).Scan(&count)
	o.PONumber = fmt.Sprintf("PO-%05d", count+1)

	today := time.Now().Format("2006-01-02")
	o.AmountDue = o.Total

	_, err := r.db.Exec(`
		INSERT INTO purchase_orders
		  (id, tenant_id, po_number, supplier_id, state, order_date,
		   currency, subtotal, tax_amount, total, amount_paid, amount_due, notes, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,0,?,?,?)`,
		o.ID, o.TenantID, o.PONumber,
		ptNullStr(o.SupplierID), o.State, today,
		o.Currency, o.Subtotal, o.TaxAmount, o.Total,
		o.AmountDue, o.Notes,
		o.CreatedAt.Format(time.RFC3339),
	)
	return err
}

// ---- GetPurchaseOrder -------------------------------------------------------

func (r *PurchasesRepository) GetPurchaseOrder(id, tenantID string) (*domain.PurchaseOrder, error) {
	o := &domain.PurchaseOrder{}
	var createdStr string
	err := r.db.QueryRow(`
		SELECT po.id, po.tenant_id, po.po_number,
		       COALESCE(po.supplier_id,''), COALESCE(s.name,''),
		       po.state, po.order_date, COALESCE(po.expected_date,''),
		       po.currency, po.subtotal, po.tax_amount, po.total,
		       po.amount_paid, po.amount_due, COALESCE(po.notes,''), po.created_at
		FROM purchase_orders po
		LEFT JOIN suppliers s ON po.supplier_id = s.id
		WHERE po.id=? AND po.tenant_id=?`, id, tenantID).Scan(
		&o.ID, &o.TenantID, &o.PONumber,
		&o.SupplierID, &o.SupplierName,
		&o.State, &o.OrderDate, &o.ExpectedDate,
		&o.Currency, &o.Subtotal, &o.TaxAmount, &o.Total,
		&o.AmountPaid, &o.AmountDue, &o.Notes, &createdStr,
	)
	if err != nil {
		return nil, err
	}
	o.CreatedAt = ptParseTime(createdStr)
	return o, nil
}

// ---- UpdateOrderState -------------------------------------------------------

func (r *PurchasesRepository) UpdateOrderState(id, tenantID, state string) error {
	_, err := r.db.Exec(`
		UPDATE purchase_orders SET state=? WHERE id=? AND tenant_id=?`,
		state, id, tenantID)
	return err
}
