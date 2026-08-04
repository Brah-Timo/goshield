package domain

import "time"

// Lead statuses
const (
	LeadStatusNew        = "new"
	LeadStatusContacted  = "contacted"
	LeadStatusQualified  = "qualified"
	LeadStatusLost       = "lost"
)

// Opportunity stages
const (
	StageNew        = "new"
	StageQualified  = "qualified"
	StageProposal   = "proposal"
	StageNegotiation = "negotiation"
	StageWon        = "won"
	StageLost       = "lost"
)

// Activity types
const (
	ActivityCall    = "call"
	ActivityEmail   = "email"
	ActivityMeeting = "meeting"
	ActivityTask    = "task"
)

// Lead represents an unqualified potential customer
type Lead struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Company     string    `json:"company"`
	Source      string    `json:"source"` // website, referral, cold_call, etc.
	Status      string    `json:"status"`
	AssignedTo  string    `json:"assigned_to"`  // user_id
	Notes       string    `json:"notes"`
	Tags        []string  `json:"tags"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Opportunity represents a qualified sales opportunity
type Opportunity struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	Name            string    `json:"name"`
	LeadID          string    `json:"lead_id,omitempty"`
	CustomerName    string    `json:"customer_name"`
	CustomerEmail   string    `json:"customer_email"`
	CustomerPhone   string    `json:"customer_phone"`
	Company         string    `json:"company"`
	Stage           string    `json:"stage"`
	ExpectedRevenue float64   `json:"expected_revenue"`
	Probability     float64   `json:"probability"` // 0-100
	ExpectedClose   time.Time `json:"expected_close"`
	AssignedTo      string    `json:"assigned_to"` // user_id
	Description     string    `json:"description"`
	LostReason      string    `json:"lost_reason,omitempty"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Activity represents interactions with leads/opportunities
type Activity struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	Type          string     `json:"type"` // call, email, meeting, task
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	LeadID        string     `json:"lead_id,omitempty"`
	OpportunityID string     `json:"opportunity_id,omitempty"`
	AssignedTo    string     `json:"assigned_to"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	IsDone        bool       `json:"is_done"`
	CreatedBy     string     `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// PipelineStats for dashboard
type PipelineStats struct {
	TotalLeads         int     `json:"total_leads"`
	TotalOpportunities int     `json:"total_opportunities"`
	TotalRevenue       float64 `json:"total_revenue"`
	WonOpportunities   int     `json:"won_opportunities"`
	LostOpportunities  int     `json:"lost_opportunities"`
	WinRate            float64 `json:"win_rate"`
	StageBreakdown     []StageCount `json:"stage_breakdown"`
}

type StageCount struct {
	Stage   string  `json:"stage"`
	Count   int     `json:"count"`
	Revenue float64 `json:"revenue"`
}

// Filters
type LeadFilter struct {
	TenantID   string
	Status     string
	AssignedTo string
	Page       int
	Limit      int
}

type OpportunityFilter struct {
	TenantID   string
	Stage      string
	AssignedTo string
	Page       int
	Limit      int
}

type ActivityFilter struct {
	TenantID      string
	LeadID        string
	OpportunityID string
	AssignedTo    string
	IsDone        *bool
	Page          int
	Limit         int
}

// Request/Response types
type CreateLeadRequest struct {
	Name       string   `json:"name"`
	Email      string   `json:"email"`
	Phone      string   `json:"phone"`
	Company    string   `json:"company"`
	Source     string   `json:"source"`
	AssignedTo string   `json:"assigned_to"`
	Notes      string   `json:"notes"`
	Tags       []string `json:"tags"`
}

type UpdateLeadRequest struct {
	Name       string   `json:"name"`
	Email      string   `json:"email"`
	Phone      string   `json:"phone"`
	Company    string   `json:"company"`
	Source     string   `json:"source"`
	Status     string   `json:"status"`
	AssignedTo string   `json:"assigned_to"`
	Notes      string   `json:"notes"`
	Tags       []string `json:"tags"`
}

type CreateOpportunityRequest struct {
	Name            string    `json:"name"`
	LeadID          string    `json:"lead_id"`
	CustomerName    string    `json:"customer_name"`
	CustomerEmail   string    `json:"customer_email"`
	CustomerPhone   string    `json:"customer_phone"`
	Company         string    `json:"company"`
	Stage           string    `json:"stage"`
	ExpectedRevenue float64   `json:"expected_revenue"`
	Probability     float64   `json:"probability"`
	ExpectedClose   time.Time `json:"expected_close"`
	AssignedTo      string    `json:"assigned_to"`
	Description     string    `json:"description"`
}

type UpdateOpportunityRequest struct {
	Name            string    `json:"name"`
	CustomerName    string    `json:"customer_name"`
	CustomerEmail   string    `json:"customer_email"`
	CustomerPhone   string    `json:"customer_phone"`
	Company         string    `json:"company"`
	Stage           string    `json:"stage"`
	ExpectedRevenue float64   `json:"expected_revenue"`
	Probability     float64   `json:"probability"`
	ExpectedClose   time.Time `json:"expected_close"`
	AssignedTo      string    `json:"assigned_to"`
	Description     string    `json:"description"`
	LostReason      string    `json:"lost_reason"`
}

type CreateActivityRequest struct {
	Type          string     `json:"type"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	LeadID        string     `json:"lead_id"`
	OpportunityID string     `json:"opportunity_id"`
	AssignedTo    string     `json:"assigned_to"`
	DueDate       *time.Time `json:"due_date"`
}
