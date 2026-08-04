package delivery
import (
	"github.com/goerp/goerp/internal/accounting/domain"
	"github.com/goerp/goerp/internal/accounting/usecase"
	"github.com/goerp/goerp/internal/shared/middleware"
	"github.com/gofiber/fiber/v2"
)
type AccountingHandler struct{ uc *usecase.AccountingUsecase }
func NewAccountingHandler(uc *usecase.AccountingUsecase) *AccountingHandler { return &AccountingHandler{uc: uc} }
func (h *AccountingHandler) RegisterRoutes(app *fiber.App, auth fiber.Handler) {
	g := app.Group("/api/v1/accounting", auth)
	g.Get("/accounts", h.ListAccounts)
	g.Get("/journal-entries", h.ListJournalEntries)
	g.Post("/journal-entries", h.CreateJournalEntry)
	g.Patch("/journal-entries/:id/post", h.PostJournalEntry)
}
func (h *AccountingHandler) ListAccounts(c *fiber.Ctx) error {
	list, err := h.uc.ListAccounts(middleware.GetTenantID(c))
	if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
	if list == nil { list = []*domain.Account{} }
	return c.JSON(fiber.Map{"data": list})
}
func (h *AccountingHandler) ListJournalEntries(c *fiber.Ctx) error {
	list, total, err := h.uc.ListJournalEntries(domain.AccountingFilter{TenantID: middleware.GetTenantID(c), Page: 1})
	if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
	if list == nil { list = []*domain.JournalEntry{} }
	return c.JSON(fiber.Map{"data": list, "total": total})
}
func (h *AccountingHandler) CreateJournalEntry(c *fiber.Ctx) error {
	var e domain.JournalEntry
	if err := c.BodyParser(&e); err != nil { return c.Status(400).JSON(fiber.Map{"error":"invalid body"}) }
	e.TenantID = middleware.GetTenantID(c)
	if err := h.uc.CreateJournalEntry(&e); err != nil { return c.Status(400).JSON(fiber.Map{"error": err.Error()}) }
	return c.Status(201).JSON(e)
}
func (h *AccountingHandler) PostJournalEntry(c *fiber.Ctx) error {
	if err := h.uc.PostJournalEntry(c.Params("id"), middleware.GetTenantID(c)); err != nil { return c.Status(400).JSON(fiber.Map{"error": err.Error()}) }
	return c.JSON(fiber.Map{"message": "posted"})
}
