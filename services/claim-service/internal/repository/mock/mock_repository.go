// Package mock provides a test double for ClaimRepository.
package mock

import (
	"context"
	"sync"

	"github.com/goshield/services/claim-service/internal/domain"
	"github.com/goshield/services/claim-service/internal/repository"
)

// ClaimRepository is a thread-safe in-memory implementation of repository.ClaimRepository.
type ClaimRepository struct {
	mu     sync.RWMutex
	claims map[string]*domain.Claim

	// Call counters for assertions
	CreateCalled         int
	GetByIDCalled        int
	ListCalled           int
	UpdateDocURLCalled   int
	UpdateAnalysisCalled int
	UpdateStatusCalled   int
	DeleteCalled         int
	GetDailyStatsCalled  int

	// Configurable error injection
	CreateErr         error
	GetByIDErr        error
	ListErr           error
	UpdateDocURLErr   error
	UpdateAnalysisErr error
	UpdateStatusErr   error
	DeleteErr         error
}

// NewClaimRepository creates an empty mock repository.
func NewClaimRepository() *ClaimRepository {
	return &ClaimRepository{
		claims: make(map[string]*domain.Claim),
	}
}

func (m *ClaimRepository) Create(ctx context.Context, claim *domain.Claim) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateCalled++
	if m.CreateErr != nil {
		return m.CreateErr
	}
	m.claims[claim.ID] = claim
	return nil
}

func (m *ClaimRepository) GetByID(ctx context.Context, id, companyID string) (*domain.Claim, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetByIDCalled++
	if m.GetByIDErr != nil {
		return nil, m.GetByIDErr
	}
	c, ok := m.claims[id]
	if !ok || c.CompanyID != companyID {
		return nil, repository.ErrNotFound
	}
	return c, nil
}

func (m *ClaimRepository) List(ctx context.Context, filter domain.ListFilter) (*domain.ListResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.ListCalled++
	if m.ListErr != nil {
		return nil, m.ListErr
	}

	var claims []*domain.Claim
	for _, c := range m.claims {
		if c.CompanyID == filter.CompanyID {
			if filter.Status == "" || c.Status == filter.Status {
				claims = append(claims, c)
			}
		}
	}
	return &domain.ListResult{
		Claims:     claims,
		Total:      int64(len(claims)),
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}, nil
}

func (m *ClaimRepository) UpdateDocURL(ctx context.Context, id, companyID, docURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdateDocURLCalled++
	if m.UpdateDocURLErr != nil {
		return m.UpdateDocURLErr
	}
	c, ok := m.claims[id]
	if !ok || c.CompanyID != companyID {
		return repository.ErrNotFound
	}
	c.DocURL = docURL
	return nil
}

func (m *ClaimRepository) UpdateAnalysis(ctx context.Context, input domain.UpdateAnalysisInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdateAnalysisCalled++
	if m.UpdateAnalysisErr != nil {
		return m.UpdateAnalysisErr
	}
	c, ok := m.claims[input.ClaimID]
	if !ok {
		return repository.ErrNotFound
	}
	c.FraudScore = input.FraudScore
	c.FraudReason = input.FraudReason
	c.RiskFactors = input.RiskFactors
	if input.FraudScore >= 0.80 {
		c.Status = domain.StatusFlagged
	} else {
		c.Status = domain.StatusApproved
	}
	return nil
}

func (m *ClaimRepository) UpdateStatus(ctx context.Context, input domain.ReviewInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdateStatusCalled++
	if m.UpdateStatusErr != nil {
		return m.UpdateStatusErr
	}
	c, ok := m.claims[input.ClaimID]
	if !ok {
		return repository.ErrNotFound
	}
	c.Status = input.Status
	c.AnalystID = input.AnalystID
	c.AnalystNotes = input.AnalystNotes
	return nil
}

func (m *ClaimRepository) UpdateStatusByID(ctx context.Context, claimID string, status domain.ClaimStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.claims[claimID]
	if !ok {
		return repository.ErrNotFound
	}
	c.Status = status
	return nil
}

func (m *ClaimRepository) Delete(ctx context.Context, id, companyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteCalled++
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	c, ok := m.claims[id]
	if !ok || c.CompanyID != companyID {
		return repository.ErrNotFound
	}
	delete(m.claims, id)
	return nil
}

func (m *ClaimRepository) GetDailyStats(ctx context.Context, companyID string, days int) ([]*domain.DailyStat, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetDailyStatsCalled++
	return []*domain.DailyStat{}, nil
}
