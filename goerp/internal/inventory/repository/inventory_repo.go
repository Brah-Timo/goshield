package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/goerp/goerp/internal/inventory/domain"
	"github.com/goerp/goerp/internal/shared/database"
)

type InventoryRepository struct {
	db *database.DB
}

func NewInventoryRepository(db *database.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

// ─── Products ─────────────────────────────────────────────────────────────────

func (r *InventoryRepository) ListProducts(f domain.ListProductsFilter) ([]*domain.Product, int, error) {
	where := []string{"p.tenant_id = ?"}
	args := []interface{}{f.TenantID}

	if f.Search != "" {
		s := "%" + f.Search + "%"
		where = append(where, "(p.sku LIKE ? OR json_extract(p.name,'$.en') LIKE ? OR p.barcode LIKE ?)")
		args = append(args, s, s, s)
	}
	if f.CategoryID != "" {
		where = append(where, "p.category_id = ?")
		args = append(args, f.CategoryID)
	}
	if f.IsActive != nil {
		where = append(where, "p.is_active = ?")
		args = append(args, boolInt(*f.IsActive))
	}

	pg := f.Page
	if pg < 1 {
		pg = 1
	}
	ps := f.PageSize
	if ps < 1 {
		ps = 50
	}
	offset := (pg - 1) * ps

	sortCol := "p.created_at"
	switch f.SortBy {
	case "name":
		sortCol = "json_extract(p.name,'$.en')"
	case "sku":
		sortCol = "p.sku"
	case "price":
		sortCol = "p.sale_price"
	case "stock":
		sortCol = "current_stock"
	}
	sortDir := "DESC"
	if strings.ToUpper(f.SortDir) == "ASC" {
		sortDir = "ASC"
	}

	havingClause := ""
	if f.LowStock {
		havingClause = `HAVING current_stock <= p.min_stock_level AND p.min_stock_level > 0 AND p.track_inventory = 1`
	}

	wc := strings.Join(where, " AND ")
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)

	query := fmt.Sprintf(`
		SELECT
			p.id, p.tenant_id, p.sku,
			COALESCE(json_extract(p.name,'$.en'), p.sku) AS name_en,
			p.name,
			COALESCE(p.description,''),
			COALESCE(p.category_id,''),
			COALESCE(pc.name,''),
			COALESCE(p.unit_of_measure,'unit'),
			p.base_price, p.cost_price, p.sale_price, p.tax_rate,
			COALESCE(p.barcode,''), COALESCE(p.qr_code,''),
			p.track_inventory, p.track_batch, p.track_serial, p.has_expiry,
			p.min_stock_level, p.reorder_point,
			p.is_active, COALESCE(p.image_url,''),
			COALESCE(p.tags,'[]'),
			p.created_at, p.updated_at,
			COALESCE(
				SUM(CASE WHEN sm.to_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END) -
				SUM(CASE WHEN sm.from_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END),
				0
			) AS current_stock,
			p.cost_price * COALESCE(
				SUM(CASE WHEN sm.to_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END) -
				SUM(CASE WHEN sm.from_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END),
				0
			) AS stock_value
		FROM products p
		LEFT JOIN product_categories pc ON pc.id = p.category_id
		LEFT JOIN stock_moves sm ON sm.product_id = p.id
		WHERE %s
		GROUP BY p.id, p.tenant_id, p.sku, p.name, p.description, p.category_id,
		         pc.name, p.unit_of_measure, p.base_price, p.cost_price, p.sale_price,
		         p.tax_rate, p.barcode, p.qr_code, p.track_inventory, p.track_batch,
		         p.track_serial, p.has_expiry, p.min_stock_level, p.reorder_point,
		         p.is_active, p.image_url, p.tags, p.created_at, p.updated_at
		%s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, wc, havingClause, sortCol, sortDir)

	args = append(args, ps, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			continue
		}
		products = append(products, p)
	}

	var total int
	r.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM products p WHERE %s`, wc), countArgs...).Scan(&total)

	if products == nil {
		products = []*domain.Product{}
	}
	return products, total, nil
}

func (r *InventoryRepository) GetProduct(id, tenantID string) (*domain.Product, error) {
	query := `
		SELECT
			p.id, p.tenant_id, p.sku,
			COALESCE(json_extract(p.name,'$.en'), p.sku),
			p.name,
			COALESCE(p.description,''),
			COALESCE(p.category_id,''),
			COALESCE(pc.name,''),
			COALESCE(p.unit_of_measure,'unit'),
			p.base_price, p.cost_price, p.sale_price, p.tax_rate,
			COALESCE(p.barcode,''), COALESCE(p.qr_code,''),
			p.track_inventory, p.track_batch, p.track_serial, p.has_expiry,
			p.min_stock_level, p.reorder_point,
			p.is_active, COALESCE(p.image_url,''),
			COALESCE(p.tags,'[]'),
			p.created_at, p.updated_at,
			COALESCE(
				SUM(CASE WHEN sm.to_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END) -
				SUM(CASE WHEN sm.from_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END),
				0
			) AS current_stock,
			p.cost_price * COALESCE(
				SUM(CASE WHEN sm.to_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END) -
				SUM(CASE WHEN sm.from_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END),
				0
			) AS stock_value
		FROM products p
		LEFT JOIN product_categories pc ON pc.id = p.category_id
		LEFT JOIN stock_moves sm ON sm.product_id = p.id
		WHERE p.id = ? AND p.tenant_id = ?
		GROUP BY p.id`
	row := r.db.QueryRow(query, id, tenantID)
	return scanProduct(row)
}

func (r *InventoryRepository) CreateProduct(p *domain.Product) error {
	p.ID = uuid.New().String()
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	nameJSON, _ := json.Marshal(p.Name)
	tagsJSON, _ := json.Marshal(p.Tags)

	_, err := r.db.Exec(`
		INSERT INTO products
		(id, tenant_id, sku, name, description, category_id, unit_of_measure,
		 base_price, cost_price, sale_price, tax_rate, barcode, qr_code,
		 track_inventory, track_batch, track_serial, has_expiry,
		 min_stock_level, reorder_point, is_active, image_url, tags,
		 created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.TenantID, p.SKU,
		string(nameJSON), p.Description,
		nullStr(p.CategoryID), p.UOM,
		p.BasePrice, p.CostPrice, p.SalePrice, p.TaxRate,
		p.Barcode, p.QRCode,
		boolInt(p.TrackInventory), boolInt(p.TrackBatch),
		boolInt(p.TrackSerial), boolInt(p.HasExpiry),
		p.MinStockLevel, p.ReorderPoint,
		boolInt(p.IsActive), p.ImageURL, string(tagsJSON),
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	return err
}

func (r *InventoryRepository) UpdateProduct(p *domain.Product) error {
	now := time.Now()
	p.UpdatedAt = now
	nameJSON, _ := json.Marshal(p.Name)
	tagsJSON, _ := json.Marshal(p.Tags)

	_, err := r.db.Exec(`
		UPDATE products SET
			sku=?, name=?, description=?, category_id=?, unit_of_measure=?,
			base_price=?, cost_price=?, sale_price=?, tax_rate=?,
			barcode=?, qr_code=?,
			track_inventory=?, track_batch=?, track_serial=?, has_expiry=?,
			min_stock_level=?, reorder_point=?,
			is_active=?, image_url=?, tags=?, updated_at=?
		WHERE id=? AND tenant_id=?`,
		p.SKU, string(nameJSON), p.Description,
		nullStr(p.CategoryID), p.UOM,
		p.BasePrice, p.CostPrice, p.SalePrice, p.TaxRate,
		p.Barcode, p.QRCode,
		boolInt(p.TrackInventory), boolInt(p.TrackBatch),
		boolInt(p.TrackSerial), boolInt(p.HasExpiry),
		p.MinStockLevel, p.ReorderPoint,
		boolInt(p.IsActive), p.ImageURL, string(tagsJSON),
		now.Format(time.RFC3339),
		p.ID, p.TenantID,
	)
	return err
}

func (r *InventoryRepository) DeleteProduct(id, tenantID string) error {
	_, err := r.db.Exec(`UPDATE products SET is_active=0 WHERE id=? AND tenant_id=?`, id, tenantID)
	return err
}

// ─── Categories ───────────────────────────────────────────────────────────────

func (r *InventoryRepository) ListCategories(tenantID string) ([]*domain.ProductCategory, error) {
	rows, err := r.db.Query(`
		SELECT c.id, c.tenant_id, c.name,
		       COALESCE(c.parent_id,''), COALESCE(p.name,''),
		       COALESCE(c.description,''),
		       COUNT(DISTINCT pr.id) AS product_count,
		       c.created_at
		FROM product_categories c
		LEFT JOIN product_categories p ON p.id = c.parent_id
		LEFT JOIN products pr ON pr.category_id = c.id AND pr.is_active = 1
		WHERE c.tenant_id = ?
		GROUP BY c.id, c.tenant_id, c.name, c.parent_id, p.name, c.description, c.created_at
		ORDER BY c.name ASC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []*domain.ProductCategory
	for rows.Next() {
		c := &domain.ProductCategory{}
		var createdAtStr string
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.ParentID, &c.ParentName,
			&c.Description, &c.ProductCount, &createdAtStr); err != nil {
			continue
		}
		if t, err := parseTime(createdAtStr); err == nil {
			c.CreatedAt = t
		}
		cats = append(cats, c)
	}
	if cats == nil {
		cats = []*domain.ProductCategory{}
	}
	return cats, nil
}

func (r *InventoryRepository) CreateCategory(c *domain.ProductCategory) error {
	c.ID = uuid.New().String()
	c.CreatedAt = time.Now()
	_, err := r.db.Exec(`
		INSERT INTO product_categories (id, tenant_id, name, parent_id, description, created_at)
		VALUES (?,?,?,?,?,?)`,
		c.ID, c.TenantID, c.Name, nullStr(c.ParentID), c.Description,
		c.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *InventoryRepository) UpdateCategory(c *domain.ProductCategory) error {
	_, err := r.db.Exec(`
		UPDATE product_categories SET name=?, description=?, parent_id=?
		WHERE id=? AND tenant_id=?`,
		c.Name, c.Description, nullStr(c.ParentID), c.ID, c.TenantID)
	return err
}

func (r *InventoryRepository) DeleteCategory(id, tenantID string) error {
	_, err := r.db.Exec(`DELETE FROM product_categories WHERE id=? AND tenant_id=?`, id, tenantID)
	return err
}

// ─── Warehouses ───────────────────────────────────────────────────────────────

func (r *InventoryRepository) ListWarehouses(tenantID string) ([]*domain.Warehouse, error) {
	rows, err := r.db.Query(`
		SELECT w.id, w.tenant_id, w.name, w.code,
		       COALESCE(w.address,''), COALESCE(w.city,''), COALESCE(w.country,''),
		       COALESCE(w.manager_id,''), w.is_active,
		       COUNT(DISTINCT sl.id) AS location_count,
		       w.created_at
		FROM warehouses w
		LEFT JOIN stock_locations sl ON sl.warehouse_id = w.id
		WHERE w.tenant_id = ?
		GROUP BY w.id
		ORDER BY w.name ASC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var whs []*domain.Warehouse
	for rows.Next() {
		w := &domain.Warehouse{}
		var isActive int
		var createdAtStr string
		if err := rows.Scan(&w.ID, &w.TenantID, &w.Name, &w.Code,
			&w.Address, &w.City, &w.Country, &w.ManagerID, &isActive,
			&w.LocationCount, &createdAtStr); err != nil {
			continue
		}
		w.IsActive = isActive == 1
		if t, err := parseTime(createdAtStr); err == nil {
			w.CreatedAt = t
		}
		whs = append(whs, w)
	}
	if whs == nil {
		whs = []*domain.Warehouse{}
	}
	return whs, nil
}

func (r *InventoryRepository) CreateWarehouse(w *domain.Warehouse) error {
	w.ID = uuid.New().String()
	w.CreatedAt = time.Now()
	_, err := r.db.Exec(`
		INSERT INTO warehouses (id, tenant_id, name, code, address, city, country, is_active, created_at)
		VALUES (?,?,?,?,?,?,?,1,?)`,
		w.ID, w.TenantID, w.Name, w.Code, w.Address, w.City, w.Country,
		w.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *InventoryRepository) UpdateWarehouse(w *domain.Warehouse) error {
	_, err := r.db.Exec(`
		UPDATE warehouses SET name=?, code=?, address=?, city=?, country=?, is_active=?
		WHERE id=? AND tenant_id=?`,
		w.Name, w.Code, w.Address, w.City, w.Country, boolInt(w.IsActive),
		w.ID, w.TenantID)
	return err
}

func (r *InventoryRepository) DeleteWarehouse(id, tenantID string) error {
	_, err := r.db.Exec(`UPDATE warehouses SET is_active=0 WHERE id=? AND tenant_id=?`, id, tenantID)
	return err
}

// ─── Locations ────────────────────────────────────────────────────────────────

func (r *InventoryRepository) ListLocations(tenantID string) ([]*domain.StockLocation, error) {
	rows, err := r.db.Query(`
		SELECT sl.id, sl.warehouse_id, COALESCE(w.name,''),
		       sl.name, COALESCE(sl.code,''),
		       COALESCE(sl.location_type,'internal'),
		       COALESCE(sl.parent_id,''), sl.created_at
		FROM stock_locations sl
		LEFT JOIN warehouses w ON w.id = sl.warehouse_id
		WHERE w.tenant_id = ?
		ORDER BY w.name ASC, sl.name ASC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locs []*domain.StockLocation
	for rows.Next() {
		l := &domain.StockLocation{}
		var createdAtStr string
		if err := rows.Scan(&l.ID, &l.WarehouseID, &l.WarehouseName,
			&l.Name, &l.Code, &l.LocationType, &l.ParentID, &createdAtStr); err != nil {
			continue
		}
		if t, err := parseTime(createdAtStr); err == nil {
			l.CreatedAt = t
		}
		locs = append(locs, l)
	}
	if locs == nil {
		locs = []*domain.StockLocation{}
	}
	return locs, nil
}

func (r *InventoryRepository) CreateLocation(l *domain.StockLocation) error {
	l.ID = uuid.New().String()
	l.CreatedAt = time.Now()
	_, err := r.db.Exec(`
		INSERT INTO stock_locations (id, warehouse_id, name, code, location_type, parent_id, created_at)
		VALUES (?,?,?,?,?,?,?)`,
		l.ID, l.WarehouseID, l.Name, l.Code, l.LocationType,
		nullStr(l.ParentID), l.CreatedAt.Format(time.RFC3339))
	return err
}

// ─── Stock Moves ──────────────────────────────────────────────────────────────

func (r *InventoryRepository) ListStockMoves(f domain.ListStockMovesFilter) ([]*domain.StockMove, int, error) {
	where := []string{"sm.tenant_id = ?"}
	args := []interface{}{f.TenantID}

	if f.ProductID != "" {
		where = append(where, "sm.product_id = ?")
		args = append(args, f.ProductID)
	}
	if f.MoveType != "" {
		where = append(where, "sm.move_type = ?")
		args = append(args, f.MoveType)
	}
	if f.State != "" {
		where = append(where, "sm.state = ?")
		args = append(args, f.State)
	}
	if f.DateFrom != "" {
		where = append(where, "sm.created_at >= ?")
		args = append(args, f.DateFrom)
	}
	if f.DateTo != "" {
		where = append(where, "sm.created_at <= ?")
		args = append(args, f.DateTo)
	}

	pg := f.Page
	if pg < 1 {
		pg = 1
	}
	ps := f.PageSize
	if ps < 1 {
		ps = 50
	}
	offset := (pg - 1) * ps

	wc := strings.Join(where, " AND ")
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)

	query := fmt.Sprintf(`
		SELECT sm.id, sm.tenant_id, sm.product_id,
		       COALESCE(json_extract(p.name,'$.en'), p.sku, ''),
		       COALESCE(p.sku,''),
		       COALESCE(sm.variant_id,''),
		       COALESCE(sm.from_location_id,''),
		       COALESCE(fl.name,''),
		       COALESCE(sm.to_location_id,''),
		       COALESCE(tl.name,''),
		       sm.quantity, sm.unit_cost,
		       sm.quantity * sm.unit_cost,
		       COALESCE(sm.batch_number,''), COALESCE(sm.serial_number,''),
		       COALESCE(sm.expiry_date,''),
		       sm.move_type, COALESCE(sm.reference,''),
		       sm.state, COALESCE(sm.notes,''),
		       COALESCE(sm.created_by,''),
		       sm.created_at, COALESCE(sm.done_at,'')
		FROM stock_moves sm
		LEFT JOIN products p ON p.id = sm.product_id
		LEFT JOIN stock_locations fl ON fl.id = sm.from_location_id
		LEFT JOIN stock_locations tl ON tl.id = sm.to_location_id
		WHERE %s
		ORDER BY sm.created_at DESC
		LIMIT ? OFFSET ?`, wc)

	args = append(args, ps, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stock moves: %w", err)
	}
	defer rows.Close()

	var moves []*domain.StockMove
	for rows.Next() {
		m := &domain.StockMove{}
		var createdAtStr, doneAtStr, expiryStr string
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.ProductID,
			&m.ProductName, &m.ProductSKU, &m.VariantID,
			&m.FromLocationID, &m.FromLocationName,
			&m.ToLocationID, &m.ToLocationName,
			&m.Quantity, &m.UnitCost, &m.TotalCost,
			&m.BatchNumber, &m.SerialNumber, &expiryStr,
			&m.MoveType, &m.Reference, &m.State, &m.Notes,
			&m.CreatedBy, &createdAtStr, &doneAtStr,
		); err != nil {
			continue
		}
		if t, err := parseTime(createdAtStr); err == nil {
			m.CreatedAt = t
		}
		if doneAtStr != "" {
			if t, err := parseTime(doneAtStr); err == nil {
				m.DoneAt = &t
			}
		}
		if expiryStr != "" {
			if t, err := parseTime(expiryStr); err == nil {
				m.ExpiryDate = &t
			}
		}
		moves = append(moves, m)
	}

	var total int
	r.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM stock_moves sm WHERE %s`, wc), countArgs...).Scan(&total)

	if moves == nil {
		moves = []*domain.StockMove{}
	}
	return moves, total, nil
}

func (r *InventoryRepository) CreateStockMove(m *domain.StockMove) error {
	m.ID = uuid.New().String()
	m.CreatedAt = time.Now()
	if m.State == "" {
		m.State = domain.MoveStateDone
	}
	doneAt := ""
	if m.State == domain.MoveStateDone {
		doneAt = m.CreatedAt.Format(time.RFC3339)
	}
	expiryDate := ""
	if m.ExpiryDate != nil {
		expiryDate = m.ExpiryDate.Format("2006-01-02")
	}

	_, err := r.db.Exec(`
		INSERT INTO stock_moves
		(id, tenant_id, product_id, variant_id,
		 from_location_id, to_location_id,
		 quantity, unit_cost, batch_number, serial_number, expiry_date,
		 move_type, reference, state, notes, created_by, created_at, done_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		m.ID, m.TenantID, m.ProductID, nullStr(m.VariantID),
		nullStr(m.FromLocationID), nullStr(m.ToLocationID),
		m.Quantity, m.UnitCost, m.BatchNumber, m.SerialNumber, expiryDate,
		m.MoveType, m.Reference, m.State, m.Notes, m.CreatedBy,
		m.CreatedAt.Format(time.RFC3339), doneAt,
	)
	return err
}

// ─── Stock Summary ────────────────────────────────────────────────────────────

func (r *InventoryRepository) GetStockSummary(tenantID string) ([]*domain.StockSummary, error) {
	rows, err := r.db.Query(`
		SELECT
			p.id, COALESCE(json_extract(p.name,'$.en'), p.sku),
			p.sku, COALESCE(pc.name,''),
			COALESCE(p.unit_of_measure,'unit'),
			COALESCE(
				SUM(CASE WHEN sm.to_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END) -
				SUM(CASE WHEN sm.from_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END),
				0
			) AS total_stock,
			COALESCE(p.cost_price,0),
			p.min_stock_level, p.reorder_point
		FROM products p
		LEFT JOIN product_categories pc ON pc.id = p.category_id
		LEFT JOIN stock_moves sm ON sm.product_id = p.id
		WHERE p.tenant_id = ? AND p.is_active = 1 AND p.track_inventory = 1
		GROUP BY p.id, p.sku, p.name, pc.name, p.unit_of_measure,
		         p.cost_price, p.min_stock_level, p.reorder_point
		ORDER BY total_stock ASC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []*domain.StockSummary
	for rows.Next() {
		s := &domain.StockSummary{}
		if err := rows.Scan(&s.ProductID, &s.ProductName, &s.SKU, &s.CategoryName,
			&s.UOM, &s.TotalStock, &s.UnitCost, &s.MinLevel, &s.ReorderPoint); err != nil {
			continue
		}
		s.AvailableQty = s.TotalStock
		s.StockValue = s.TotalStock * s.UnitCost
		s.IsLow = s.MinLevel > 0 && s.TotalStock <= s.MinLevel
		s.NeedsReorder = s.ReorderPoint > 0 && s.TotalStock <= s.ReorderPoint
		summaries = append(summaries, s)
	}
	if summaries == nil {
		summaries = []*domain.StockSummary{}
	}
	return summaries, nil
}

// ─── Inventory Stats ──────────────────────────────────────────────────────────

func (r *InventoryRepository) GetInventoryStats(tenantID string) (*domain.InventoryStats, error) {
	stats := &domain.InventoryStats{}

	r.db.QueryRow(`SELECT COUNT(*) FROM products WHERE tenant_id=?`, tenantID).
		Scan(&stats.TotalProducts)
	r.db.QueryRow(`SELECT COUNT(*) FROM products WHERE tenant_id=? AND is_active=1`, tenantID).
		Scan(&stats.ActiveProducts)
	r.db.QueryRow(`SELECT COUNT(*) FROM product_categories WHERE tenant_id=?`, tenantID).
		Scan(&stats.TotalCategories)
	r.db.QueryRow(`SELECT COUNT(*) FROM warehouses WHERE tenant_id=? AND is_active=1`, tenantID).
		Scan(&stats.TotalWarehouses)

	// Low stock count: active tracked products where current_stock <= min_stock_level
	r.db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT p.id FROM products p
			LEFT JOIN stock_moves sm ON sm.product_id = p.id AND sm.state='done'
			WHERE p.tenant_id=? AND p.is_active=1 AND p.track_inventory=1 AND p.min_stock_level > 0
			GROUP BY p.id, p.min_stock_level
			HAVING COALESCE(
				SUM(CASE WHEN sm.to_location_id IS NOT NULL THEN sm.quantity ELSE 0 END) -
				SUM(CASE WHEN sm.from_location_id IS NOT NULL THEN sm.quantity ELSE 0 END),
				0) <= p.min_stock_level
		)`, tenantID).Scan(&stats.LowStockCount)

	// Out of stock
	r.db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT p.id FROM products p
			LEFT JOIN stock_moves sm ON sm.product_id = p.id AND sm.state='done'
			WHERE p.tenant_id=? AND p.is_active=1 AND p.track_inventory=1
			GROUP BY p.id
			HAVING COALESCE(
				SUM(CASE WHEN sm.to_location_id IS NOT NULL THEN sm.quantity ELSE 0 END) -
				SUM(CASE WHEN sm.from_location_id IS NOT NULL THEN sm.quantity ELSE 0 END),
				0) <= 0
		)`, tenantID).Scan(&stats.OutOfStock)

	// Total stock value
	r.db.QueryRow(`
		SELECT COALESCE(SUM(stock_value),0) FROM (
			SELECT p.id,
				p.cost_price * COALESCE(
					SUM(CASE WHEN sm.to_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END) -
					SUM(CASE WHEN sm.from_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END),
					0) AS stock_value
			FROM products p
			LEFT JOIN stock_moves sm ON sm.product_id = p.id
			WHERE p.tenant_id=? AND p.is_active=1
			GROUP BY p.id, p.cost_price
		)`, tenantID).Scan(&stats.TotalStockValue)

	// Moves this month
	r.db.QueryRow(`
		SELECT COUNT(*) FROM stock_moves
		WHERE tenant_id=? AND strftime('%Y-%m', created_at) = strftime('%Y-%m', 'now')
		`, tenantID).Scan(&stats.MovesThisMonth)

	// Incoming value this month
	r.db.QueryRow(`
		SELECT COALESCE(SUM(quantity*unit_cost),0) FROM stock_moves
		WHERE tenant_id=? AND move_type='in' AND state='done'
		AND strftime('%Y-%m', created_at) = strftime('%Y-%m', 'now')
		`, tenantID).Scan(&stats.IncomingValue)

	// Outgoing value this month
	r.db.QueryRow(`
		SELECT COALESCE(SUM(quantity*unit_cost),0) FROM stock_moves
		WHERE tenant_id=? AND move_type='out' AND state='done'
		AND strftime('%Y-%m', created_at) = strftime('%Y-%m', 'now')
		`, tenantID).Scan(&stats.OutgoingValue)

	return stats, nil
}

// ─── Batches ──────────────────────────────────────────────────────────────────

func (r *InventoryRepository) ListBatches(tenantID, productID string) ([]*domain.Batch, error) {
	where := "b.tenant_id = ?"
	args := []interface{}{tenantID}
	if productID != "" {
		where += " AND b.product_id = ?"
		args = append(args, productID)
	}

	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT b.id, b.tenant_id, b.product_id,
		       COALESCE(json_extract(p.name,'$.en'), p.sku, ''),
		       b.batch_number,
		       COALESCE(b.manufacture_date,''),
		       COALESCE(b.expiry_date,''),
		       b.qty_received, b.qty_remaining,
		       COALESCE(b.supplier_id,''),
		       b.created_at
		FROM batches b
		LEFT JOIN products p ON p.id = b.product_id
		WHERE %s
		ORDER BY b.created_at DESC`, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batches []*domain.Batch
	for rows.Next() {
		b := &domain.Batch{}
		var mfgStr, expStr, createdAtStr string
		if err := rows.Scan(&b.ID, &b.TenantID, &b.ProductID, &b.ProductName,
			&b.BatchNumber, &mfgStr, &expStr,
			&b.QtyReceived, &b.QtyRemaining, &b.SupplierID, &createdAtStr); err != nil {
			continue
		}
		if mfgStr != "" {
			if t, err := parseTime(mfgStr); err == nil {
				b.ManufactureDate = &t
			}
		}
		if expStr != "" {
			if t, err := parseTime(expStr); err == nil {
				b.ExpiryDate = &t
			}
		}
		if t, err := parseTime(createdAtStr); err == nil {
			b.CreatedAt = t
		}
		batches = append(batches, b)
	}
	if batches == nil {
		batches = []*domain.Batch{}
	}
	return batches, nil
}

func (r *InventoryRepository) CreateBatch(b *domain.Batch) error {
	b.ID = uuid.New().String()
	b.CreatedAt = time.Now()
	mfgDate := ""
	if b.ManufactureDate != nil {
		mfgDate = b.ManufactureDate.Format("2006-01-02")
	}
	expDate := ""
	if b.ExpiryDate != nil {
		expDate = b.ExpiryDate.Format("2006-01-02")
	}
	_, err := r.db.Exec(`
		INSERT INTO batches
		(id, tenant_id, product_id, batch_number, manufacture_date, expiry_date,
		 qty_received, qty_remaining, supplier_id, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		b.ID, b.TenantID, b.ProductID, b.BatchNumber,
		mfgDate, expDate,
		b.QtyReceived, b.QtyReceived, // qty_remaining = qty_received initially
		nullStr(b.SupplierID), b.CreatedAt.Format(time.RFC3339))
	return err
}

// ─── Variants ─────────────────────────────────────────────────────────────────

func (r *InventoryRepository) ListVariants(productID, tenantID string) ([]*domain.ProductVariant, error) {
	rows, err := r.db.Query(`
		SELECT pv.id, pv.product_id,
		       COALESCE(json_extract(p.name,'$.en'), p.sku, ''),
		       pv.sku, COALESCE(pv.attributes,'{}'),
		       pv.price_modifier, pv.stock_qty,
		       COALESCE(pv.barcode,''), pv.is_active, pv.created_at
		FROM product_variants pv
		JOIN products p ON p.id = pv.product_id
		WHERE pv.product_id = ? AND p.tenant_id = ?
		ORDER BY pv.sku ASC`, productID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []*domain.ProductVariant
	for rows.Next() {
		v := &domain.ProductVariant{}
		var attrsStr, createdAtStr string
		var isActive int
		if err := rows.Scan(&v.ID, &v.ProductID, &v.ProductName,
			&v.SKU, &attrsStr,
			&v.PriceModifier, &v.StockQty, &v.Barcode, &isActive, &createdAtStr); err != nil {
			continue
		}
		v.IsActive = isActive == 1
		if attrsStr != "" {
			_ = json.Unmarshal([]byte(attrsStr), &v.Attributes)
		}
		if t, err := parseTime(createdAtStr); err == nil {
			v.CreatedAt = t
		}
		variants = append(variants, v)
	}
	if variants == nil {
		variants = []*domain.ProductVariant{}
	}
	return variants, nil
}

func (r *InventoryRepository) CreateVariant(v *domain.ProductVariant) error {
	v.ID = uuid.New().String()
	v.CreatedAt = time.Now()
	attrsJSON, _ := json.Marshal(v.Attributes)
	_, err := r.db.Exec(`
		INSERT INTO product_variants
		(id, product_id, sku, attributes, price_modifier, stock_qty, barcode, is_active, created_at)
		VALUES (?,?,?,?,?,?,?,1,?)`,
		v.ID, v.ProductID, v.SKU, string(attrsJSON),
		v.PriceModifier, v.StockQty, v.Barcode,
		v.CreatedAt.Format(time.RFC3339))
	return err
}

// ─── Low Stock ────────────────────────────────────────────────────────────────

func (r *InventoryRepository) GetLowStockProducts(tenantID string) ([]*domain.Product, error) {
	t := true
	products, _, err := r.ListProducts(domain.ListProductsFilter{
		TenantID: tenantID,
		IsActive: &t,
		LowStock: true,
		Page:     1,
		PageSize: 200,
	})
	return products, err
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func scanProduct(row interface{ Scan(...interface{}) error }) (*domain.Product, error) {
	p := &domain.Product{}
	var isActive, trackInv, trackBatch, trackSerial, hasExpiry int
	var nameEnStr, nameJSONStr, tagsStr, createdAtStr, updatedAtStr string
	var currentStock, stockValue float64

	err := row.Scan(
		&p.ID, &p.TenantID, &p.SKU,
		&nameEnStr, &nameJSONStr,
		&p.Description, &p.CategoryID, &p.CategoryName,
		&p.UOM, &p.BasePrice, &p.CostPrice, &p.SalePrice, &p.TaxRate,
		&p.Barcode, &p.QRCode,
		&trackInv, &trackBatch, &trackSerial, &hasExpiry,
		&p.MinStockLevel, &p.ReorderPoint,
		&isActive, &p.ImageURL, &tagsStr,
		&createdAtStr, &updatedAtStr,
		&currentStock, &stockValue,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product not found")
	}
	if err != nil {
		return nil, err
	}

	p.IsActive = isActive == 1
	p.TrackInventory = trackInv == 1
	p.TrackBatch = trackBatch == 1
	p.TrackSerial = trackSerial == 1
	p.HasExpiry = hasExpiry == 1
	p.CurrentStock = currentStock
	p.StockValue = stockValue
	p.AvailableQty = currentStock

	// Parse name JSON
	p.Name = map[string]string{"en": nameEnStr}
	if nameJSONStr != "" && nameJSONStr != nameEnStr {
		var nm map[string]string
		if err2 := json.Unmarshal([]byte(nameJSONStr), &nm); err2 == nil {
			p.Name = nm
		}
	}
	p.NameEn = p.Name["en"]

	// Parse tags
	if tagsStr != "" {
		_ = json.Unmarshal([]byte(tagsStr), &p.Tags)
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}

	if t, err := parseTime(createdAtStr); err == nil {
		p.CreatedAt = t
	}
	if t, err := parseTime(updatedAtStr); err == nil {
		p.UpdatedAt = t
	}

	return p, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func parseTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}
