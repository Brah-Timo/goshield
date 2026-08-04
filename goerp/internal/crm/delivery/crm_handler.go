package delivery

import (
	"github.com/gofiber/fiber/v2"
	"github.com/goerp/goerp/internal/crm/domain"
	"github.com/goerp/goerp/internal/crm/usecase"
	"github.com/goerp/goerp/internal/shared/middleware"
)

type CRMHandler struct {
	uc *usecase.CRMUsecase
}

func NewCRMHandler(uc *usecase.CRMUsecase) *CRMHandler {
	return &CRMHandler{uc: uc}
}

func (h *CRMHandler) RegisterRoutes(app *fiber.App, auth fiber.Handler) {
	v1 := app.Group("/api/v1/crm", auth)

	// Leads
	v1.Get("/leads", h.ListLeads)
	v1.Post("/leads", h.CreateLead)
	v1.Get("/leads/:id", h.GetLead)
	v1.Put("/leads/:id", h.UpdateLead)
	v1.Delete("/leads/:id", h.DeleteLead)
	v1.Post("/leads/:id/convert", h.ConvertLeadToOpportunity)

	// Opportunities
	v1.Get("/opportunities", h.ListOpportunities)
	v1.Post("/opportunities", h.CreateOpportunity)
	v1.Get("/opportunities/:id", h.GetOpportunity)
	v1.Put("/opportunities/:id", h.UpdateOpportunity)
	v1.Patch("/opportunities/:id/won", h.MarkWon)
	v1.Patch("/opportunities/:id/lost", h.MarkLost)

	// Activities
	v1.Get("/activities", h.ListActivities)
	v1.Post("/activities", h.CreateActivity)
	v1.Patch("/activities/:id/complete", h.CompleteActivity)

	// Pipeline Stats
	v1.Get("/pipeline/stats", h.GetPipelineStats)
}

// ─── LEADS ────────────────────────────────────────────────────────────────────

func (h *CRMHandler) ListLeads(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	status := c.Query("status")

	leads, total, err := h.uc.ListLeads(domain.LeadFilter{
		TenantID: tenantID,
		Status:   status,
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": leads, "total": total, "page": page, "limit": limit})
}

func (h *CRMHandler) GetLead(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	id := c.Params("id")
	lead, err := h.uc.GetLead(id, tenantID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "lead not found"})
	}
	return c.JSON(lead)
}

func (h *CRMHandler) CreateLead(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	claims := middleware.GetClaims(c)

	var req domain.CreateLeadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	lead, err := h.uc.CreateLead(req, tenantID, claims.UserID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(lead)
}

func (h *CRMHandler) UpdateLead(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	id := c.Params("id")

	var req domain.UpdateLeadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	lead, err := h.uc.UpdateLead(id, req, tenantID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(lead)
}

func (h *CRMHandler) DeleteLead(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	id := c.Params("id")
	if err := h.uc.DeleteLead(id, tenantID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "lead deleted"})
}

func (h *CRMHandler) ConvertLeadToOpportunity(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	claims := middleware.GetClaims(c)
	leadID := c.Params("id")

	var req domain.CreateOpportunityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	opp, err := h.uc.ConvertToOpportunity(leadID, tenantID, claims.UserID, req)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(opp)
}

// ─── OPPORTUNITIES ────────────────────────────────────────────────────────────

func (h *CRMHandler) ListOpportunities(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	stage := c.Query("stage")

	opps, total, err := h.uc.ListOpportunities(domain.OpportunityFilter{
		TenantID: tenantID,
		Stage:    stage,
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": opps, "total": total, "page": page, "limit": limit})
}

func (h *CRMHandler) GetOpportunity(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	opp, err := h.uc.GetOpportunity(c.Params("id"), tenantID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "opportunity not found"})
	}
	return c.JSON(opp)
}

func (h *CRMHandler) CreateOpportunity(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	claims := middleware.GetClaims(c)

	var req domain.CreateOpportunityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	opp, err := h.uc.CreateOpportunity(req, tenantID, claims.UserID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(opp)
}

func (h *CRMHandler) UpdateOpportunity(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	id := c.Params("id")

	var req domain.UpdateOpportunityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	opp, err := h.uc.UpdateOpportunity(id, req, tenantID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(opp)
}

func (h *CRMHandler) MarkWon(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	opp, err := h.uc.WinOpportunity(c.Params("id"), tenantID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(opp)
}

func (h *CRMHandler) MarkLost(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	var body struct {
		Reason string `json:"reason"`
	}
	c.BodyParser(&body)
	opp, err := h.uc.LoseOpportunity(c.Params("id"), tenantID, body.Reason)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(opp)
}

// ─── ACTIVITIES ───────────────────────────────────────────────────────────────

func (h *CRMHandler) ListActivities(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	acts, total, err := h.uc.ListActivities(domain.ActivityFilter{
		TenantID:      tenantID,
		LeadID:        c.Query("lead_id"),
		OpportunityID: c.Query("opportunity_id"),
		Limit:         c.QueryInt("limit", 50),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": acts, "total": total})
}

func (h *CRMHandler) CreateActivity(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	claims := middleware.GetClaims(c)

	var req domain.CreateActivityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	act, err := h.uc.CreateActivity(req, tenantID, claims.UserID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(act)
}

func (h *CRMHandler) CompleteActivity(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	if err := h.uc.CompleteActivity(c.Params("id"), tenantID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "activity completed"})
}

func (h *CRMHandler) GetPipelineStats(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	stats, err := h.uc.GetPipelineStats(tenantID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(stats)
}
