package usecase

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
	"unicode"

	"github.com/goerp/goerp/internal/inventory/domain"
	"github.com/goerp/goerp/internal/inventory/repository"
	"github.com/goerp/goerp/internal/shared/events"
)

// ─── Usecase ──────────────────────────────────────────────────────────────────

type InventoryUsecase struct {
	repo *repository.InventoryRepository
}

func NewInventoryUsecase(repo *repository.InventoryRepository) *InventoryUsecase {
	return &InventoryUsecase{repo: repo}
}

// ─── Products ─────────────────────────────────────────────────────────────────

func (u *InventoryUsecase) ListProducts(f domain.ListProductsFilter) ([]*domain.Product, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 50
	}
	if f.SortDir != "asc" && f.SortDir != "desc" {
		f.SortDir = "desc"
	}
	allowedSort := map[string]bool{"name": true, "sku": true, "price": true, "stock": true, "created_at": true}
	if !allowedSort[f.SortBy] {
		f.SortBy = "created_at"
	}
	return u.repo.ListProducts(f)
}

func (u *InventoryUsecase) GetProduct(id, tenantID string) (*domain.Product, error) {
	if id == "" {
		return nil, fmt.Errorf("product ID is required")
	}
	return u.repo.GetProduct(id, tenantID)
}

func (u *InventoryUsecase) CreateProduct(p *domain.Product) error {
	// ── Mandatory field validation ────────────────────────────────────────────
	if p.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	nameEn := strings.TrimSpace(p.Name["en"])
	if nameEn == "" {
		return fmt.Errorf("product name (en) is required")
	}
	p.Name["en"] = nameEn

	// ── Auto-generate SKU if empty ────────────────────────────────────────────
	if strings.TrimSpace(p.SKU) == "" {
		p.SKU = generateSKU(nameEn)
	} else {
		p.SKU = strings.ToUpper(strings.TrimSpace(p.SKU))
	}

	// ── Price validation ──────────────────────────────────────────────────────
	if p.CostPrice < 0 {
		return fmt.Errorf("cost_price cannot be negative")
	}
	if p.SalePrice < 0 {
		return fmt.Errorf("sale_price cannot be negative")
	}
	if p.BasePrice < 0 {
		return fmt.Errorf("base_price cannot be negative")
	}
	if p.SalePrice > 0 && p.CostPrice > 0 && p.SalePrice < p.CostPrice {
		return fmt.Errorf("sale_price (%.2f) cannot be less than cost_price (%.2f)", p.SalePrice, p.CostPrice)
	}

	// ── Stock level validation ────────────────────────────────────────────────
	if p.MinStockLevel < 0 {
		return fmt.Errorf("min_stock_level cannot be negative")
	}
	if p.ReorderPoint < 0 {
		return fmt.Errorf("reorder_point cannot be negative")
	}
	if p.ReorderPoint > 0 && p.MinStockLevel > 0 && p.ReorderPoint < p.MinStockLevel {
		return fmt.Errorf("reorder_point (%.0f) should be greater than or equal to min_stock_level (%.0f)",
			p.ReorderPoint, p.MinStockLevel)
	}

	// ── Tax rate validation ───────────────────────────────────────────────────
	if p.TaxRate < 0 || p.TaxRate > 100 {
		return fmt.Errorf("tax_rate must be between 0 and 100")
	}

	// ── Set defaults ──────────────────────────────────────────────────────────
	if p.UOM == "" {
		p.UOM = "unit"
	}
	p.IsActive = true

	// ── Tags cleanup ──────────────────────────────────────────────────────────
	if len(p.Tags) > 0 {
		cleaned := make([]string, 0, len(p.Tags))
		for _, t := range p.Tags {
			t = strings.TrimSpace(strings.ToLower(t))
			if t != "" {
				cleaned = append(cleaned, t)
			}
		}
		p.Tags = cleaned
	}

	return u.repo.CreateProduct(p)
}

func (u *InventoryUsecase) UpdateProduct(p *domain.Product) error {
	if p.ID == "" {
		return fmt.Errorf("product ID is required")
	}
	if p.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}

	// Fetch current to ensure it exists + merge tenant guard
	existing, err := u.repo.GetProduct(p.ID, p.TenantID)
	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}

	// Protect immutable / computed fields
	p.TenantID = existing.TenantID
	p.CreatedAt = existing.CreatedAt

	// Validate name if provided
	if len(p.Name) > 0 {
		nameEn := strings.TrimSpace(p.Name["en"])
		if nameEn == "" {
			return fmt.Errorf("product name (en) cannot be empty")
		}
		p.Name["en"] = nameEn
	} else {
		p.Name = existing.Name
	}

	// Validate SKU
	if strings.TrimSpace(p.SKU) == "" {
		p.SKU = existing.SKU
	} else {
		p.SKU = strings.ToUpper(strings.TrimSpace(p.SKU))
	}

	// Price validations
	if p.CostPrice < 0 {
		return fmt.Errorf("cost_price cannot be negative")
	}
	if p.SalePrice < 0 {
		return fmt.Errorf("sale_price cannot be negative")
	}
	if p.SalePrice > 0 && p.CostPrice > 0 && p.SalePrice < p.CostPrice {
		return fmt.Errorf("sale_price (%.2f) cannot be less than cost_price (%.2f)", p.SalePrice, p.CostPrice)
	}
	if p.TaxRate < 0 || p.TaxRate > 100 {
		return fmt.Errorf("tax_rate must be between 0 and 100")
	}
	if p.MinStockLevel < 0 {
		return fmt.Errorf("min_stock_level cannot be negative")
	}
	if p.ReorderPoint < 0 {
		return fmt.Errorf("reorder_point cannot be negative")
	}

	// Tags cleanup
	if len(p.Tags) > 0 {
		cleaned := make([]string, 0, len(p.Tags))
		for _, t := range p.Tags {
			t = strings.TrimSpace(strings.ToLower(t))
			if t != "" {
				cleaned = append(cleaned, t)
			}
		}
		p.Tags = cleaned
	}

	return u.repo.UpdateProduct(p)
}

func (u *InventoryUsecase) DeleteProduct(id, tenantID string) error {
	if id == "" {
		return fmt.Errorf("product ID is required")
	}
	// Verify exists and belongs to tenant
	_, err := u.repo.GetProduct(id, tenantID)
	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}
	return u.repo.DeleteProduct(id, tenantID)
}

// ─── Categories ───────────────────────────────────────────────────────────────

func (u *InventoryUsecase) ListCategories(tenantID string) ([]*domain.ProductCategory, error) {
	return u.repo.ListCategories(tenantID)
}

func (u *InventoryUsecase) CreateCategory(cat *domain.ProductCategory) error {
	if cat.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	cat.Name = strings.TrimSpace(cat.Name)
	if cat.Name == "" {
		return fmt.Errorf("category name is required")
	}
	if len(cat.Name) > 100 {
		return fmt.Errorf("category name cannot exceed 100 characters")
	}
	cat.Description = strings.TrimSpace(cat.Description)
	return u.repo.CreateCategory(cat)
}

func (u *InventoryUsecase) UpdateCategory(cat *domain.ProductCategory) error {
	if cat.ID == "" {
		return fmt.Errorf("category ID is required")
	}
	cat.Name = strings.TrimSpace(cat.Name)
	if cat.Name == "" {
		return fmt.Errorf("category name is required")
	}
	return u.repo.UpdateCategory(cat)
}

func (u *InventoryUsecase) DeleteCategory(id, tenantID string) error {
	if id == "" {
		return fmt.Errorf("category ID is required")
	}
	return u.repo.DeleteCategory(id, tenantID)
}

// ─── Warehouses ───────────────────────────────────────────────────────────────

func (u *InventoryUsecase) ListWarehouses(tenantID string) ([]*domain.Warehouse, error) {
	return u.repo.ListWarehouses(tenantID)
}

func (u *InventoryUsecase) CreateWarehouse(wh *domain.Warehouse) error {
	if wh.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	wh.Name = strings.TrimSpace(wh.Name)
	if wh.Name == "" {
		return fmt.Errorf("warehouse name is required")
	}
	wh.Code = strings.ToUpper(strings.TrimSpace(wh.Code))
	if wh.Code == "" {
		wh.Code = generateCode(wh.Name, 5)
	}
	wh.IsActive = true
	return u.repo.CreateWarehouse(wh)
}

func (u *InventoryUsecase) UpdateWarehouse(wh *domain.Warehouse) error {
	if wh.ID == "" {
		return fmt.Errorf("warehouse ID is required")
	}
	wh.Name = strings.TrimSpace(wh.Name)
	if wh.Name == "" {
		return fmt.Errorf("warehouse name is required")
	}
	return u.repo.UpdateWarehouse(wh)
}

func (u *InventoryUsecase) DeleteWarehouse(id, tenantID string) error {
	if id == "" {
		return fmt.Errorf("warehouse ID is required")
	}
	return u.repo.DeleteWarehouse(id, tenantID)
}

// ─── Locations ────────────────────────────────────────────────────────────────

func (u *InventoryUsecase) ListLocations(tenantID, warehouseID string) ([]*domain.StockLocation, error) {
	return u.repo.ListLocations(warehouseID)
}

func (u *InventoryUsecase) CreateLocation(loc *domain.StockLocation) error {
	if loc.WarehouseID == "" {
		return fmt.Errorf("warehouse_id is required")
	}
	loc.Name = strings.TrimSpace(loc.Name)
	if loc.Name == "" {
		return fmt.Errorf("location name is required")
	}
	loc.Code = strings.ToUpper(strings.TrimSpace(loc.Code))
	if loc.Code == "" {
		loc.Code = generateCode(loc.Name, 6)
	}
	if loc.LocationType == "" {
		loc.LocationType = domain.LocationTypeInternal
	}
	return u.repo.CreateLocation(loc)
}

// ─── Stock Moves ──────────────────────────────────────────────────────────────

func (u *InventoryUsecase) ListStockMoves(f domain.ListStockMovesFilter) ([]*domain.StockMove, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 50
	}
	return u.repo.ListStockMoves(f)
}

func (u *InventoryUsecase) CreateStockMove(m *domain.StockMove) error {
	// ── Mandatory field validation ────────────────────────────────────────────
	if m.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if m.ProductID == "" {
		return fmt.Errorf("product_id is required")
	}
	if m.Quantity <= 0 {
		return fmt.Errorf("quantity must be greater than zero")
	}

	// ── Move type validation ──────────────────────────────────────────────────
	validTypes := map[string]bool{
		domain.MoveTypeIn:       true,
		domain.MoveTypeOut:      true,
		domain.MoveTypeTransfer: true,
		domain.MoveTypeAdjust:   true,
		domain.MoveTypeReturn:   true,
		domain.MoveTypeScrap:    true,
	}
	if !validTypes[m.MoveType] {
		return fmt.Errorf("invalid move_type: must be one of in, out, transfer, adjust, return, scrap")
	}

	// ── Location validation based on move type ────────────────────────────────
	switch m.MoveType {
	case domain.MoveTypeIn:
		if m.ToLocationID == "" {
			return fmt.Errorf("to_location_id is required for inbound moves")
		}
	case domain.MoveTypeOut, domain.MoveTypeScrap:
		if m.FromLocationID == "" {
			return fmt.Errorf("from_location_id is required for outbound moves")
		}
	case domain.MoveTypeTransfer:
		if m.FromLocationID == "" || m.ToLocationID == "" {
			return fmt.Errorf("both from_location_id and to_location_id are required for transfers")
		}
		if m.FromLocationID == m.ToLocationID {
			return fmt.Errorf("from_location_id and to_location_id cannot be the same for transfers")
		}
	case domain.MoveTypeAdjust:
		if m.Notes == "" {
			return fmt.Errorf("notes are required for stock adjustments (explain the reason)")
		}
	case domain.MoveTypeReturn:
		if m.Reference == "" {
			return fmt.Errorf("reference is required for return moves")
		}
	}

	// ── Unit cost validation ──────────────────────────────────────────────────
	if m.UnitCost < 0 {
		return fmt.Errorf("unit_cost cannot be negative")
	}

	// ── Auto-generate reference if empty ─────────────────────────────────────
	if m.Reference == "" {
		m.Reference = generateMoveReference(m.MoveType)
	}

	// ── Set state to done and record done_at ──────────────────────────────────
	m.State = domain.MoveStateDone
	now := time.Now()
	m.DoneAt = &now

	if err := u.repo.CreateStockMove(m); err != nil {
		return err
	}

	// ── Publish event for accounting/reporting integration ────────────────────
	events.Publish(events.StockMoved, m)

	// ── Reorder alert check ───────────────────────────────────────────────────
	if m.MoveType == domain.MoveTypeOut || m.MoveType == domain.MoveTypeScrap {
		go u.checkReorderAlert(m.ProductID, m.TenantID)
	}

	return nil
}

// ─── Stock Adjustment (dedicated endpoint) ───────────────────────────────────

func (u *InventoryUsecase) AdjustStock(adj *domain.StockAdjustment, tenantID, userID string) error {
	if tenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if adj.ProductID == "" {
		return fmt.Errorf("product_id is required")
	}
	if adj.Quantity == 0 {
		return fmt.Errorf("quantity cannot be zero for an adjustment")
	}
	if adj.Reason == "" {
		return fmt.Errorf("reason is required for stock adjustments")
	}

	// Validate product exists
	_, err := u.repo.GetProduct(adj.ProductID, tenantID)
	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}

	// Build adjustment move: positive = add stock, negative = remove stock
	now := time.Now()
	move := &domain.StockMove{
		TenantID:  tenantID,
		ProductID: adj.ProductID,
		Quantity:  abs(adj.Quantity),
		MoveType:  domain.MoveTypeAdjust,
		Reference: generateMoveReference(domain.MoveTypeAdjust),
		Notes:     fmt.Sprintf("Reason: %s. %s", adj.Reason, adj.Notes),
		State:     domain.MoveStateDone,
		CreatedBy: userID,
		DoneAt:    &now,
	}

	// Positive = inbound to location, Negative = outbound from location
	if adj.Quantity > 0 {
		move.ToLocationID = adj.LocationID
	} else {
		move.FromLocationID = adj.LocationID
	}

	if err := u.repo.CreateStockMove(move); err != nil {
		return err
	}

	events.Publish(events.StockMoved, move)

	// Check reorder if reducing stock
	if adj.Quantity < 0 {
		go u.checkReorderAlert(adj.ProductID, tenantID)
	}

	return nil
}

// ─── Stock Summary ────────────────────────────────────────────────────────────

func (u *InventoryUsecase) GetStockSummary(tenantID string) ([]*domain.StockSummary, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	return u.repo.GetStockSummary(tenantID)
}

func (u *InventoryUsecase) GetLowStockProducts(tenantID string) ([]*domain.Product, int, error) {
	if tenantID == "" {
		return nil, 0, fmt.Errorf("tenant_id is required")
	}
	t := true
	f := domain.ListProductsFilter{
		TenantID: tenantID,
		IsActive: &t,
		LowStock: true,
		Page:     1,
		PageSize: 200,
	}
	return u.repo.ListProducts(f)
}

// ─── Inventory Stats ──────────────────────────────────────────────────────────

func (u *InventoryUsecase) GetInventoryStats(tenantID string) (*domain.InventoryStats, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	return u.repo.GetInventoryStats(tenantID)
}

// ─── Batches ──────────────────────────────────────────────────────────────────

func (u *InventoryUsecase) ListBatches(tenantID, productID string) ([]*domain.Batch, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	return u.repo.ListBatches(tenantID, productID)
}

func (u *InventoryUsecase) CreateBatch(b *domain.Batch) error {
	if b.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if b.ProductID == "" {
		return fmt.Errorf("product_id is required")
	}
	b.BatchNumber = strings.TrimSpace(b.BatchNumber)
	if b.BatchNumber == "" {
		return fmt.Errorf("batch_number is required")
	}
	if b.QtyReceived <= 0 {
		return fmt.Errorf("qty_received must be greater than zero")
	}

	// Validate product exists and has batch tracking enabled
	prod, err := u.repo.GetProduct(b.ProductID, b.TenantID)
	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}
	if !prod.TrackBatch {
		return fmt.Errorf("product '%s' does not have batch tracking enabled", prod.Name["en"])
	}

	// Expiry date validation
	if b.ExpiryDate != nil && b.ManufactureDate != nil {
		if b.ExpiryDate.Before(*b.ManufactureDate) {
			return fmt.Errorf("expiry_date cannot be before manufacture_date")
		}
	}
	if b.ExpiryDate != nil && b.ExpiryDate.Before(time.Now()) {
		return fmt.Errorf("expiry_date cannot be in the past for new batches")
	}

	// Set remaining qty to received initially
	b.QtyRemaining = b.QtyReceived

	return u.repo.CreateBatch(b)
}

// ─── Product Variants ─────────────────────────────────────────────────────────

func (u *InventoryUsecase) ListVariants(tenantID, productID string) ([]*domain.ProductVariant, error) {
	if productID == "" {
		return nil, fmt.Errorf("product_id is required")
	}
	return u.repo.ListVariants(tenantID, productID)
}

func (u *InventoryUsecase) CreateVariant(v *domain.ProductVariant) error {
	if v.ProductID == "" {
		return fmt.Errorf("product_id is required")
	}
	v.SKU = strings.ToUpper(strings.TrimSpace(v.SKU))
	if v.SKU == "" {
		return fmt.Errorf("sku is required for variant")
	}
	if v.PriceModifier < -100 {
		return fmt.Errorf("price_modifier cannot be less than -100%%")
	}
	v.IsActive = true
	return u.repo.CreateVariant(v)
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// generateSKU creates an SKU from the product name: uppercase initials + random 4-digit suffix.
// Example: "Laptop Pro 15" -> "LP15-3842"
func generateSKU(name string) string {
	words := strings.Fields(name)
	prefix := ""
	for _, w := range words {
		runes := []rune(w)
		if len(runes) > 0 && unicode.IsLetter(runes[0]) {
			prefix += strings.ToUpper(string(runes[0]))
		}
		if len(prefix) >= 4 {
			break
		}
	}
	if prefix == "" {
		prefix = "PROD"
	}
	// Pad to at least 2 chars
	if len(prefix) < 2 {
		prefix = prefix + "X"
	}
	suffix := fmt.Sprintf("%04d", rand.Intn(10000)) //nolint:gosec
	return prefix + "-" + suffix
}

// generateCode creates a short uppercase code from a name.
func generateCode(name string, maxLen int) string {
	words := strings.Fields(name)
	code := ""
	for _, w := range words {
		w = strings.ToUpper(w)
		for _, r := range w {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				code += string(r)
			}
		}
		if len(code) >= maxLen {
			break
		}
	}
	if len(code) > maxLen {
		code = code[:maxLen]
	}
	if code == "" {
		code = fmt.Sprintf("LOC%03d", rand.Intn(1000)) //nolint:gosec
	}
	return code
}

// generateMoveReference creates a reference number for stock moves.
func generateMoveReference(moveType string) string {
	prefix := map[string]string{
		domain.MoveTypeIn:       "IN",
		domain.MoveTypeOut:      "OUT",
		domain.MoveTypeTransfer: "TRF",
		domain.MoveTypeAdjust:   "ADJ",
		domain.MoveTypeReturn:   "RET",
		domain.MoveTypeScrap:    "SCR",
	}
	p, ok := prefix[moveType]
	if !ok {
		p = "MOV"
	}
	ts := time.Now().Format("20060102")
	seq := fmt.Sprintf("%05d", rand.Intn(100000)) //nolint:gosec
	return fmt.Sprintf("%s-%s-%s", p, ts, seq)
}

// checkReorderAlert runs asynchronously to publish a reorder alert event if
// current stock falls at or below the product's reorder_point.
func (u *InventoryUsecase) checkReorderAlert(productID, tenantID string) {
	prod, err := u.repo.GetProduct(productID, tenantID)
	if err != nil || prod == nil {
		return
	}
	if prod.ReorderPoint > 0 && prod.CurrentStock <= prod.ReorderPoint {
		events.Publish(events.StockLow, map[string]interface{}{
			"product_id":    productID,
			"product_name":  prod.Name["en"],
			"sku":           prod.SKU,
			"current_stock": prod.CurrentStock,
			"reorder_point": prod.ReorderPoint,
			"tenant_id":     tenantID,
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
