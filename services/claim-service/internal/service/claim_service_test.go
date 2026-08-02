package service_test

import (
	"context"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/goshield/pkg/config"
	"github.com/goshield/pkg/events"
	"github.com/goshield/services/claim-service/internal/domain"
	"github.com/goshield/services/claim-service/internal/repository"
	mockrepo "github.com/goshield/services/claim-service/internal/repository/mock"
	"github.com/goshield/services/claim-service/internal/service"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func buildTestConfig() *config.AppConfig {
	return &config.AppConfig{
		Kafka: config.KafkaConfig{
			TopicClaimsNew:      "claims.new",
			TopicClaimsAnalyzed: "claims.analyzed",
			TopicClaimsFlagged:  "claims.flagged",
			TopicClaimsFailed:   "claims.failed",
		},
		Notification: config.NotificationConfig{
			FraudThresholdFlag:  0.80,
			FraudThresholdAlert: 0.95,
		},
	}
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestCreateClaim_Success(t *testing.T) {
	log := zaptest.NewLogger(t)
	repo := mockrepo.NewClaimRepository()
	producer := events.NewProducer([]string{}, log)
	defer producer.Close() //nolint:errcheck

	cfg := buildTestConfig()
	svc := service.New(repo, nil, producer, cfg, log)

	input := domain.CreateClaimInput{
		UserID:       "user-1",
		CompanyID:    "company-1",
		PolicyNumber: "POL-001",
		ClaimType:    domain.ClaimTypeHealth,
		Amount:       5000.0,
		Description:  "Medical expenses after accident",
	}

	claim, err := svc.CreateClaim(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if claim.ID == "" {
		t.Error("expected non-empty claim ID")
	}
	if claim.Status != domain.StatusPending {
		t.Errorf("expected PENDING status, got: %s", claim.Status)
	}
	if claim.CompanyID != "company-1" {
		t.Errorf("expected company-1, got: %s", claim.CompanyID)
	}
	if repo.CreateCalled != 1 {
		t.Errorf("expected Create to be called once, got: %d", repo.CreateCalled)
	}
}

func TestCreateClaim_RepoError(t *testing.T) {
	log := zaptest.NewLogger(t)
	repo := mockrepo.NewClaimRepository()
	repo.CreateErr = repository.ErrNotFound
	producer := events.NewProducer([]string{}, log)
	defer producer.Close() //nolint:errcheck

	svc := service.New(repo, nil, producer, buildTestConfig(), log)

	_, err := svc.CreateClaim(context.Background(), domain.CreateClaimInput{
		UserID:       "user-1",
		CompanyID:    "company-1",
		PolicyNumber: "POL-002",
		ClaimType:    domain.ClaimTypeCar,
		Amount:       1000,
	})
	if err == nil {
		t.Fatal("expected error when repo fails, got nil")
	}
}

func TestGetClaim_NotFound(t *testing.T) {
	log := zaptest.NewLogger(t)
	repo := mockrepo.NewClaimRepository()
	producer := events.NewProducer([]string{}, log)
	defer producer.Close() //nolint:errcheck

	svc := service.New(repo, nil, producer, buildTestConfig(), log)

	_, err := svc.GetClaim(context.Background(), "non-existent-id", "company-1")
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestHandleAnalysisResult_FlagsHighScore(t *testing.T) {
	log := zaptest.NewLogger(t)
	repo := mockrepo.NewClaimRepository()
	producer := events.NewProducer([]string{}, log)
	defer producer.Close() //nolint:errcheck

	cfg := buildTestConfig()
	svc := service.New(repo, nil, producer, cfg, log)

	// First create a claim to update
	claim, err := svc.CreateClaim(context.Background(), domain.CreateClaimInput{
		UserID:       "user-1",
		CompanyID:    "company-1",
		PolicyNumber: "POL-003",
		ClaimType:    domain.ClaimTypeProperty,
		Amount:       50000,
	})
	if err != nil {
		t.Fatalf("create claim failed: %v", err)
	}

	// Now apply a high fraud score
	err = svc.HandleAnalysisResult(context.Background(), events.ClaimAnalyzedPayload{
		ClaimID:     claim.ID,
		FraudScore:  0.92,
		Reason:      "Unusually high amount with new account",
		RiskFactors: []string{"high_amount", "new_account"},
	})
	if err != nil {
		t.Fatalf("handle analysis result failed: %v", err)
	}

	// Verify claim was updated
	updated, err := repo.GetByID(context.Background(), claim.ID, "company-1")
	if err != nil {
		t.Fatalf("get claim failed: %v", err)
	}
	if updated.Status != domain.StatusFlagged {
		t.Errorf("expected FLAGGED status, got: %s", updated.Status)
	}
	if updated.FraudScore != 0.92 {
		t.Errorf("expected fraud_score=0.92, got: %f", updated.FraudScore)
	}
}

func TestHandleAnalysisResult_ApprovesLowScore(t *testing.T) {
	log := zaptest.NewLogger(t)
	repo := mockrepo.NewClaimRepository()
	producer := events.NewProducer([]string{}, log)
	defer producer.Close() //nolint:errcheck

	svc := service.New(repo, nil, producer, buildTestConfig(), log)

	claim, _ := svc.CreateClaim(context.Background(), domain.CreateClaimInput{
		UserID:       "user-1",
		CompanyID:    "company-1",
		PolicyNumber: "POL-004",
		ClaimType:    domain.ClaimTypeTravel,
		Amount:       500,
	})

	err := svc.HandleAnalysisResult(context.Background(), events.ClaimAnalyzedPayload{
		ClaimID:     claim.ID,
		FraudScore:  0.12,
		Reason:      "Normal claim pattern",
		RiskFactors: []string{},
	})
	if err != nil {
		t.Fatalf("handle analysis result failed: %v", err)
	}

	updated, _ := repo.GetByID(context.Background(), claim.ID, "company-1")
	if updated.Status != domain.StatusApproved {
		t.Errorf("expected APPROVED status for low fraud score, got: %s", updated.Status)
	}
}

func TestReviewClaim_InvalidStatus(t *testing.T) {
	log := zaptest.NewLogger(t)
	repo := mockrepo.NewClaimRepository()
	producer := events.NewProducer([]string{}, log)
	defer producer.Close() //nolint:errcheck

	svc := service.New(repo, nil, producer, buildTestConfig(), log)

	err := svc.ReviewClaim(context.Background(), domain.ReviewInput{
		ClaimID:   "some-id",
		AnalystID: "analyst-1",
		Status:    domain.StatusPending, // Not a valid review status
	})
	if err == nil {
		t.Fatal("expected error for invalid review status, got nil")
	}
}

func TestListClaims_FilterByStatus(t *testing.T) {
	log := zaptest.NewLogger(t)
	repo := mockrepo.NewClaimRepository()
	producer := events.NewProducer([]string{}, log)
	defer producer.Close() //nolint:errcheck

	svc := service.New(repo, nil, producer, buildTestConfig(), log)

	// Create two claims
	c1, _ := svc.CreateClaim(context.Background(), domain.CreateClaimInput{
		UserID: "u1", CompanyID: "co1", PolicyNumber: "P1",
		ClaimType: domain.ClaimTypeHealth, Amount: 100,
	})
	_, _ = svc.CreateClaim(context.Background(), domain.CreateClaimInput{
		UserID: "u1", CompanyID: "co1", PolicyNumber: "P2",
		ClaimType: domain.ClaimTypeCar, Amount: 200,
	})

	// Flag one manually
	_ = svc.HandleAnalysisResult(context.Background(), events.ClaimAnalyzedPayload{
		ClaimID: c1.ID, FraudScore: 0.9,
	})

	result, err := svc.ListClaims(context.Background(), domain.ListFilter{
		CompanyID: "co1",
		Status:    domain.StatusFlagged,
	})
	if err != nil {
		t.Fatalf("list claims failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 flagged claim, got: %d", result.Total)
	}
}

func TestDeleteClaim_NotFound(t *testing.T) {
	log := zaptest.NewLogger(t)
	repo := mockrepo.NewClaimRepository()
	producer := events.NewProducer([]string{}, log)
	defer producer.Close() //nolint:errcheck

	svc := service.New(repo, nil, producer, buildTestConfig(), log)

	err := svc.DeleteClaim(context.Background(), "ghost-id", "company-1")
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
