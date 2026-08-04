package delivery

import (
	"github.com/goerp/goerp/internal/purchases/domain"
	"github.com/goerp/goerp/internal/purchases/usecase"
	"github.com/goerp/goerp/internal/shared/middleware"
	"github.com/gofiber/fiber/v2"
)

type PurchasesHandler struct{ uc *usecase.PurchasesUsecase }
func NewPurchasesHandler(uc *usecase.PurchasesUsecase) *PurchasesHandler { return &PurchasesHandler{uc: uc} }

func (h *PurchasesHandler) RegisterRoutes(app *fiber.App, auth fiber.Handler) {
	g := app.Group("/api/v1/purchases", auth)
	g.Get("/suppliers", h.ListSuppliers)
	g.Post("/suppliers", h.CreateSupplier)
	g.Get("/orders", h.ListOrders)
	g.Post("/orders", h.CreateOrder)
}

func (h *PurchasesHandler) ListSuppliers(c *fiber.Ctx) error {
	list, total, err := h.uc.ListSuppliers(middleware.GetTenantID(c), 1)
	if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
	if list == nil { list = []*domain.Supplier{} }
	return c.JSON(fiber.Map{"data": list, "total": total})
}
func (h *PurchasesHandler) CreateSupplier(c *fiber.Ctx) error {
	var s domain.Supplier
	if err := c.BodyParser(&s); err != nil { return c.Status(400).JSON(fiber.Map{"error":"invalid body"}) }
	s.TenantID = middleware.GetTenantID(c)
	if err := h.uc.CreateSupplier(&s); err != nil { return c.Status(400).JSON(fiber.Map{"error": err.Error()}) }
	return c.Status(201).JSON(s)
}
func (h *PurchasesHandler) ListOrders(c *fiber.Ctx) error {
	list, total, err := h.uc.ListPurchaseOrders(domain.PurchaseFilter{TenantID: middleware.GetTenantID(c), Page: 1})
	if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
	if list == nil { list = []*domain.PurchaseOrder{} }
	return c.JSON(fiber.Map{"data": list, "total": total})
}
func (h *PurchasesHandler) CreateOrder(c *fiber.Ctx) error {
	var o domain.PurchaseOrder
	if err := c.BodyParser(&o); err != nil { return c.Status(400).JSON(fiber.Map{"error":"invalid body"}) }
	o.TenantID = middleware.GetTenantID(c)
	if err := h.uc.CreatePurchaseOrder(&o); err != nil { return c.Status(400).JSON(fiber.Map{"error": err.Error()}) }
	return c.Status(201).JSON(o)
}
