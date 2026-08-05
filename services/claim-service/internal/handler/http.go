// Package handler provides the HTTP layer for the claim-service using Fiber.
package handler

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/goshield/pkg/middleware"
	"github.com/goshield/services/claim-service/internal/domain"
	"github.com/goshield/services/claim-service/internal/repository"
	"github.com/goshield/services/claim-service/internal/service"
)

const maxUploadBytes = 20 << 20 // 20 MB

// Handler groups all Fiber route handlers for the claim-service.
type Handler struct {
	svc      service.ClaimService
	validate *validator.Validate
	logger   *zap.Logger
}

// New creates a new Handler.
func New(svc service.ClaimService, logger *zap.Logger) *Handler {
	return &Handler{
		svc:      svc,
		validate: validator.New(),
		logger:   logger,
	}
}

// ─── Request / Response DTOs ─────────────────────────────────────────────────

type createClaimRequest struct {
	PolicyNumber string  `json:"policy_number" validate:"required,min=3,max=50"`
	ClaimType    string  `json:"claim_type"    validate:"required,oneof=HEALTH CAR PROPERTY LIFE TRAVEL OTHER"`
	Amount       float64 `json:"amount"        validate:"required,gt=0"`
	IncidentDate string  `json:"incident_date"` // RFC3339 date string
	Description  string  `json:"description"   validate:"max=2000"`
}

type reviewClaimRequest struct {
	Status       string `json:"status"        validate:"required,oneof=APPROVED REJECTED MORE_INFO"`
	AnalystNotes string `json:"analyst_notes" validate:"max=2000"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details any    `json:"details,omitempty"`
}

type successResponse struct {
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

// ─── Routes ──────────────────────────────────────────────────────────────────

// RegisterRoutes wires all claim routes onto the given Fiber router group.
func (h *Handler) RegisterRoutes(r fiber.Router) {
	claims := r.Group("/claims")
	claims.Post("/", h.CreateClaim)
	claims.Get("/", h.ListClaims)
	claims.Get("/stats", h.GetStats)
	claims.Get("/export", h.ExportClaims) // CSV export — must come before /:id
	claims.Get("/:id", h.GetClaim)
	claims.Post("/:id/document", h.UploadDocument)
	claims.Post("/:id/review", h.ReviewClaim)  // NEW-I fix: frontend POSTs to /:id/review
	claims.Patch("/:id/review", h.ReviewClaim) // keep PATCH alias for API compatibility
	claims.Delete("/:id", h.DeleteClaim)
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// CreateClaim godoc
// @Summary  Create a new insurance claim
// @Tags     claims
// @Accept   json
// @Produce  json
// @Param    body body createClaimRequest true "Claim payload"
// @Success  201 {object} successResponse
// @Router   /claims [post]
func (h *Handler) CreateClaim(c *fiber.Ctx) error {
	userID := middleware.UserIDFromContext(c.UserContext())
	companyID := middleware.CompanyIDFromContext(c.UserContext())
	if userID == "" || companyID == "" {
		return respondError(c, http.StatusUnauthorized, "missing identity in token", nil)
	}

	var req createClaimRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
	}
	if err := h.validate.Struct(req); err != nil {
		return respondError(c, http.StatusUnprocessableEntity, "validation failed", formatValidationErrors(err))
	}

	input := domain.CreateClaimInput{
		UserID:       userID,
		CompanyID:    companyID,
		PolicyNumber: req.PolicyNumber,
		ClaimType:    domain.ClaimType(req.ClaimType),
		Amount:       req.Amount,
		Description:  req.Description,
	}
	if req.IncidentDate != "" {
		t, err := time.Parse("2006-01-02", req.IncidentDate)
		if err == nil {
			input.IncidentDate = &t
		}
	}

	claim, err := h.svc.CreateClaim(c.Context(), input)
	if err != nil {
		h.logger.Error("create claim failed", zap.Error(err))
		return respondError(c, http.StatusInternalServerError, "failed to create claim", nil)
	}

	return c.Status(http.StatusCreated).JSON(successResponse{Data: claim})
}

// GetClaim godoc
// @Summary  Get a claim by ID
// @Tags     claims
// @Produce  json
// @Param    id path string true "Claim ID"
// @Success  200 {object} successResponse
// @Router   /claims/{id} [get]
func (h *Handler) GetClaim(c *fiber.Ctx) error {
	companyID := middleware.CompanyIDFromContext(c.UserContext())
	if companyID == "" {
		return respondError(c, http.StatusUnauthorized, "missing company identity", nil)
	}

	id := c.Params("id")
	if id == "" {
		return respondError(c, http.StatusBadRequest, "missing claim id", nil)
	}

	claim, err := h.svc.GetClaim(c.Context(), id, companyID)
	if err != nil {
		if err == repository.ErrNotFound {
			return respondError(c, http.StatusNotFound, "claim not found", nil)
		}
		h.logger.Error("get claim failed", zap.String("id", id), zap.Error(err))
		return respondError(c, http.StatusInternalServerError, "failed to retrieve claim", nil)
	}

	return c.JSON(successResponse{Data: claim})
}

// ListClaims godoc
// @Summary  List claims with filtering and pagination
// @Tags     claims
// @Produce  json
// @Param    status    query string false "Filter by status"
// @Param    page      query int    false "Page number"
// @Param    page_size query int    false "Page size"
// @Success  200 {object} successResponse
// @Router   /claims [get]
func (h *Handler) ListClaims(c *fiber.Ctx) error {
	companyID := middleware.CompanyIDFromContext(c.UserContext())
	if companyID == "" {
		return respondError(c, http.StatusUnauthorized, "missing company identity", nil)
	}

	filter := domain.ListFilter{
		CompanyID: companyID,
		Status:    domain.ClaimStatus(c.Query("status")),
		ClaimType: domain.ClaimType(c.Query("claim_type")),
		SortBy:    c.Query("sort_by", "created_at"),
		SortOrder: c.Query("sort_order", "desc"),
	}

	if v := c.QueryInt("page", 1); v > 0 {
		filter.Page = v
	}
	if v := c.QueryInt("page_size", 20); v > 0 {
		filter.PageSize = v
	}
	if v, err := strconv.ParseFloat(c.Query("min_amount"), 64); err == nil {
		filter.MinAmount = v
	}
	if v, err := strconv.ParseFloat(c.Query("max_amount"), 64); err == nil {
		filter.MaxAmount = v
	}
	if v := c.Query("date_from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.DateFrom = &t
		}
	}
	if v := c.Query("date_to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.DateTo = &t
		}
	}

	result, err := h.svc.ListClaims(c.Context(), filter)
	if err != nil {
		h.logger.Error("list claims failed", zap.Error(err))
		return respondError(c, http.StatusInternalServerError, "failed to list claims", nil)
	}

	return c.JSON(successResponse{Data: result})
}

// UploadDocument godoc
// @Summary  Upload a document (PDF/JPG/PNG) for a claim
// @Tags     claims
// @Accept   multipart/form-data
// @Produce  json
// @Param    id   path     string true  "Claim ID"
// @Param    file formData file   true  "Document file"
// @Success  200 {object} successResponse
// @Router   /claims/{id}/document [post]
func (h *Handler) UploadDocument(c *fiber.Ctx) error {
	companyID := middleware.CompanyIDFromContext(c.UserContext())
	if companyID == "" {
		return respondError(c, http.StatusUnauthorized, "missing company identity", nil)
	}

	claimID := c.Params("id")
	if claimID == "" {
		return respondError(c, http.StatusBadRequest, "missing claim id", nil)
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return respondError(c, http.StatusBadRequest, "file field required", err.Error())
	}
	if fileHeader.Size > maxUploadBytes {
		return respondError(c, http.StatusRequestEntityTooLarge,
			"file too large (max 20 MB)", nil)
	}

	f, err := fileHeader.Open()
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "failed to open upload", nil)
	}
	defer f.Close()

	// Buffer into memory so we can pass size accurately.
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, f); err != nil {
		return respondError(c, http.StatusInternalServerError, "failed to read upload", nil)
	}

	docURL, err := h.svc.UploadDocument(
		c.Context(),
		claimID, companyID,
		fileHeader.Filename,
		bytes.NewReader(buf.Bytes()),
		int64(buf.Len()),
	)
	if err != nil {
		h.logger.Error("upload document failed",
			zap.String("claim_id", claimID),
			zap.Error(err),
		)
		if err == repository.ErrNotFound {
			return respondError(c, http.StatusNotFound, "claim not found", nil)
		}
		return respondError(c, http.StatusInternalServerError, err.Error(), nil)
	}

	return c.JSON(successResponse{
		Data:    map[string]string{"doc_url": docURL},
		Message: "document uploaded successfully",
	})
}

// ReviewClaim godoc
// @Summary  Submit analyst review decision
// @Tags     claims
// @Accept   json
// @Produce  json
// @Param    id   path string true "Claim ID"
// @Param    body body reviewClaimRequest true "Review decision"
// @Success  200 {object} successResponse
// @Router   /claims/{id}/review [patch]
func (h *Handler) ReviewClaim(c *fiber.Ctx) error {
	userID := middleware.UserIDFromContext(c.UserContext())
	companyID := middleware.CompanyIDFromContext(c.UserContext())
	if userID == "" || companyID == "" {
		return respondError(c, http.StatusUnauthorized, "missing identity", nil)
	}

	claimID := c.Params("id")
	if claimID == "" {
		return respondError(c, http.StatusBadRequest, "missing claim id", nil)
	}

	var req reviewClaimRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
	}
	if err := h.validate.Struct(req); err != nil {
		return respondError(c, http.StatusUnprocessableEntity, "validation failed", formatValidationErrors(err))
	}

	if err := h.svc.ReviewClaim(c.Context(), domain.ReviewInput{
		ClaimID:      claimID,
		AnalystID:    userID,
		Status:       domain.ClaimStatus(req.Status),
		AnalystNotes: req.AnalystNotes,
	}); err != nil {
		if err == repository.ErrNotFound {
			return respondError(c, http.StatusNotFound, "claim not found", nil)
		}
		h.logger.Error("review claim failed", zap.String("claim_id", claimID), zap.Error(err))
		return respondError(c, http.StatusInternalServerError, "failed to review claim", nil)
	}

	return c.JSON(successResponse{Message: "claim reviewed successfully"})
}

// DeleteClaim godoc
// @Summary  Delete a claim (admin only)
// @Tags     claims
// @Produce  json
// @Param    id path string true "Claim ID"
// @Success  204
// @Router   /claims/{id} [delete]
func (h *Handler) DeleteClaim(c *fiber.Ctx) error {
	role := middleware.RoleFromContext(c.UserContext())
	if role != "ADMIN" {
		return respondError(c, http.StatusForbidden, "admin role required", nil)
	}

	companyID := middleware.CompanyIDFromContext(c.UserContext())
	claimID := c.Params("id")

	if err := h.svc.DeleteClaim(c.Context(), claimID, companyID); err != nil {
		if err == repository.ErrNotFound {
			return respondError(c, http.StatusNotFound, "claim not found", nil)
		}
		return respondError(c, http.StatusInternalServerError, "failed to delete claim", nil)
	}

	return c.SendStatus(http.StatusNoContent)
}

// GetStats godoc
// @Summary  Get daily fraud statistics
// @Tags     claims
// @Produce  json
// @Param    days query int false "Number of days (default 30)"
// @Success  200 {object} successResponse
// @Router   /claims/stats [get]
func (h *Handler) GetStats(c *fiber.Ctx) error {
	companyID := middleware.CompanyIDFromContext(c.UserContext())
	if companyID == "" {
		return respondError(c, http.StatusUnauthorized, "missing company identity", nil)
	}

	days := c.QueryInt("days", 30)

	stats, err := h.svc.GetDailyStats(c.Context(), companyID, days)
	if err != nil {
		h.logger.Error("get stats failed", zap.Error(err))
		return respondError(c, http.StatusInternalServerError, "failed to retrieve stats", nil)
	}

	return c.JSON(successResponse{Data: stats})
}

// ExportClaims godoc
// @Summary  Export claims as CSV
// @Tags     claims
// @Produce  text/csv
// @Param    status     query string false "Filter by status"
// @Param    claim_type query string false "Filter by claim type"
// @Param    date_from  query string false "Start date (YYYY-MM-DD)"
// @Param    date_to    query string false "End date (YYYY-MM-DD)"
// @Success  200 {string} string "CSV file"
// @Router   /claims/export [get]
func (h *Handler) ExportClaims(c *fiber.Ctx) error {
	companyID := middleware.CompanyIDFromContext(c.UserContext())
	if companyID == "" {
		return respondError(c, http.StatusUnauthorized, "missing company identity", nil)
	}

	// Re-use the list filter but with a large page size to fetch all matching claims.
	filter := domain.ListFilter{
		CompanyID: companyID,
		Status:    domain.ClaimStatus(c.Query("status")),
		ClaimType: domain.ClaimType(c.Query("claim_type")),
		Page:      1,
		PageSize:  10000, // export cap
		SortBy:    "created_at",
		SortOrder: "desc",
	}
	if v, err := strconv.ParseFloat(c.Query("min_amount"), 64); err == nil {
		filter.MinAmount = v
	}
	if v, err := strconv.ParseFloat(c.Query("max_amount"), 64); err == nil {
		filter.MaxAmount = v
	}
	if v := c.Query("date_from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.DateFrom = &t
		}
	}
	if v := c.Query("date_to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.DateTo = &t
		}
	}

	result, err := h.svc.ListClaims(c.Context(), filter)
	if err != nil {
		h.logger.Error("export claims failed", zap.Error(err))
		return respondError(c, http.StatusInternalServerError, "failed to export claims", nil)
	}

	// Build CSV in memory.
	var buf bytes.Buffer
	buf.WriteString("id,policy_number,claim_type,status,amount,fraud_score,risk_level,incident_date,created_at,reviewed_by,analyst_notes\n")
	for _, cl := range result.Claims {
		incidentDate := ""
		if cl.IncidentDate != nil {
			incidentDate = cl.IncidentDate.Format("2006-01-02")
		}
		row := []string{
			cl.ID,
			cl.PolicyNumber,
			string(cl.ClaimType),
			string(cl.Status),
			strconv.FormatFloat(cl.Amount, 'f', 2, 64),
			strconv.FormatFloat(cl.FraudScore, 'f', 4, 64),
			"", // risk_level not in domain.Claim yet — placeholder
			incidentDate,
			cl.CreatedAt.Format(time.RFC3339),
			cl.AnalystID,
			csvEscape(cl.AnalystNotes),
		}
		buf.WriteString(csvRow(row) + "\n")
	}

	filename := "goshield-claims-" + time.Now().Format("2006-01-02") + ".csv"
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	return c.Status(http.StatusOK).Send(buf.Bytes())
}

// csvEscape wraps a field in double-quotes if it contains commas, quotes, or newlines.
func csvEscape(s string) string {
	for _, ch := range s {
		if ch == ',' || ch == '"' || ch == '\n' || ch == '\r' {
			return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
		}
	}
	return s
}

// csvRow joins fields with commas, escaping each field.
func csvRow(fields []string) string {
	escaped := make([]string, len(fields))
	for i, f := range fields {
		escaped[i] = csvEscape(f)
	}
	return strings.Join(escaped, ",")
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func respondError(c *fiber.Ctx, status int, msg string, details any) error {
	return c.Status(status).JSON(errorResponse{
		Error:   msg,
		Code:    status,
		Details: details,
	})
}

func formatValidationErrors(err error) map[string]string {
	errs := make(map[string]string)
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			errs[fe.Field()] = fe.Tag()
		}
	} else {
		errs["validation"] = err.Error()
	}
	return errs
}

// healthResponse for liveness/readiness probes.
type healthResponse struct {
	Status    string         `json:"status"`
	Service   string         `json:"service"`
	Timestamp time.Time      `json:"timestamp"`
	Checks    map[string]any `json:"checks,omitempty"`
}

// Health returns service health status.
func Health(c *fiber.Ctx) error {
	return c.JSON(healthResponse{
		Status:    "ok",
		Service:   "claim-service",
		Timestamp: time.Now().UTC(),
	})
}


