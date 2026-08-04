package repository

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/goerp/goerp/internal/crm/domain"
	"github.com/goerp/goerp/internal/shared/database"
)

// CRMRepository uses *database.DB (wraps *sql.DB) so it shares the SQLite
// connection managed by the application bootstrap.
type CRMRepository struct {
	db *database.DB
}

func NewCRMRepository(db *database.DB) *CRMRepository {
	return &CRMRepository{db: db}
}

// ---- helpers ----------------------------------------------------------------

func crmParseTime(s string) time.Time {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05+00:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Now()
}

func crmParseTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t := crmParseTime(s)
	return &t
}

// ─── LEADS ───────────────────────────────────────────────────────────────────

func (r *CRMRepository) ListLeads(filter domain.LeadFilter) ([]domain.Lead, int, error) {
	offset := 0
	if filter.Page > 1 {
		offset = (filter.Page - 1) * filter.Limit
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}

	query := `SELECT id, tenant_id, name, COALESCE(email,''), COALESCE(phone,''),
		COALESCE(company,''), COALESCE(source,''), status,
		COALESCE(assigned_to,''), COALESCE(notes,''),
		created_by, created_at, updated_at
		FROM crm_leads WHERE tenant_id=?`
	args := []interface{}{filter.TenantID}
	pIdx := 2

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status=?")
		args = append(args, filter.Status)
		pIdx++
	}
	if filter.AssignedTo != "" {
		query += fmt.Sprintf(" AND assigned_to=?")
		args = append(args, filter.AssignedTo)
		pIdx++
	}
	_ = pIdx

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT ? OFFSET ?")
	args = append(args, filter.Limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var leads []domain.Lead
	for rows.Next() {
		var l domain.Lead
		var createdStr, updatedStr string
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.Name, &l.Email, &l.Phone,
			&l.Company, &l.Source, &l.Status, &l.AssignedTo, &l.Notes,
			&l.CreatedBy, &createdStr, &updatedStr,
		); err != nil {
			continue
		}
		l.CreatedAt = crmParseTime(createdStr)
		l.UpdatedAt = crmParseTime(updatedStr)
		leads = append(leads, l)
	}

	var total int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM crm_leads WHERE tenant_id=?`, filter.TenantID).Scan(&total)
	return leads, total, nil
}

func (r *CRMRepository) GetLead(id, tenantID string) (*domain.Lead, error) {
	var l domain.Lead
	var createdStr, updatedStr string
	err := r.db.QueryRow(`
		SELECT id, tenant_id, name, COALESCE(email,''), COALESCE(phone,''),
		       COALESCE(company,''), COALESCE(source,''), status,
		       COALESCE(assigned_to,''), COALESCE(notes,''),
		       created_by, created_at, updated_at
		FROM crm_leads WHERE id=? AND tenant_id=?`, id, tenantID).Scan(
		&l.ID, &l.TenantID, &l.Name, &l.Email, &l.Phone,
		&l.Company, &l.Source, &l.Status, &l.AssignedTo, &l.Notes,
		&l.CreatedBy, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}
	l.CreatedAt = crmParseTime(createdStr)
	l.UpdatedAt = crmParseTime(updatedStr)
	return &l, nil
}

func (r *CRMRepository) CreateLead(l *domain.Lead) error {
	l.ID = uuid.New().String()
	l.CreatedAt = time.Now()
	l.UpdatedAt = time.Now()
	if l.Status == "" {
		l.Status = domain.LeadStatusNew
	}

	_, err := r.db.Exec(`
		INSERT INTO crm_leads
		  (id, tenant_id, name, email, phone, company, source, status,
		   assigned_to, notes, created_by, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		l.ID, l.TenantID, l.Name, l.Email, l.Phone,
		l.Company, l.Source, l.Status,
		l.AssignedTo, l.Notes, l.CreatedBy,
		l.CreatedAt.Format(time.RFC3339),
		l.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *CRMRepository) UpdateLead(l *domain.Lead) error {
	l.UpdatedAt = time.Now()
	_, err := r.db.Exec(`
		UPDATE crm_leads SET name=?, email=?, phone=?, company=?,
		       source=?, status=?, assigned_to=?, notes=?, updated_at=?
		WHERE id=? AND tenant_id=?`,
		l.Name, l.Email, l.Phone, l.Company,
		l.Source, l.Status, l.AssignedTo, l.Notes,
		l.UpdatedAt.Format(time.RFC3339),
		l.ID, l.TenantID,
	)
	return err
}

func (r *CRMRepository) DeleteLead(id, tenantID string) error {
	_, err := r.db.Exec(`DELETE FROM crm_leads WHERE id=? AND tenant_id=?`, id, tenantID)
	return err
}

// ─── OPPORTUNITIES ────────────────────────────────────────────────────────────

func (r *CRMRepository) ListOpportunities(filter domain.OpportunityFilter) ([]domain.Opportunity, int, error) {
	offset := 0
	if filter.Page > 1 {
		offset = (filter.Page - 1) * filter.Limit
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}

	query := `SELECT id, tenant_id, name, COALESCE(lead_id,''),
		customer_name, COALESCE(customer_email,''), COALESCE(customer_phone,''),
		COALESCE(company,''), stage, expected_revenue,
		probability, COALESCE(expected_close,''),
		COALESCE(assigned_to,''), COALESCE(description,''),
		COALESCE(lost_reason,''), created_by, created_at, updated_at
		FROM crm_opportunities WHERE tenant_id=?`
	args := []interface{}{filter.TenantID}

	if filter.Stage != "" {
		query += " AND stage=?"
		args = append(args, filter.Stage)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var opps []domain.Opportunity
	for rows.Next() {
		var o domain.Opportunity
		var createdStr, updatedStr, expectedClose string
		if err := rows.Scan(
			&o.ID, &o.TenantID, &o.Name, &o.LeadID,
			&o.CustomerName, &o.CustomerEmail, &o.CustomerPhone, &o.Company,
			&o.Stage, &o.ExpectedRevenue, &o.Probability, &expectedClose,
			&o.AssignedTo, &o.Description, &o.LostReason,
			&o.CreatedBy, &createdStr, &updatedStr,
		); err != nil {
			continue
		}
		o.CreatedAt = crmParseTime(createdStr)
		o.UpdatedAt = crmParseTime(updatedStr)
		if expectedClose != "" {
			t := crmParseTime(expectedClose)
			o.ExpectedClose = t
		}
		opps = append(opps, o)
	}

	var total int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM crm_opportunities WHERE tenant_id=?`, filter.TenantID).Scan(&total)
	return opps, total, nil
}

func (r *CRMRepository) GetOpportunity(id, tenantID string) (*domain.Opportunity, error) {
	var o domain.Opportunity
	var createdStr, updatedStr, expectedClose string
	err := r.db.QueryRow(`
		SELECT id, tenant_id, name, COALESCE(lead_id,''),
		       customer_name, COALESCE(customer_email,''), COALESCE(customer_phone,''),
		       COALESCE(company,''), stage, expected_revenue,
		       probability, COALESCE(expected_close,''),
		       COALESCE(assigned_to,''), COALESCE(description,''),
		       COALESCE(lost_reason,''), created_by, created_at, updated_at
		FROM crm_opportunities WHERE id=? AND tenant_id=?`, id, tenantID).Scan(
		&o.ID, &o.TenantID, &o.Name, &o.LeadID,
		&o.CustomerName, &o.CustomerEmail, &o.CustomerPhone, &o.Company,
		&o.Stage, &o.ExpectedRevenue, &o.Probability, &expectedClose,
		&o.AssignedTo, &o.Description, &o.LostReason,
		&o.CreatedBy, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}
	o.CreatedAt = crmParseTime(createdStr)
	o.UpdatedAt = crmParseTime(updatedStr)
	if expectedClose != "" {
		o.ExpectedClose = crmParseTime(expectedClose)
	}
	return &o, nil
}

func (r *CRMRepository) CreateOpportunity(o *domain.Opportunity) error {
	o.ID = uuid.New().String()
	o.CreatedAt = time.Now()
	o.UpdatedAt = time.Now()
	if o.Stage == "" {
		o.Stage = domain.StageNew
	}

	var expectedClose interface{}
	if !o.ExpectedClose.IsZero() {
		expectedClose = o.ExpectedClose.Format("2006-01-02")
	}

	_, err := r.db.Exec(`
		INSERT INTO crm_opportunities
		  (id, tenant_id, name, lead_id, customer_name, customer_email, customer_phone,
		   company, stage, expected_revenue, probability, expected_close,
		   assigned_to, description, lost_reason, created_by, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		o.ID, o.TenantID, o.Name,
		crmNullStr(o.LeadID), o.CustomerName, o.CustomerEmail, o.CustomerPhone,
		o.Company, o.Stage, o.ExpectedRevenue, o.Probability, expectedClose,
		o.AssignedTo, o.Description, o.LostReason, o.CreatedBy,
		o.CreatedAt.Format(time.RFC3339),
		o.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *CRMRepository) UpdateOpportunity(o *domain.Opportunity) error {
	o.UpdatedAt = time.Now()
	var expectedClose interface{}
	if !o.ExpectedClose.IsZero() {
		expectedClose = o.ExpectedClose.Format("2006-01-02")
	}
	_, err := r.db.Exec(`
		UPDATE crm_opportunities SET name=?, customer_name=?,
		       customer_email=?, customer_phone=?, company=?, stage=?,
		       expected_revenue=?, probability=?, expected_close=?,
		       assigned_to=?, description=?, lost_reason=?, updated_at=?
		WHERE id=? AND tenant_id=?`,
		o.Name, o.CustomerName, o.CustomerEmail, o.CustomerPhone,
		o.Company, o.Stage, o.ExpectedRevenue, o.Probability, expectedClose,
		o.AssignedTo, o.Description, o.LostReason,
		o.UpdatedAt.Format(time.RFC3339),
		o.ID, o.TenantID,
	)
	return err
}

// ─── ACTIVITIES ───────────────────────────────────────────────────────────────

func (r *CRMRepository) ListActivities(filter domain.ActivityFilter) ([]domain.Activity, int, error) {
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	query := `SELECT id, tenant_id, type, title, COALESCE(description,''),
		COALESCE(lead_id,''), COALESCE(opportunity_id,''),
		COALESCE(assigned_to,''), COALESCE(due_date,''),
		COALESCE(completed_at,''), is_done, created_by, created_at, updated_at
		FROM crm_activities WHERE tenant_id=?`
	args := []interface{}{filter.TenantID}

	if filter.LeadID != "" {
		query += " AND lead_id=?"
		args = append(args, filter.LeadID)
	}
	if filter.OpportunityID != "" {
		query += " AND opportunity_id=?"
		args = append(args, filter.OpportunityID)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, 0)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var acts []domain.Activity
	for rows.Next() {
		var a domain.Activity
		var createdStr, updatedStr, dueDateStr, completedAtStr string
		var isDone int
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.Type, &a.Title, &a.Description,
			&a.LeadID, &a.OpportunityID, &a.AssignedTo,
			&dueDateStr, &completedAtStr, &isDone,
			&a.CreatedBy, &createdStr, &updatedStr,
		); err != nil {
			continue
		}
		a.IsDone = isDone == 1
		a.CreatedAt = crmParseTime(createdStr)
		a.UpdatedAt = crmParseTime(updatedStr)
		a.DueDate = crmParseTimePtr(dueDateStr)
		a.CompletedAt = crmParseTimePtr(completedAtStr)
		acts = append(acts, a)
	}

	var total int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM crm_activities WHERE tenant_id=?`, filter.TenantID).Scan(&total)
	return acts, total, nil
}

func (r *CRMRepository) CreateActivity(a *domain.Activity) error {
	a.ID = uuid.New().String()
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()

	var dueDate interface{}
	if a.DueDate != nil {
		dueDate = a.DueDate.Format("2006-01-02")
	}

	_, err := r.db.Exec(`
		INSERT INTO crm_activities
		  (id, tenant_id, type, title, description, lead_id, opportunity_id,
		   assigned_to, due_date, is_done, created_by, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,0,?,?,?)`,
		a.ID, a.TenantID, a.Type, a.Title, a.Description,
		crmNullStr(a.LeadID), crmNullStr(a.OpportunityID),
		a.AssignedTo, dueDate, a.CreatedBy,
		a.CreatedAt.Format(time.RFC3339),
		a.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *CRMRepository) CompleteActivity(id, tenantID string) error {
	now := time.Now()
	_, err := r.db.Exec(`
		UPDATE crm_activities SET is_done=1, completed_at=?, updated_at=?
		WHERE id=? AND tenant_id=?`,
		now.Format(time.RFC3339), now.Format(time.RFC3339), id, tenantID,
	)
	return err
}

// ─── PIPELINE STATS ──────────────────────────────────────────────────────────

func (r *CRMRepository) GetPipelineStats(tenantID string) (*domain.PipelineStats, error) {
	stats := &domain.PipelineStats{}

	_ = r.db.QueryRow(`SELECT COUNT(*) FROM crm_leads WHERE tenant_id=?`, tenantID).Scan(&stats.TotalLeads)
	_ = r.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(expected_revenue),0) FROM crm_opportunities WHERE tenant_id=?`, tenantID).Scan(&stats.TotalOpportunities, &stats.TotalRevenue)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM crm_opportunities WHERE tenant_id=? AND stage='won'`, tenantID).Scan(&stats.WonOpportunities)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM crm_opportunities WHERE tenant_id=? AND stage='lost'`, tenantID).Scan(&stats.LostOpportunities)

	if stats.TotalOpportunities > 0 {
		stats.WinRate = float64(stats.WonOpportunities) / float64(stats.TotalOpportunities) * 100
	}

	// Stage breakdown
	rows, err := r.db.Query(`
		SELECT stage, COUNT(*), COALESCE(SUM(expected_revenue),0)
		FROM crm_opportunities WHERE tenant_id=?
		GROUP BY stage
		ORDER BY CASE stage
			WHEN 'new' THEN 1 WHEN 'qualified' THEN 2
			WHEN 'proposal' THEN 3 WHEN 'negotiation' THEN 4
			WHEN 'won' THEN 5 WHEN 'lost' THEN 6 ELSE 7 END`,
		tenantID,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sc domain.StageCount
			rows.Scan(&sc.Stage, &sc.Count, &sc.Revenue)
			stats.StageBreakdown = append(stats.StageBreakdown, sc)
		}
	}

	return stats, nil
}

// ---- nullStr helper ---------------------------------------------------------

func crmNullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
