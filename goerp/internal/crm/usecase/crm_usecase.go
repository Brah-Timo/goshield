package usecase

import (
	"fmt"

	"github.com/goerp/goerp/internal/crm/domain"
	"github.com/goerp/goerp/internal/crm/repository"
	"github.com/goerp/goerp/internal/shared/events"
)

type CRMUsecase struct {
	repo *repository.CRMRepository
	bus  *events.Bus
}

func NewCRMUsecase(repo *repository.CRMRepository) *CRMUsecase {
	return &CRMUsecase{repo: repo, bus: events.GetBus()}
}

// ─── LEADS ────────────────────────────────────────────────────────────────────

func (uc *CRMUsecase) ListLeads(filter domain.LeadFilter) ([]domain.Lead, int, error) {
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	return uc.repo.ListLeads(filter)
}

func (uc *CRMUsecase) GetLead(id, tenantID string) (*domain.Lead, error) {
	return uc.repo.GetLead(id, tenantID)
}

func (uc *CRMUsecase) CreateLead(req domain.CreateLeadRequest, tenantID, createdBy string) (*domain.Lead, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("lead name is required")
	}

	lead := &domain.Lead{
		TenantID:   tenantID,
		Name:       req.Name,
		Email:      req.Email,
		Phone:      req.Phone,
		Company:    req.Company,
		Source:     req.Source,
		Status:     domain.LeadStatusNew,
		AssignedTo: req.AssignedTo,
		Notes:      req.Notes,
		Tags:       req.Tags,
		CreatedBy:  createdBy,
	}

	if err := uc.repo.CreateLead(lead); err != nil {
		return nil, fmt.Errorf("failed to create lead: %w", err)
	}

	uc.bus.Publish(events.LeadCreated, lead)
	return lead, nil
}

func (uc *CRMUsecase) UpdateLead(id string, req domain.UpdateLeadRequest, tenantID string) (*domain.Lead, error) {
	lead, err := uc.repo.GetLead(id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("lead not found: %w", err)
	}

	if req.Name != "" {
		lead.Name = req.Name
	}
	if req.Email != "" {
		lead.Email = req.Email
	}
	if req.Phone != "" {
		lead.Phone = req.Phone
	}
	if req.Company != "" {
		lead.Company = req.Company
	}
	if req.Source != "" {
		lead.Source = req.Source
	}
	if req.Status != "" {
		lead.Status = req.Status
	}
	if req.AssignedTo != "" {
		lead.AssignedTo = req.AssignedTo
	}
	if req.Notes != "" {
		lead.Notes = req.Notes
	}
	if len(req.Tags) > 0 {
		lead.Tags = req.Tags
	}

	if err := uc.repo.UpdateLead(lead); err != nil {
		return nil, fmt.Errorf("failed to update lead: %w", err)
	}
	return lead, nil
}

func (uc *CRMUsecase) DeleteLead(id, tenantID string) error {
	return uc.repo.DeleteLead(id, tenantID)
}

// ConvertToOpportunity converts a lead to an opportunity
func (uc *CRMUsecase) ConvertToOpportunity(leadID, tenantID, createdBy string, req domain.CreateOpportunityRequest) (*domain.Opportunity, error) {
	lead, err := uc.repo.GetLead(leadID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("lead not found: %w", err)
	}

	req.LeadID = lead.ID
	if req.CustomerName == "" {
		req.CustomerName = lead.Name
	}
	if req.CustomerEmail == "" {
		req.CustomerEmail = lead.Email
	}
	if req.CustomerPhone == "" {
		req.CustomerPhone = lead.Phone
	}
	if req.Company == "" {
		req.Company = lead.Company
	}

	opp, err := uc.CreateOpportunity(req, tenantID, createdBy)
	if err != nil {
		return nil, err
	}

	// Update lead status to qualified
	lead.Status = domain.LeadStatusQualified
	uc.repo.UpdateLead(lead)

	return opp, nil
}

// ─── OPPORTUNITIES ────────────────────────────────────────────────────────────

func (uc *CRMUsecase) ListOpportunities(filter domain.OpportunityFilter) ([]domain.Opportunity, int, error) {
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	return uc.repo.ListOpportunities(filter)
}

func (uc *CRMUsecase) GetOpportunity(id, tenantID string) (*domain.Opportunity, error) {
	return uc.repo.GetOpportunity(id, tenantID)
}

func (uc *CRMUsecase) CreateOpportunity(req domain.CreateOpportunityRequest, tenantID, createdBy string) (*domain.Opportunity, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("opportunity name is required")
	}
	if req.CustomerName == "" {
		return nil, fmt.Errorf("customer name is required")
	}

	opp := &domain.Opportunity{
		TenantID:        tenantID,
		Name:            req.Name,
		LeadID:          req.LeadID,
		CustomerName:    req.CustomerName,
		CustomerEmail:   req.CustomerEmail,
		CustomerPhone:   req.CustomerPhone,
		Company:         req.Company,
		Stage:           req.Stage,
		ExpectedRevenue: req.ExpectedRevenue,
		Probability:     req.Probability,
		ExpectedClose:   req.ExpectedClose,
		AssignedTo:      req.AssignedTo,
		Description:     req.Description,
		CreatedBy:       createdBy,
	}
	if opp.Stage == "" {
		opp.Stage = domain.StageNew
	}

	if err := uc.repo.CreateOpportunity(opp); err != nil {
		return nil, fmt.Errorf("failed to create opportunity: %w", err)
	}
	return opp, nil
}

func (uc *CRMUsecase) UpdateOpportunity(id string, req domain.UpdateOpportunityRequest, tenantID string) (*domain.Opportunity, error) {
	opp, err := uc.repo.GetOpportunity(id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("opportunity not found: %w", err)
	}

	if req.Name != "" {
		opp.Name = req.Name
	}
	if req.CustomerName != "" {
		opp.CustomerName = req.CustomerName
	}
	if req.CustomerEmail != "" {
		opp.CustomerEmail = req.CustomerEmail
	}
	if req.Stage != "" {
		opp.Stage = req.Stage
	}
	if req.ExpectedRevenue > 0 {
		opp.ExpectedRevenue = req.ExpectedRevenue
	}
	if req.Probability >= 0 {
		opp.Probability = req.Probability
	}
	if req.AssignedTo != "" {
		opp.AssignedTo = req.AssignedTo
	}
	if req.Description != "" {
		opp.Description = req.Description
	}
	if req.LostReason != "" {
		opp.LostReason = req.LostReason
	}

	if err := uc.repo.UpdateOpportunity(opp); err != nil {
		return nil, fmt.Errorf("failed to update opportunity: %w", err)
	}
	return opp, nil
}

func (uc *CRMUsecase) WinOpportunity(id, tenantID string) (*domain.Opportunity, error) {
	opp, err := uc.repo.GetOpportunity(id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("opportunity not found: %w", err)
	}
	opp.Stage = domain.StageWon
	opp.Probability = 100
	if err := uc.repo.UpdateOpportunity(opp); err != nil {
		return nil, err
	}
	return opp, nil
}

func (uc *CRMUsecase) LoseOpportunity(id, tenantID, reason string) (*domain.Opportunity, error) {
	opp, err := uc.repo.GetOpportunity(id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("opportunity not found: %w", err)
	}
	opp.Stage = domain.StageLost
	opp.Probability = 0
	opp.LostReason = reason
	if err := uc.repo.UpdateOpportunity(opp); err != nil {
		return nil, err
	}
	return opp, nil
}

// ─── ACTIVITIES ───────────────────────────────────────────────────────────────

func (uc *CRMUsecase) ListActivities(filter domain.ActivityFilter) ([]domain.Activity, int, error) {
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	return uc.repo.ListActivities(filter)
}

func (uc *CRMUsecase) CreateActivity(req domain.CreateActivityRequest, tenantID, createdBy string) (*domain.Activity, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("activity title is required")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("activity type is required")
	}

	act := &domain.Activity{
		TenantID:      tenantID,
		Type:          req.Type,
		Title:         req.Title,
		Description:   req.Description,
		LeadID:        req.LeadID,
		OpportunityID: req.OpportunityID,
		AssignedTo:    req.AssignedTo,
		DueDate:       req.DueDate,
		IsDone:        false,
		CreatedBy:     createdBy,
	}

	if err := uc.repo.CreateActivity(act); err != nil {
		return nil, fmt.Errorf("failed to create activity: %w", err)
	}
	return act, nil
}

func (uc *CRMUsecase) CompleteActivity(id, tenantID string) error {
	return uc.repo.CompleteActivity(id, tenantID)
}

func (uc *CRMUsecase) GetPipelineStats(tenantID string) (*domain.PipelineStats, error) {
	return uc.repo.GetPipelineStats(tenantID)
}
