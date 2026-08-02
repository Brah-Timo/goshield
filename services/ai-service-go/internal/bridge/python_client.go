// Package bridge provides an HTTP client to the Python FastAPI AI service.
package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// InferenceRequest mirrors the Python FastAPI InferenceRequest schema.
type InferenceRequest struct {
	ClaimID          string  `json:"claim_id"`
	Amount           float64 `json:"amount"`
	ClaimType        string  `json:"claim_type"`
	Description      string  `json:"description"`
	PolicyNumber     string  `json:"policy_number"`
	UserID           string  `json:"user_id"`
	CompanyID        string  `json:"company_id"`
	AccountAgeDays   int32   `json:"account_age_days"`
	PriorClaimsCount int32   `json:"prior_claims_count"`
	IncidentDate     string  `json:"incident_date,omitempty"`
	DocURL           string  `json:"doc_url,omitempty"`
}

// RiskFactor mirrors the Python RiskFactor schema.
type RiskFactor struct {
	Feature    string  `json:"feature"`
	Importance float64 `json:"importance"`
	Direction  string  `json:"direction"`
	Value      float64 `json:"value"`
}

// InferenceResponse mirrors the Python FastAPI InferenceResponse schema.
type InferenceResponse struct {
	ClaimID          string       `json:"claim_id"`
	FraudScore       float64      `json:"fraud_score"`
	RiskLevel        string       `json:"risk_level"`
	Reason           string       `json:"reason"`
	RiskFactors      []string     `json:"risk_factors"`
	ShapValues       []RiskFactor `json:"shap_values"`
	ModelVersion     string       `json:"model_version"`
	Confidence       float64      `json:"confidence"`
	ProcessingTimeMs float64      `json:"processing_time_ms"`
}

// PythonAIClient calls the Python FastAPI inference endpoint.
type PythonAIClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewPythonAIClient creates a client for the Python AI service.
func NewPythonAIClient(baseURL string, logger *zap.Logger) *PythonAIClient {
	return &PythonAIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    20,
				IdleConnTimeout: 90 * time.Second,
			},
		},
		logger: logger,
	}
}

// Analyze calls /api/v1/analyze on the Python service.
func (c *PythonAIClient) Analyze(ctx context.Context, req InferenceRequest) (*InferenceResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/api/v1/analyze"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call python ai service: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("python ai service returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result InferenceResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	c.logger.Debug("ai inference complete",
		zap.String("claim_id", result.ClaimID),
		zap.Float64("fraud_score", result.FraudScore),
		zap.String("risk_level", result.RiskLevel),
		zap.Float64("ms", result.ProcessingTimeMs),
	)

	return &result, nil
}

// HealthCheck pings the Python service readiness endpoint.
func (c *PythonAIClient) HealthCheck(ctx context.Context) error {
	url := c.baseURL + "/api/v1/readyz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("python ai health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("python ai not ready: status %d", resp.StatusCode)
	}
	return nil
}
