// Package domain defines the core business entities and value objects for the claim-service.
package domain

import (
	"time"
)

// ClaimStatus represents the lifecycle state of a claim.
type ClaimStatus string

const (
	StatusPending    ClaimStatus = "PENDING"
	StatusProcessing ClaimStatus = "PROCESSING"
	StatusFlagged    ClaimStatus = "FLAGGED"
	StatusApproved   ClaimStatus = "APPROVED"
	StatusRejected   ClaimStatus = "REJECTED"
	StatusMoreInfo   ClaimStatus = "MORE_INFO"
)

// ClaimType categorizes the kind of insurance claim.
type ClaimType string

const (
	ClaimTypeHealth   ClaimType = "HEALTH"
	ClaimTypeCar      ClaimType = "CAR"
	ClaimTypeProperty ClaimType = "PROPERTY"
	ClaimTypeLife     ClaimType = "LIFE"
	ClaimTypeTravel   ClaimType = "TRAVEL"
	ClaimTypeOther    ClaimType = "OTHER"
)

// Claim is the central aggregate root for insurance claim processing.
type Claim struct {
	ID           string      `json:"id"`
	UserID       string      `json:"user_id"`
	CompanyID    string      `json:"company_id"`
	PolicyNumber string      `json:"policy_number"`
	ClaimType    ClaimType   `json:"claim_type"`
	Amount       float64     `json:"amount"`
	IncidentDate *time.Time  `json:"incident_date,omitempty"`
	DocURL       string      `json:"doc_url,omitempty"`
	Description  string      `json:"description,omitempty"`

	// AI analysis results
	FraudScore  float64  `json:"fraud_score"`
	FraudReason string   `json:"fraud_reason,omitempty"`
	RiskFactors []string `json:"risk_factors,omitempty"`

	// Review
	Status       ClaimStatus `json:"status"`
	AnalystID    string      `json:"analyst_id,omitempty"`
	AnalystNotes string      `json:"analyst_notes,omitempty"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ListFilter contains pagination and filtering parameters for listing claims.
type ListFilter struct {
	CompanyID string
	UserID    string
	Status    ClaimStatus
	ClaimType ClaimType
	MinAmount float64
	MaxAmount float64
	DateFrom  *time.Time
	DateTo    *time.Time
	Page      int
	PageSize  int
	SortBy    string // created_at | fraud_score | amount
	SortOrder string // asc | desc
}

// ListResult wraps a page of claims with total count for pagination.
type ListResult struct {
	Claims     []*Claim `json:"claims"`
	Total      int64    `json:"total"`
	Page       int      `json:"page"`
	PageSize   int      `json:"page_size"`
	TotalPages int      `json:"total_pages"`
}

// CreateClaimInput is the validated input for creating a new claim.
type CreateClaimInput struct {
	UserID       string
	CompanyID    string
	PolicyNumber string
	ClaimType    ClaimType
	Amount       float64
	IncidentDate *time.Time
	Description  string
}

// UpdateAnalysisInput carries AI analysis results from the kafka consumer.
type UpdateAnalysisInput struct {
	ClaimID     string
	FraudScore  float64
	FraudReason string
	RiskFactors []string
}

// ReviewInput carries a human analyst decision.
type ReviewInput struct {
	ClaimID      string
	AnalystID    string
	Status       ClaimStatus
	AnalystNotes string
}

// DailyStat is returned by the stats query for the dashboard overview.
type DailyStat struct {
	Date           time.Time `json:"date"`
	TotalClaims    int64     `json:"total_claims"`
	FlaggedClaims  int64     `json:"flagged_claims"`
	ApprovedClaims int64     `json:"approved_claims"`
	RejectedClaims int64     `json:"rejected_claims"`
	AvgFraudScore  float64   `json:"avg_fraud_score"`
	TotalAmount    float64   `json:"total_amount"`
}
