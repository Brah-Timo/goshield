package delivery

import (
	"github.com/goerp/goerp/internal/sales/domain"
	"github.com/goerp/goerp/internal/sales/usecase"
	"github.com/goerp/goerp/internal/shared/middleware"
	"github.com/gofiber/fiber/v2"
)

type SalesHandler struct{ uc *usecase.SalesUsecase }

func NewSalesHandler(uc *usecase.SalesUsecase) *SalesHandler { return &SalesHandler{uc: uc} }

func (h *SalesHandler) RegisterRoutes(app *fiber.App, auth fiber.Handler) {
	g := app.Group("/api/v1/sales", auth)
	g.Get("/customers", h.ListCustomers)
	g.Post("/customers", h.CreateCustomer)
	g.Get("/customers/:id", h.GetCustomer)
	g.Get("/orders", h.ListOrders)
	g.Post("/orders", h.CreateOrder)
	g.Patch("/orders/:id/confirm", h.ConfirmOrder)
	g.Patch("/orders/:id/cancel", h.CancelOrder)
	g.Get("/invoices", h.ListInvoices)
	g.Post("/invoices", h.CreateInvoice)
}

func (h *SalesHandler) ListCustomers(c *fiber.Ctx) error {
	tid := middleware.GetTenantID(c)
	list, total, err := h.uc.ListCustomers(tid, c.Query("search"), pageQ(c))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if list == nil {
		list = []*domain.Customer{}
	}
	return c.JSON(fiber.Map{"data": list, "total": total})
}

func (h *SalesHandler) CreateCustomer(c *fiber.Ctx) error {
	var cust domain.Customer
	if err := c.BodyParser(&cust); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	cust.TenantID = middleware.GetTenantID(c)
	if err := h.uc.CreateCustomer(&cust); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(cust)
}

func (h *SalesHandler) GetCustomer(c *fiber.Ctx) error {
	cust, err := h.uc.GetCustomer(c.Params("id"), middleware.GetTenantID(c))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cust)
}

func (h *SalesHandler) ListOrders(c *fiber.Ctx) error {
	tid := middleware.GetTenantID(c)
	list, total, err := h.uc.ListSalesOrders(domain.SalesFilter{
		TenantID: tid, State: c.Query("state"),
		Search: c.Query("search"), Page: pageQ(c), PageSize: 50,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if list == nil {
		list = []*domain.SalesOrder{}
	}
	return c.JSON(fiber.Map{"data": list, "total": total})
}

func (h *SalesHandler) CreateOrder(c *fiber.Ctx) error {
	var o domain.SalesOrder
	if err := c.BodyParser(&o); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	o.TenantID = middleware.GetTenantID(c)
	if err := h.uc.CreateSalesOrder(&o); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(o)
}

func (h *SalesHandler) ConfirmOrder(c *fiber.Ctx) error {
	if err := h.uc.ConfirmOrder(c.Params("id"), middleware.GetTenantID(c)); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "order confirmed"})
}

func (h *SalesHandler) CancelOrder(c *fiber.Ctx) error {
	if err := h.uc.CancelOrder(c.Params("id"), middleware.GetTenantID(c)); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "order cancelled"})
}

func (h *SalesHandler) ListInvoices(c *fiber.Ctx) error {
	tid := middleware.GetTenantID(c)
	list, total, err := h.uc.ListInvoices(domain.SalesFilter{
		TenantID: tid, State: c.Query("state"), Page: pageQ(c), PageSize: 50,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if list == nil {
		list = []*domain.Invoice{}
	}
	return c.JSON(fiber.Map{"data": list, "total": total})
}

func (h *SalesHandler) CreateInvoice(c *fiber.Ctx) error {
	var inv domain.Invoice
	if err := c.BodyParser(&inv); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	inv.TenantID = middleware.GetTenantID(c)
	if err := h.uc.CreateInvoice(&inv); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(inv)
}

func pageQ(c *fiber.Ctx) int {
	p := c.QueryInt("page", 1)
	if p < 1 {
		return 1
	}
	return p
}
