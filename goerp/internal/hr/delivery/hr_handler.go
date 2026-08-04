package delivery

import (
	"github.com/gofiber/fiber/v2"
	"github.com/goerp/goerp/internal/hr/domain"
	"github.com/goerp/goerp/internal/hr/usecase"
	"github.com/goerp/goerp/internal/shared/middleware"
)

type HRHandler struct {
	uc *usecase.HRUsecase
}

func NewHRHandler(uc *usecase.HRUsecase) *HRHandler {
	return &HRHandler{uc: uc}
}

func (h *HRHandler) RegisterRoutes(app *fiber.App, auth fiber.Handler) {
	v1 := app.Group("/api/v1/hr", auth)

	// Employees
	v1.Get("/employees", h.ListEmployees)
	v1.Post("/employees", h.CreateEmployee)
	v1.Get("/employees/:id", h.GetEmployee)
	v1.Put("/employees/:id", h.UpdateEmployee)

	// Contracts
	v1.Get("/employees/:id/contract", h.GetEmployeeContract)
	v1.Post("/contracts", h.CreateContract)

	// Attendance
	v1.Get("/attendance", h.ListAttendance)
	v1.Post("/attendance", h.RecordAttendance)

	// Leave Requests
	v1.Get("/leaves", h.ListLeaveRequests)
	v1.Post("/leaves", h.CreateLeaveRequest)
	v1.Patch("/leaves/:id/approve", h.ApproveLeave)
	v1.Patch("/leaves/:id/reject", h.RejectLeave)

	// Payroll
	v1.Get("/payroll", h.ListPayrollRuns)
	v1.Post("/payroll/generate", h.GeneratePayroll)

	// Stats
	v1.Get("/stats", h.GetHRStats)
}

// ─── EMPLOYEES ────────────────────────────────────────────────────────────────

func (h *HRHandler) ListEmployees(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	emps, total, err := h.uc.ListEmployees(domain.EmployeeFilter{
		TenantID:   tenantID,
		Status:     c.Query("status"),
		Department: c.Query("department"),
		Page:       c.QueryInt("page", 1),
		Limit:      c.QueryInt("limit", 20),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": emps, "total": total})
}

func (h *HRHandler) GetEmployee(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	emp, err := h.uc.GetEmployee(c.Params("id"), tenantID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "employee not found"})
	}
	return c.JSON(emp)
}

func (h *HRHandler) CreateEmployee(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	claims := middleware.GetClaims(c)

	var req domain.CreateEmployeeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	emp, err := h.uc.CreateEmployee(req, tenantID, claims.UserID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(emp)
}

func (h *HRHandler) UpdateEmployee(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	id := c.Params("id")

	var req domain.UpdateEmployeeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	emp, err := h.uc.UpdateEmployee(id, req, tenantID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(emp)
}

// ─── CONTRACTS ────────────────────────────────────────────────────────────────

func (h *HRHandler) GetEmployeeContract(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	contract, err := h.uc.GetEmployeeContract(c.Params("id"), tenantID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "no active contract found"})
	}
	return c.JSON(contract)
}

func (h *HRHandler) CreateContract(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	claims := middleware.GetClaims(c)

	var req domain.CreateContractRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	contract, err := h.uc.CreateContract(req, tenantID, claims.UserID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(contract)
}

// ─── ATTENDANCE ───────────────────────────────────────────────────────────────

func (h *HRHandler) ListAttendance(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	atts, total, err := h.uc.ListAttendance(domain.AttendanceFilter{
		TenantID:   tenantID,
		EmployeeID: c.Query("employee_id"),
		Limit:      c.QueryInt("limit", 31),
		Page:       c.QueryInt("page", 1),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": atts, "total": total})
}

func (h *HRHandler) RecordAttendance(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)

	var req struct {
		EmployeeID string  `json:"employee_id"`
		Status     string  `json:"status"`
		CheckIn    *string `json:"check_in"`
		CheckOut   *string `json:"check_out"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	att, err := h.uc.RecordAttendance(req.EmployeeID, tenantID, req.Status, nil, nil)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(att)
}

// ─── LEAVE REQUESTS ───────────────────────────────────────────────────────────

func (h *HRHandler) ListLeaveRequests(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	leaves, total, err := h.uc.ListLeaveRequests(domain.LeaveFilter{
		TenantID:   tenantID,
		Status:     c.Query("status"),
		EmployeeID: c.Query("employee_id"),
		Page:       c.QueryInt("page", 1),
		Limit:      c.QueryInt("limit", 20),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": leaves, "total": total})
}

func (h *HRHandler) CreateLeaveRequest(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	claims := middleware.GetClaims(c)

	var req domain.CreateLeaveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	leave, err := h.uc.CreateLeaveRequest(req, tenantID, claims.UserID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(leave)
}

func (h *HRHandler) ApproveLeave(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	claims := middleware.GetClaims(c)
	if err := h.uc.ApproveLeave(c.Params("id"), tenantID, claims.UserID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "leave request approved"})
}

func (h *HRHandler) RejectLeave(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	var body struct {
		Reason string `json:"reason"`
	}
	c.BodyParser(&body)
	if err := h.uc.RejectLeave(c.Params("id"), tenantID, body.Reason); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "leave request rejected"})
}

// ─── PAYROLL ──────────────────────────────────────────────────────────────────

func (h *HRHandler) ListPayrollRuns(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	runs, err := h.uc.ListPayrollRuns(tenantID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": runs, "total": len(runs)})
}

func (h *HRHandler) GeneratePayroll(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	claims := middleware.GetClaims(c)

	var req domain.CreatePayrollRunRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	run, err := h.uc.GeneratePayroll(req, tenantID, claims.UserID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(run)
}

// ─── STATS ────────────────────────────────────────────────────────────────────

func (h *HRHandler) GetHRStats(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	stats, err := h.uc.GetHRStats(tenantID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(stats)
}
