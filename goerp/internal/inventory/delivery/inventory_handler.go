package delivery

import (
	"github.com/goerp/goerp/internal/inventory/domain"
	"github.com/goerp/goerp/internal/inventory/usecase"
	"github.com/goerp/goerp/internal/shared/middleware"
	"github.com/gofiber/fiber/v2"
)

// ─── Handler ──────────────────────────────────────────────────────────────────

type InventoryHandler struct {
	uc *usecase.InventoryUsecase
}

func NewInventoryHandler(uc *usecase.InventoryUsecase) *InventoryHandler {
	return &InventoryHandler{uc: uc}
}

// ─── Route Registration ───────────────────────────────────────────────────────

func (h *InventoryHandler) RegisterRoutes(app *fiber.App, auth fiber.Handler) {
	inv := app.Group("/api/v1/inventory", auth)

	// ── Products ────────────────────────────────────────────────────────────
	inv.Get("/products", h.ListProducts)
	inv.Post("/products", h.CreateProduct)
	inv.Get("/products/low-stock", h.GetLowStockProducts)
	inv.Get("/products/:id", h.GetProduct)
	inv.Put("/products/:id", h.UpdateProduct)
	inv.Delete("/products/:id", h.DeleteProduct)
	inv.Get("/products/:id/variants", h.ListVariants)
	inv.Post("/products/:id/variants", h.CreateVariant)

	// ── Categories ──────────────────────────────────────────────────────────
	inv.Get("/categories", h.ListCategories)
	inv.Post("/categories", h.CreateCategory)
	inv.Put("/categories/:id", h.UpdateCategory)
	inv.Delete("/categories/:id", h.DeleteCategory)

	// ── Warehouses ──────────────────────────────────────────────────────────
	inv.Get("/warehouses", h.ListWarehouses)
	inv.Post("/warehouses", h.CreateWarehouse)
	inv.Put("/warehouses/:id", h.UpdateWarehouse)
	inv.Delete("/warehouses/:id", h.DeleteWarehouse)

	// ── Locations ───────────────────────────────────────────────────────────
	inv.Get("/locations", h.ListLocations)
	inv.Post("/locations", h.CreateLocation)

	// ── Stock Moves ─────────────────────────────────────────────────────────
	inv.Get("/stock-moves", h.ListStockMoves)
	inv.Post("/stock-moves", h.CreateStockMove)

	// ── Stock Adjustment ────────────────────────────────────────────────────
	inv.Post("/adjust", h.AdjustStock)

	// ── Batches ─────────────────────────────────────────────────────────────
	inv.Get("/batches", h.ListBatches)
	inv.Post("/batches", h.CreateBatch)

	// ── Summary & Stats ─────────────────────────────────────────────────────
	inv.Get("/stock-summary", h.GetStockSummary)
	inv.Get("/stats", h.GetInventoryStats)
}

// ─── Products ─────────────────────────────────────────────────────────────────

// GET /api/v1/inventory/products
// Query: search, category_id, is_active (bool), low_stock (bool), page, page_size, sort_by, sort_dir
func (h *InventoryHandler) ListProducts(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)

	f := domain.ListProductsFilter{
		TenantID:   tenantID,
		Search:     c.Query("search"),
		CategoryID: c.Query("category_id"),
		Page:       clampMin(c.QueryInt("page", 1), 1),
		PageSize:   clampRange(c.QueryInt("page_size", 50), 1, 200),
		SortBy:     c.Query("sort_by", "created_at"),
		SortDir:    c.Query("sort_dir", "desc"),
	}

	// is_active filter
	if raw := c.Query("is_active"); raw != "" {
		v := raw == "true" || raw == "1"
		f.IsActive = &v
	}

	// low_stock filter
	if c.Query("low_stock") == "true" || c.Query("low_stock") == "1" {
		f.LowStock = true
	}

	products, total, err := h.uc.ListProducts(f)
	if err != nil {
		return c.Status(500).JSON(errResp(err))
	}
	if products == nil {
		products = []*domain.Product{}
	}
	return c.JSON(fiber.Map{
		"data":      products,
		"total":     total,
		"page":      f.Page,
		"page_size": f.PageSize,
	})
}

// GET /api/v1/inventory/products/low-stock
func (h *InventoryHandler) GetLowStockProducts(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	products, total, err := h.uc.GetLowStockProducts(tenantID)
	if err != nil {
		return c.Status(500).JSON(errResp(err))
	}
	if products == nil {
		products = []*domain.Product{}
	}
	return c.JSON(fiber.Map{"data": products, "total": total})
}

// GET /api/v1/inventory/products/:id
func (h *InventoryHandler) GetProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	tenantID := middleware.GetTenantID(c)
	p, err := h.uc.GetProduct(id, tenantID)
	if err != nil {
		return c.Status(404).JSON(errResp(err))
	}
	return c.JSON(p)
}

// POST /api/v1/inventory/products
func (h *InventoryHandler) CreateProduct(c *fiber.Ctx) error {
	var p domain.Product
	if err := c.BodyParser(&p); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body: " + err.Error()})
	}
	p.TenantID = middleware.GetTenantID(c)

	// Ensure Name map is initialized
	if p.Name == nil {
		p.Name = make(map[string]string)
	}
	// Accept name_en as shorthand for Name["en"]
	if p.NameEn != "" && p.Name["en"] == "" {
		p.Name["en"] = p.NameEn
	}

	if err := h.uc.CreateProduct(&p); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.Status(201).JSON(p)
}

// PUT /api/v1/inventory/products/:id
func (h *InventoryHandler) UpdateProduct(c *fiber.Ctx) error {
	var p domain.Product
	if err := c.BodyParser(&p); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body: " + err.Error()})
	}
	p.ID = c.Params("id")
	p.TenantID = middleware.GetTenantID(c)

	if p.Name == nil {
		p.Name = make(map[string]string)
	}
	if p.NameEn != "" && p.Name["en"] == "" {
		p.Name["en"] = p.NameEn
	}

	if err := h.uc.UpdateProduct(&p); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.JSON(p)
}

// DELETE /api/v1/inventory/products/:id
func (h *InventoryHandler) DeleteProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	tenantID := middleware.GetTenantID(c)
	if err := h.uc.DeleteProduct(id, tenantID); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.JSON(fiber.Map{"message": "product deactivated", "id": id})
}

// ─── Variants ─────────────────────────────────────────────────────────────────

// GET /api/v1/inventory/products/:id/variants
func (h *InventoryHandler) ListVariants(c *fiber.Ctx) error {
	productID := c.Params("id")
	tenantID := middleware.GetTenantID(c)
	variants, err := h.uc.ListVariants(tenantID, productID)
	if err != nil {
		return c.Status(500).JSON(errResp(err))
	}
	if variants == nil {
		variants = []*domain.ProductVariant{}
	}
	return c.JSON(fiber.Map{"data": variants, "total": len(variants)})
}

// POST /api/v1/inventory/products/:id/variants
func (h *InventoryHandler) CreateVariant(c *fiber.Ctx) error {
	var v domain.ProductVariant
	if err := c.BodyParser(&v); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	v.ProductID = c.Params("id")
	if err := h.uc.CreateVariant(&v); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.Status(201).JSON(v)
}

// ─── Categories ───────────────────────────────────────────────────────────────

// GET /api/v1/inventory/categories
func (h *InventoryHandler) ListCategories(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	cats, err := h.uc.ListCategories(tenantID)
	if err != nil {
		return c.Status(500).JSON(errResp(err))
	}
	if cats == nil {
		cats = []*domain.ProductCategory{}
	}
	return c.JSON(fiber.Map{"data": cats, "total": len(cats)})
}

// POST /api/v1/inventory/categories
func (h *InventoryHandler) CreateCategory(c *fiber.Ctx) error {
	var cat domain.ProductCategory
	if err := c.BodyParser(&cat); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	cat.TenantID = middleware.GetTenantID(c)
	if err := h.uc.CreateCategory(&cat); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.Status(201).JSON(cat)
}

// PUT /api/v1/inventory/categories/:id
func (h *InventoryHandler) UpdateCategory(c *fiber.Ctx) error {
	var cat domain.ProductCategory
	if err := c.BodyParser(&cat); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	cat.ID = c.Params("id")
	cat.TenantID = middleware.GetTenantID(c)
	if err := h.uc.UpdateCategory(&cat); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.JSON(cat)
}

// DELETE /api/v1/inventory/categories/:id
func (h *InventoryHandler) DeleteCategory(c *fiber.Ctx) error {
	id := c.Params("id")
	tenantID := middleware.GetTenantID(c)
	if err := h.uc.DeleteCategory(id, tenantID); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.JSON(fiber.Map{"message": "category deleted", "id": id})
}

// ─── Warehouses ───────────────────────────────────────────────────────────────

// GET /api/v1/inventory/warehouses
func (h *InventoryHandler) ListWarehouses(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	wh, err := h.uc.ListWarehouses(tenantID)
	if err != nil {
		return c.Status(500).JSON(errResp(err))
	}
	if wh == nil {
		wh = []*domain.Warehouse{}
	}
	return c.JSON(fiber.Map{"data": wh, "total": len(wh)})
}

// POST /api/v1/inventory/warehouses
func (h *InventoryHandler) CreateWarehouse(c *fiber.Ctx) error {
	var wh domain.Warehouse
	if err := c.BodyParser(&wh); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	wh.TenantID = middleware.GetTenantID(c)
	if err := h.uc.CreateWarehouse(&wh); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.Status(201).JSON(wh)
}

// PUT /api/v1/inventory/warehouses/:id
func (h *InventoryHandler) UpdateWarehouse(c *fiber.Ctx) error {
	var wh domain.Warehouse
	if err := c.BodyParser(&wh); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	wh.ID = c.Params("id")
	wh.TenantID = middleware.GetTenantID(c)
	if err := h.uc.UpdateWarehouse(&wh); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.JSON(wh)
}

// DELETE /api/v1/inventory/warehouses/:id
func (h *InventoryHandler) DeleteWarehouse(c *fiber.Ctx) error {
	id := c.Params("id")
	tenantID := middleware.GetTenantID(c)
	if err := h.uc.DeleteWarehouse(id, tenantID); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.JSON(fiber.Map{"message": "warehouse deactivated", "id": id})
}

// ─── Locations ────────────────────────────────────────────────────────────────

// GET /api/v1/inventory/locations?warehouse_id=xxx
func (h *InventoryHandler) ListLocations(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	warehouseID := c.Query("warehouse_id")
	locs, err := h.uc.ListLocations(tenantID, warehouseID)
	if err != nil {
		return c.Status(500).JSON(errResp(err))
	}
	if locs == nil {
		locs = []*domain.StockLocation{}
	}
	return c.JSON(fiber.Map{"data": locs, "total": len(locs)})
}

// POST /api/v1/inventory/locations
func (h *InventoryHandler) CreateLocation(c *fiber.Ctx) error {
	var loc domain.StockLocation
	if err := c.BodyParser(&loc); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := h.uc.CreateLocation(&loc); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.Status(201).JSON(loc)
}

// ─── Stock Moves ──────────────────────────────────────────────────────────────

// GET /api/v1/inventory/stock-moves
// Query: product_id, move_type, state, date_from, date_to, page, page_size
func (h *InventoryHandler) ListStockMoves(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	f := domain.ListStockMovesFilter{
		TenantID:  tenantID,
		ProductID: c.Query("product_id"),
		MoveType:  c.Query("move_type"),
		State:     c.Query("state"),
		DateFrom:  c.Query("date_from"),
		DateTo:    c.Query("date_to"),
		Page:      clampMin(c.QueryInt("page", 1), 1),
		PageSize:  clampRange(c.QueryInt("page_size", 50), 1, 200),
	}
	moves, total, err := h.uc.ListStockMoves(f)
	if err != nil {
		return c.Status(500).JSON(errResp(err))
	}
	if moves == nil {
		moves = []*domain.StockMove{}
	}
	return c.JSON(fiber.Map{
		"data":      moves,
		"total":     total,
		"page":      f.Page,
		"page_size": f.PageSize,
	})
}

// POST /api/v1/inventory/stock-moves
func (h *InventoryHandler) CreateStockMove(c *fiber.Ctx) error {
	var m domain.StockMove
	if err := c.BodyParser(&m); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	m.TenantID = middleware.GetTenantID(c)
	m.CreatedBy = middleware.GetUserID(c)
	if err := h.uc.CreateStockMove(&m); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.Status(201).JSON(m)
}

// ─── Stock Adjustment ─────────────────────────────────────────────────────────

// POST /api/v1/inventory/adjust
// Body: { product_id, location_id, quantity (pos/neg), reason, notes }
func (h *InventoryHandler) AdjustStock(c *fiber.Ctx) error {
	var adj domain.StockAdjustment
	if err := c.BodyParser(&adj); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	if err := h.uc.AdjustStock(&adj, tenantID, userID); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.JSON(fiber.Map{
		"message":    "stock adjusted successfully",
		"product_id": adj.ProductID,
		"quantity":   adj.Quantity,
		"reason":     adj.Reason,
	})
}

// ─── Batches ──────────────────────────────────────────────────────────────────

// GET /api/v1/inventory/batches?product_id=xxx
func (h *InventoryHandler) ListBatches(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	productID := c.Query("product_id")
	batches, err := h.uc.ListBatches(tenantID, productID)
	if err != nil {
		return c.Status(500).JSON(errResp(err))
	}
	if batches == nil {
		batches = []*domain.Batch{}
	}
	return c.JSON(fiber.Map{"data": batches, "total": len(batches)})
}

// POST /api/v1/inventory/batches
func (h *InventoryHandler) CreateBatch(c *fiber.Ctx) error {
	var b domain.Batch
	if err := c.BodyParser(&b); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	b.TenantID = middleware.GetTenantID(c)
	if err := h.uc.CreateBatch(&b); err != nil {
		return c.Status(422).JSON(errResp(err))
	}
	return c.Status(201).JSON(b)
}

// ─── Summary & Stats ──────────────────────────────────────────────────────────

// GET /api/v1/inventory/stock-summary
func (h *InventoryHandler) GetStockSummary(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	s, err := h.uc.GetStockSummary(tenantID)
	if err != nil {
		return c.Status(500).JSON(errResp(err))
	}
	if s == nil {
		s = []*domain.StockSummary{}
	}
	return c.JSON(fiber.Map{"data": s, "total": len(s)})
}

// GET /api/v1/inventory/stats
func (h *InventoryHandler) GetInventoryStats(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	stats, err := h.uc.GetInventoryStats(tenantID)
	if err != nil {
		return c.Status(500).JSON(errResp(err))
	}
	return c.JSON(stats)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func errResp(err error) fiber.Map {
	return fiber.Map{"error": err.Error()}
}

func clampMin(v, min int) int {
	if v < min {
		return min
	}
	return v
}

func clampRange(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Kept for backwards compatibility with any code that used max1.
func max1(v int) int {
	return clampMin(v, 1)
}
