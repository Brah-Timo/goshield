// Package service contains the business logic layer for the claim-service.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"io"

	"github.com/goshield/pkg/config"
	"github.com/goshield/pkg/events"
	"github.com/goshield/pkg/storage"
	"github.com/goshield/services/claim-service/internal/domain"
	"github.com/goshield/services/claim-service/internal/repository"
)

// ClaimService defines the business operations for claims.
type ClaimService interface {
	// CreateClaim initialises a new claim record and publishes a Kafka event.
	CreateClaim(ctx context.Context, input domain.CreateClaimInput) (*domain.Claim, error)

	// UploadDocument stores the claim document and updates the record's doc_url.
	UploadDocument(ctx context.Context, claimID, companyID, filename string, reader io.Reader, size int64) (string, error)

	// GetClaim retrieves a single claim scoped to the caller's company.
	GetClaim(ctx context.Context, id, companyID string) (*domain.Claim, error)

	// ListClaims returns a paginated, filtered list of claims.
	ListClaims(ctx context.Context, filter domain.ListFilter) (*domain.ListResult, error)

	// ReviewClaim applies an analyst decision (approve/reject/more-info).
	ReviewClaim(ctx context.Context, input domain.ReviewInput) error

	// HandleAnalysisResult processes the result delivered by the AI service via Kafka.
	HandleAnalysisResult(ctx context.Context, payload events.ClaimAnalyzedPayload) error

	// DeleteClaim removes a claim permanently (admin only).
	DeleteClaim(ctx context.Context, id, companyID string) error

	// GetDailyStats returns dashboard analytics.
	GetDailyStats(ctx context.Context, companyID string, days int) ([]*domain.DailyStat, error)
}

type claimService struct {
	repo     repository.ClaimRepository
	storage  storage.Client
	producer *events.Producer
	cfg      *config.AppConfig
	logger   *zap.Logger
}

// New creates a fully wired ClaimService.
func New(
	repo repository.ClaimRepository,
	stg storage.Client,
	producer *events.Producer,
	cfg *config.AppConfig,
	logger *zap.Logger,
) ClaimService {
	return &claimService{
		repo:     repo,
		storage:  stg,
		producer: producer,
		cfg:      cfg,
		logger:   logger,
	}
}

// CreateClaim creates a new PENDING claim and publishes claim.created to Kafka.
func (s *claimService) CreateClaim(ctx context.Context, input domain.CreateClaimInput) (*domain.Claim, error) {
	claim := &domain.Claim{
		ID:           uuid.New().String(),
		UserID:       input.UserID,
		CompanyID:    input.CompanyID,
		PolicyNumber: input.PolicyNumber,
		ClaimType:    input.ClaimType,
		Amount:       input.Amount,
		IncidentDate: input.IncidentDate,
		Description:  input.Description,
		Status:       domain.StatusPending,
		FraudScore:   0.0,
		RiskFactors:  []string{},
	}

	if err := s.repo.Create(ctx, claim); err != nil {
		return nil, fmt.Errorf("create claim record: %w", err)
	}

	// Determine account age (simplified from users table; will be enriched later).
	accountAgeDays := int32(0)
	incidentDateStr := ""
	if claim.IncidentDate != nil {
		incidentDateStr = claim.IncidentDate.Format("2006-01-02")
	}

	payload := events.ClaimCreatedPayload{
		ClaimID:        claim.ID,
		DocURL:         claim.DocURL,
		Amount:         claim.Amount,
		ClaimType:      string(claim.ClaimType),
		Description:    claim.Description,
		PolicyNumber:   claim.PolicyNumber,
		UserID:         claim.UserID,
		CompanyID:      claim.CompanyID,
		AccountAgeDays: accountAgeDays,
		PriorClaims:    0,
		IncidentDate:   incidentDateStr,
	}

	if err := s.producer.PublishPayload(
		ctx,
		s.cfg.Kafka.TopicClaimsNew,
		events.EventClaimCreated,
		claim.ID,
		claim.CompanyID,
		payload,
	); err != nil {
		// Log but don't fail — the record is already in DB, a retry job can re-enqueue.
		s.logger.Error("failed to publish claim.created event",
			zap.String("claim_id", claim.ID),
			zap.Error(err),
		)
	}

	s.logger.Info("claim created",
		zap.String("claim_id", claim.ID),
		zap.String("company_id", claim.CompanyID),
		zap.Float64("amount", claim.Amount),
	)

	return claim, nil
}

// UploadDocument uploads the document to object storage and updates the claim record.
func (s *claimService) UploadDocument(
	ctx context.Context,
	claimID, companyID, filename string,
	reader io.Reader,
	size int64,
) (string, error) {
	if s.storage == nil {
		return "", fmt.Errorf("storage not configured")
	}
	ext := strings.ToLower(filepath.Ext(filename))
	allowedExts := map[string]bool{".pdf": true, ".jpg": true, ".jpeg": true, ".png": true}
	if !allowedExts[ext] {
		return "", fmt.Errorf("unsupported file type: %s (allowed: pdf, jpg, jpeg, png)", ext)
	}

	// Build storage key: {company_id}/{claim_id}/{timestamp}{ext}
	key := fmt.Sprintf("%s/%s/%d%s", companyID, claimID, time.Now().UnixMilli(), ext)

	contentType := "application/pdf"
	if ext == ".jpg" || ext == ".jpeg" {
		contentType = "image/jpeg"
	} else if ext == ".png" {
		contentType = "image/png"
	}

	docURL, err := s.storage.Upload(ctx, key, reader, size, contentType)
	if err != nil {
		return "", fmt.Errorf("upload document: %w", err)
	}

	if err := s.repo.UpdateDocURL(ctx, claimID, companyID, docURL); err != nil {
		return "", fmt.Errorf("update doc_url: %w", err)
	}

	// Re-publish the claim.created event with the doc_url now set.
	claim, err := s.repo.GetByID(ctx, claimID, companyID)
	if err == nil {
		incidentDateStr := ""
		if claim.IncidentDate != nil {
			incidentDateStr = claim.IncidentDate.Format("2006-01-02")
		}
		payload := events.ClaimCreatedPayload{
			ClaimID:      claim.ID,
			DocURL:       docURL,
			Amount:       claim.Amount,
			ClaimType:    string(claim.ClaimType),
			Description:  claim.Description,
			PolicyNumber: claim.PolicyNumber,
			UserID:       claim.UserID,
			CompanyID:    claim.CompanyID,
			IncidentDate: incidentDateStr,
		}
		if pubErr := s.producer.PublishPayload(
			ctx,
			s.cfg.Kafka.TopicClaimsNew,
			events.EventClaimCreated,
			claim.ID,
			claim.CompanyID,
			payload,
		); pubErr != nil {
			s.logger.Warn("failed to re-publish claim.created after doc upload",
				zap.String("claim_id", claimID),
				zap.Error(pubErr),
			)
		}
	}

	s.logger.Info("document uploaded",
		zap.String("claim_id", claimID),
		zap.String("doc_url", docURL),
	)
	return docURL, nil
}

// GetClaim retrieves a claim with tenant isolation.
func (s *claimService) GetClaim(ctx context.Context, id, companyID string) (*domain.Claim, error) {
	claim, err := s.repo.GetByID(ctx, id, companyID)
	if err != nil {
		return nil, err
	}
	return claim, nil
}

// ListClaims returns paginated claims for the caller's company.
func (s *claimService) ListClaims(ctx context.Context, filter domain.ListFilter) (*domain.ListResult, error) {
	return s.repo.List(ctx, filter)
}

// ReviewClaim processes an analyst decision and publishes the appropriate event.
func (s *claimService) ReviewClaim(ctx context.Context, input domain.ReviewInput) error {
	allowed := map[domain.ClaimStatus]bool{
		domain.StatusApproved: true,
		domain.StatusRejected: true,
		domain.StatusMoreInfo: true,
	}
	if !allowed[input.Status] {
		return fmt.Errorf("invalid review status: %s", input.Status)
	}

	if err := s.repo.UpdateStatus(ctx, input); err != nil {
		return fmt.Errorf("update claim status: %w", err)
	}

	// Publish domain event.
	eventType := events.EventClaimApproved
	if input.Status == domain.StatusRejected {
		eventType = events.EventClaimRejected
	}

	type reviewPayload struct {
		ClaimID      string `json:"claim_id"`
		Status       string `json:"status"`
		AnalystID    string `json:"analyst_id"`
		AnalystNotes string `json:"analyst_notes"`
	}

	payloadData, _ := json.Marshal(reviewPayload{
		ClaimID:      input.ClaimID,
		Status:       string(input.Status),
		AnalystID:    input.AnalystID,
		AnalystNotes: input.AnalystNotes,
	})

	_ = s.producer.Publish(ctx, s.cfg.Kafka.TopicClaimsNew, events.Event{
		ID:         uuid.New().String(),
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		Payload:    payloadData,
	})

	s.logger.Info("claim reviewed",
		zap.String("claim_id", input.ClaimID),
		zap.String("status", string(input.Status)),
		zap.String("analyst_id", input.AnalystID),
	)
	return nil
}

// HandleAnalysisResult processes AI results received via Kafka consumer.
func (s *claimService) HandleAnalysisResult(ctx context.Context, payload events.ClaimAnalyzedPayload) error {
	if err := s.repo.UpdateAnalysis(ctx, domain.UpdateAnalysisInput{
		ClaimID:     payload.ClaimID,
		FraudScore:  payload.FraudScore,
		FraudReason: payload.Reason,
		RiskFactors: payload.RiskFactors,
	}); err != nil {
		return fmt.Errorf("update analysis result: %w", err)
	}

	// If fraud score >= threshold, also publish flagged event.
	if payload.FraudScore >= s.cfg.Notification.FraudThresholdFlag {
		flaggedPayload := events.ClaimFlaggedPayload{
			ClaimID:    payload.ClaimID,
			FraudScore: payload.FraudScore,
			CompanyID:  "",
			Reason:     payload.Reason,
		}
		if pubErr := s.producer.PublishPayload(
			ctx,
			s.cfg.Kafka.TopicClaimsFlagged,
			events.EventClaimFlagged,
			uuid.New().String(),
			"",
			flaggedPayload,
		); pubErr != nil {
			s.logger.Warn("failed to publish claim.flagged",
				zap.String("claim_id", payload.ClaimID),
				zap.Error(pubErr),
			)
		}
	}

	s.logger.Info("analysis result applied",
		zap.String("claim_id", payload.ClaimID),
		zap.Float64("fraud_score", payload.FraudScore),
	)
	return nil
}

// DeleteClaim permanently removes a claim (caller must verify ADMIN role).
func (s *claimService) DeleteClaim(ctx context.Context, id, companyID string) error {
	return s.repo.Delete(ctx, id, companyID)
}

// GetDailyStats returns aggregated fraud stats for the dashboard.
func (s *claimService) GetDailyStats(ctx context.Context, companyID string, days int) ([]*domain.DailyStat, error) {
	return s.repo.GetDailyStats(ctx, companyID, days)
}
