package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/goerp/goerp/internal/sales/domain"
	"github.com/goerp/goerp/internal/shared/database"
)

type SalesRepository struct {
	db *database.DB
}

func NewSalesRepository(db *database.DB) *SalesRepository {
	return &SalesRepository{db: db}
}

// ─── Customers ────────────────────────────────────────────────────────────────

func (r *SalesRepository) ListCustomers(tenantID, search string, page int) ([]*domain.Customer, int, error) {
	if page < 1 {
		page = 1
	}
	limit := 20
	offset := (page - 1) * limit

	where := "tenant_id=?"
	args := []interface{}{tenantID}
	if search != "" {
		where += " AND (name LIKE ? OR email LIKE ? OR company_name LIKE ?)"
		s := "%" + search + "%"
		args = append(args, s, s, s)
	}
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)

	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT id, tenant_id, name, COALESCE(email,''), COALESCE(phone,''),
		       COALESCE(company_name,''), is_active, created_at
		FROM customers WHERE %s
		ORDER BY name ASC LIMIT ? OFFSET ?`, where),
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var customers []*domain.Customer
	for rows.Next() {
		c := &domain.Customer{}
		var isActive int
		var createdAt string
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Email, &c.Phone,
			&c.CompanyName, &isActive, &createdAt); err != nil {
			continue
		}
		c.IsActive = isActive == 1
		customers = append(customers, c)
	}
	var total int
	r.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM customers WHERE %s`, where), countArgs...).Scan(&total)
	if customers == nil {
		customers = []*domain.Customer{}
	}
	return customers, total, nil
}

func (r *SalesRepository) GetCustomer(id, tenantID string) (*domain.Customer, error) {
	c := &domain.Customer{}
	var isActive int
	var createdAt string
	err := r.db.QueryRow(`
		SELECT id, tenant_id, name, COALESCE(email,''), COALESCE(phone,''),
		       COALESCE(company_name,''), is_active, created_at
		FROM customers WHERE id=? AND tenant_id=?`, id, tenantID).
		Scan(&c.ID, &c.TenantID, &c.Name, &c.Email, &c.Phone,
			&c.CompanyName, &isActive, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("customer not found")
	}
	c.IsActive = isActive == 1
	return c, err
}

func (r *SalesRepository) CreateCustomer(c *domain.Customer) error {
	c.ID = uuid.New().String()
	c.CreatedAt = time.Now()
	_, err := r.db.Exec(`
		INSERT INTO customers (id, tenant_id, name, email, phone, company_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,1,?,?)`,
		c.ID, c.TenantID, c.Name, c.Email, c.Phone, c.CompanyName,
		c.CreatedAt.Format(time.RFC3339), c.CreatedAt.Format(time.RFC3339))
	return err
}

// ─── Sales Orders ─────────────────────────────────────────────────────────────

func (r *SalesRepository) ListSalesOrders(f domain.SalesFilter) ([]*domain.SalesOrder, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	limit := f.PageSize
	if limit < 1 {
		limit = 20
	}
	offset := (f.Page - 1) * limit

	where := "so.tenant_id=?"
	args := []interface{}{f.TenantID}
	if f.State != "" {
		where += " AND so.state=?"
		args = append(args, f.State)
	}
	if f.CustomerID != "" {
		where += " AND so.customer_id=?"
		args = append(args, f.CustomerID)
	}
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)

	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT so.id, so.tenant_id, so.order_number, COALESCE(so.customer_id,''),
		       COALESCE(c.name,''), so.state, COALESCE(so.order_date,''),
		       COALESCE(so.total,0), so.created_at
		FROM sales_orders so
		LEFT JOIN customers c ON c.id = so.customer_id
		WHERE %s ORDER BY so.created_at DESC LIMIT ? OFFSET ?`, where),
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*domain.SalesOrder
	for rows.Next() {
		o := &domain.SalesOrder{}
		var createdAt string
		if err := rows.Scan(&o.ID, &o.TenantID, &o.OrderNumber, &o.CustomerID,
			&o.CustomerName, &o.State, &o.OrderDate, &o.Total, &createdAt); err != nil {
			continue
		}
		orders = append(orders, o)
	}
	var total int
	r.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM sales_orders so WHERE %s`, where), countArgs...).Scan(&total)
	if orders == nil {
		orders = []*domain.SalesOrder{}
	}
	return orders, total, nil
}

func (r *SalesRepository) GetSalesOrder(id, tenantID string) (*domain.SalesOrder, error) {
	o := &domain.SalesOrder{}
	var createdAt string
	err := r.db.QueryRow(`
		SELECT so.id, so.tenant_id, so.order_number, COALESCE(so.customer_id,''),
		       COALESCE(c.name,''), so.state, COALESCE(so.order_date,''),
		       COALESCE(so.total,0), so.created_at
		FROM sales_orders so
		LEFT JOIN customers c ON c.id = so.customer_id
		WHERE so.id=? AND so.tenant_id=?`, id, tenantID).
		Scan(&o.ID, &o.TenantID, &o.OrderNumber, &o.CustomerID,
			&o.CustomerName, &o.State, &o.OrderDate, &o.Total, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found")
	}
	return o, err
}

func (r *SalesRepository) CreateSalesOrder(o *domain.SalesOrder) error {
	o.ID = uuid.New().String()
	now := time.Now()
	o.CreatedAt = now
	if o.OrderDate == "" {
		o.OrderDate = now.Format("2006-01-02")
	}
	if o.State == "" {
		o.State = "draft"
	}
	o.OrderNumber = fmt.Sprintf("SO-%s-%06d", now.Format("2006"), now.UnixNano()%1000000)

	_, err := r.db.Exec(`
		INSERT INTO sales_orders
		(id, tenant_id, order_number, customer_id, state, order_date,
		 subtotal, tax_amount, total, notes, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		o.ID, o.TenantID, o.OrderNumber, o.CustomerID,
		o.State, o.OrderDate,
		o.Subtotal, o.TaxAmount, o.Total, o.Notes,
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	return err
}

func (r *SalesRepository) UpdateOrderState(id, tenantID, state string) error {
	_, err := r.db.Exec(`
		UPDATE sales_orders SET state=?, updated_at=? WHERE id=? AND tenant_id=?`,
		state, time.Now().Format(time.RFC3339), id, tenantID)
	return err
}

// ─── Invoices ─────────────────────────────────────────────────────────────────

func (r *SalesRepository) ListInvoices(f domain.SalesFilter) ([]*domain.Invoice, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	limit := f.PageSize
	if limit < 1 {
		limit = 20
	}
	offset := (f.Page - 1) * limit

	where := "inv.tenant_id=?"
	args := []interface{}{f.TenantID}
	if f.State != "" {
		where += " AND inv.state=?"
		args = append(args, f.State)
	}
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)

	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT inv.id, inv.tenant_id, inv.invoice_number, COALESCE(inv.customer_id,''),
		       COALESCE(c.name,''), inv.state, COALESCE(inv.invoice_date,''),
		       COALESCE(inv.total,0), COALESCE(inv.amount_due,0), inv.created_at
		FROM invoices inv
		LEFT JOIN customers c ON c.id = inv.customer_id
		WHERE %s ORDER BY inv.created_at DESC LIMIT ? OFFSET ?`, where),
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var invs []*domain.Invoice
	for rows.Next() {
		inv := &domain.Invoice{}
		var createdAt string
		if err := rows.Scan(&inv.ID, &inv.TenantID, &inv.InvoiceNumber, &inv.CustomerID,
			&inv.CustomerName, &inv.State, &inv.InvoiceDate,
			&inv.Total, &inv.AmountDue, &createdAt); err != nil {
			continue
		}
		invs = append(invs, inv)
	}
	var total int
	r.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM invoices inv WHERE %s`, where), countArgs...).Scan(&total)
	if invs == nil {
		invs = []*domain.Invoice{}
	}
	return invs, total, nil
}

func (r *SalesRepository) GetInvoice(id, tenantID string) (*domain.Invoice, error) {
	inv := &domain.Invoice{}
	var createdAt string
	err := r.db.QueryRow(`
		SELECT inv.id, inv.tenant_id, inv.invoice_number, COALESCE(inv.customer_id,''),
		       COALESCE(c.name,''), inv.state, COALESCE(inv.invoice_date,''),
		       COALESCE(inv.total,0), COALESCE(inv.amount_due,0), inv.created_at
		FROM invoices inv
		LEFT JOIN customers c ON c.id = inv.customer_id
		WHERE inv.id=? AND inv.tenant_id=?`, id, tenantID).
		Scan(&inv.ID, &inv.TenantID, &inv.InvoiceNumber, &inv.CustomerID,
			&inv.CustomerName, &inv.State, &inv.InvoiceDate,
			&inv.Total, &inv.AmountDue, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invoice not found")
	}
	return inv, err
}

func (r *SalesRepository) CreateInvoice(inv *domain.Invoice) error {
	inv.ID = uuid.New().String()
	now := time.Now()
	inv.CreatedAt = now
	if inv.InvoiceDate == "" {
		inv.InvoiceDate = now.Format("2006-01-02")
	}
	if inv.State == "" {
		inv.State = "draft"
	}
	inv.InvoiceNumber = fmt.Sprintf("INV-%s-%06d", now.Format("2006"), now.UnixNano()%1000000)
	inv.AmountDue = inv.Total

	_, err := r.db.Exec(`
		INSERT INTO invoices
		(id, tenant_id, invoice_number, customer_id, state, invoice_date,
		 subtotal, tax_amount, total, amount_due, notes, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		inv.ID, inv.TenantID, inv.InvoiceNumber, inv.CustomerID,
		inv.State, inv.InvoiceDate,
		inv.Subtotal, inv.TaxAmount, inv.Total, inv.AmountDue, inv.Notes,
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	return err
}

func (r *SalesRepository) UpdateInvoiceState(id, tenantID, state string) error {
	_, err := r.db.Exec(`
		UPDATE invoices SET state=?, updated_at=? WHERE id=? AND tenant_id=?`,
		state, time.Now().Format(time.RFC3339), id, tenantID)
	return err
}
