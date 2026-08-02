// Package repository provides PostgreSQL data access for claims.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/goshield/pkg/database"
	"github.com/goshield/services/claim-service/internal/domain"
)

// ErrNotFound is returned when a claim does not exist.
var ErrNotFound = errors.New("claim not found")

// ErrForbidden is returned when a user attempts to access a claim from a different tenant.
var ErrForbidden = errors.New("access to this claim is forbidden")

// ClaimRepository defines the data access contract.
type ClaimRepository interface {
	Create(ctx context.Context, claim *domain.Claim) error
	GetByID(ctx context.Context, id, companyID string) (*domain.Claim, error)
	List(ctx context.Context, filter domain.ListFilter) (*domain.ListResult, error)
	UpdateDocURL(ctx context.Context, id, companyID, docURL string) error
	UpdateAnalysis(ctx context.Context, input domain.UpdateAnalysisInput) error
	UpdateStatus(ctx context.Context, input domain.ReviewInput) error
	UpdateStatusByID(ctx context.Context, claimID string, status domain.ClaimStatus) error
	Delete(ctx context.Context, id, companyID string) error
	GetDailyStats(ctx context.Context, companyID string, days int) ([]*domain.DailyStat, error)
}

type claimRepo struct {
	db     *database.DB
	logger *zap.Logger
}

// New creates a new ClaimRepository backed by PostgreSQL.
func New(db *database.DB, logger *zap.Logger) ClaimRepository {
	return &claimRepo{db: db, logger: logger}
}

// Create inserts a new claim record.
func (r *claimRepo) Create(ctx context.Context, claim *domain.Claim) error {
	const q = `
		INSERT INTO claims (
			id, user_id, company_id, policy_number, claim_type,
			amount, incident_date, doc_url, description,
			fraud_score, fraud_reason, risk_factors,
			status, analyst_id, analyst_notes,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12,
			$13, $14, $15,
			$16, $17
		)`

	riskJSON, err := json.Marshal(claim.RiskFactors)
	if err != nil {
		return fmt.Errorf("marshal risk_factors: %w", err)
	}

	now := time.Now().UTC()
	claim.CreatedAt = now
	claim.UpdatedAt = now

	_, err = r.db.Pool.Exec(ctx, q,
		claim.ID,
		claim.UserID,
		claim.CompanyID,
		claim.PolicyNumber,
		string(claim.ClaimType),
		claim.Amount,
		claim.IncidentDate,
		claim.DocURL,
		claim.Description,
		claim.FraudScore,
		claim.FraudReason,
		riskJSON,
		string(claim.Status),
		nullString(claim.AnalystID),
		nullString(claim.AnalystNotes),
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert claim: %w", err)
	}
	return nil
}

// GetByID retrieves a single claim scoped to the given company (multi-tenancy).
func (r *claimRepo) GetByID(ctx context.Context, id, companyID string) (*domain.Claim, error) {
	const q = `
		SELECT
			id, user_id, company_id, policy_number, claim_type,
			amount, incident_date, doc_url, description,
			fraud_score, fraud_reason, risk_factors,
			status, analyst_id, analyst_notes,
			created_at, updated_at
		FROM claims
		WHERE id = $1 AND company_id = $2`

	row := r.db.Pool.QueryRow(ctx, q, id, companyID)
	return scanClaim(row)
}

// List returns a paginated, filtered list of claims scoped to a company.
func (r *claimRepo) List(ctx context.Context, filter domain.ListFilter) (*domain.ListResult, error) {
	where, args := buildWhereClause(filter)

	// Count total.
	countQ := "SELECT COUNT(*) FROM claims" + where
	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count claims: %w", err)
	}

	// Sort + paginate.
	sortBy := "created_at"
	if filter.SortBy == "fraud_score" || filter.SortBy == "amount" {
		sortBy = filter.SortBy
	}
	sortOrder := "DESC"
	if strings.ToUpper(filter.SortOrder) == "ASC" {
		sortOrder = "ASC"
	}

	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	argIdx := len(args) + 1
	dataQ := fmt.Sprintf(`
		SELECT
			id, user_id, company_id, policy_number, claim_type,
			amount, incident_date, doc_url, description,
			fraud_score, fraud_reason, risk_factors,
			status, analyst_id, analyst_notes,
			created_at, updated_at
		FROM claims%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		where, sortBy, sortOrder, argIdx, argIdx+1)

	args = append(args, pageSize, offset)
	rows, err := r.db.Pool.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, fmt.Errorf("query claims: %w", err)
	}
	defer rows.Close()

	var claims []*domain.Claim
	for rows.Next() {
		c, err := scanClaimRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan claim row: %w", err)
		}
		claims = append(claims, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claims: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	return &domain.ListResult{
		Claims:     claims,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateDocURL sets the storage URL for a claim's document after upload.
func (r *claimRepo) UpdateDocURL(ctx context.Context, id, companyID, docURL string) error {
	const q = `
		UPDATE claims
		SET doc_url = $1, updated_at = NOW()
		WHERE id = $2 AND company_id = $3`

	tag, err := r.db.Pool.Exec(ctx, q, docURL, id, companyID)
	if err != nil {
		return fmt.Errorf("update doc_url: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateAnalysis applies AI analysis results and transitions status.
func (r *claimRepo) UpdateAnalysis(ctx context.Context, input domain.UpdateAnalysisInput) error {
	riskJSON, err := json.Marshal(input.RiskFactors)
	if err != nil {
		return fmt.Errorf("marshal risk_factors: %w", err)
	}

	const q = `
		UPDATE claims
		SET
			fraud_score   = $1,
			fraud_reason  = $2,
			risk_factors  = $3,
			status        = CASE
				WHEN $1 >= 0.80 THEN 'FLAGGED'::claim_status
				ELSE 'APPROVED'::claim_status
			END,
			updated_at    = NOW()
		WHERE id = $4`

	tag, err := r.db.Pool.Exec(ctx, q,
		input.FraudScore,
		input.FraudReason,
		riskJSON,
		input.ClaimID,
	)
	if err != nil {
		return fmt.Errorf("update analysis: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateStatus applies a human analyst decision to a claim.
func (r *claimRepo) UpdateStatus(ctx context.Context, input domain.ReviewInput) error {
	const q = `
		UPDATE claims
		SET
			status        = $1,
			analyst_id    = $2,
			analyst_notes = $3,
			updated_at    = NOW()
		WHERE id = $4 AND company_id = (SELECT company_id FROM claims WHERE id = $4)`

	tag, err := r.db.Pool.Exec(ctx, q,
		string(input.Status),
		nullString(input.AnalystID),
		nullString(input.AnalystNotes),
		input.ClaimID,
	)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateStatusByID sets the status field without analyst context (internal use).
func (r *claimRepo) UpdateStatusByID(ctx context.Context, claimID string, status domain.ClaimStatus) error {
	const q = `UPDATE claims SET status = $1, updated_at = NOW() WHERE id = $2`
	tag, err := r.db.Pool.Exec(ctx, q, string(status), claimID)
	if err != nil {
		return fmt.Errorf("update status by id: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete soft-deletes a claim (hard delete for simplicity; add deleted_at if needed).
func (r *claimRepo) Delete(ctx context.Context, id, companyID string) error {
	const q = `DELETE FROM claims WHERE id = $1 AND company_id = $2`
	tag, err := r.db.Pool.Exec(ctx, q, id, companyID)
	if err != nil {
		return fmt.Errorf("delete claim: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetDailyStats returns aggregate fraud statistics per day for the dashboard.
func (r *claimRepo) GetDailyStats(ctx context.Context, companyID string, days int) ([]*domain.DailyStat, error) {
	if days <= 0 || days > 90 {
		days = 30
	}
	const q = `
		SELECT
			DATE_TRUNC('day', created_at) AS date,
			COUNT(*) AS total_claims,
			COUNT(*) FILTER (WHERE status = 'FLAGGED') AS flagged_claims,
			COUNT(*) FILTER (WHERE status = 'APPROVED') AS approved_claims,
			COUNT(*) FILTER (WHERE status = 'REJECTED') AS rejected_claims,
			COALESCE(AVG(fraud_score), 0) AS avg_fraud_score,
			COALESCE(SUM(amount), 0) AS total_amount
		FROM claims
		WHERE company_id = $1
		  AND created_at >= NOW() - ($2 || ' days')::interval
		GROUP BY 1
		ORDER BY 1 DESC`

	rows, err := r.db.Pool.Query(ctx, q, companyID, days)
	if err != nil {
		return nil, fmt.Errorf("daily stats query: %w", err)
	}
	defer rows.Close()

	var stats []*domain.DailyStat
	for rows.Next() {
		s := &domain.DailyStat{}
		if err := rows.Scan(
			&s.Date,
			&s.TotalClaims,
			&s.FlaggedClaims,
			&s.ApprovedClaims,
			&s.RejectedClaims,
			&s.AvgFraudScore,
			&s.TotalAmount,
		); err != nil {
			return nil, fmt.Errorf("scan daily stat: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// ─── helpers ────────────────────────────────────────────────────────────────

func buildWhereClause(f domain.ListFilter) (string, []any) {
	var conditions []string
	var args []any
	idx := 1

	if f.CompanyID != "" {
		conditions = append(conditions, fmt.Sprintf("company_id = $%d", idx))
		args = append(args, f.CompanyID)
		idx++
	}
	if f.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", idx))
		args = append(args, f.UserID)
		idx++
	}
	if f.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, string(f.Status))
		idx++
	}
	if f.ClaimType != "" {
		conditions = append(conditions, fmt.Sprintf("claim_type = $%d", idx))
		args = append(args, string(f.ClaimType))
		idx++
	}
	if f.MinAmount > 0 {
		conditions = append(conditions, fmt.Sprintf("amount >= $%d", idx))
		args = append(args, f.MinAmount)
		idx++
	}
	if f.MaxAmount > 0 {
		conditions = append(conditions, fmt.Sprintf("amount <= $%d", idx))
		args = append(args, f.MaxAmount)
		idx++
	}
	if f.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, *f.DateFrom)
		idx++
	}
	if f.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", idx))
		args = append(args, *f.DateTo)
		idx++
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func scanClaim(row pgx.Row) (*domain.Claim, error) {
	c := &domain.Claim{}
	var claimType, status string
	var riskJSON []byte
	var analystID, analystNotes *string

	err := row.Scan(
		&c.ID,
		&c.UserID,
		&c.CompanyID,
		&c.PolicyNumber,
		&claimType,
		&c.Amount,
		&c.IncidentDate,
		&c.DocURL,
		&c.Description,
		&c.FraudScore,
		&c.FraudReason,
		&riskJSON,
		&status,
		&analystID,
		&analystNotes,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan claim: %w", err)
	}

	c.ClaimType = domain.ClaimType(claimType)
	c.Status = domain.ClaimStatus(status)
	if analystID != nil {
		c.AnalystID = *analystID
	}
	if analystNotes != nil {
		c.AnalystNotes = *analystNotes
	}
	if len(riskJSON) > 0 && string(riskJSON) != "null" {
		_ = json.Unmarshal(riskJSON, &c.RiskFactors)
	}
	return c, nil
}

func scanClaimRow(rows pgx.Rows) (*domain.Claim, error) {
	c := &domain.Claim{}
	var claimType, status string
	var riskJSON []byte
	var analystID, analystNotes *string

	err := rows.Scan(
		&c.ID,
		&c.UserID,
		&c.CompanyID,
		&c.PolicyNumber,
		&claimType,
		&c.Amount,
		&c.IncidentDate,
		&c.DocURL,
		&c.Description,
		&c.FraudScore,
		&c.FraudReason,
		&riskJSON,
		&status,
		&analystID,
		&analystNotes,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan claim row: %w", err)
	}

	c.ClaimType = domain.ClaimType(claimType)
	c.Status = domain.ClaimStatus(status)
	if analystID != nil {
		c.AnalystID = *analystID
	}
	if analystNotes != nil {
		c.AnalystNotes = *analystNotes
	}
	if len(riskJSON) > 0 && string(riskJSON) != "null" {
		_ = json.Unmarshal(riskJSON, &c.RiskFactors)
	}
	return c, nil
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Compile-time check: pgxpool is imported.
var _ *pgxpool.Pool
