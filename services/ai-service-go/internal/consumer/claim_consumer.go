// Package consumer processes claims.new Kafka events and dispatches to AI inference.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/goshield/pkg/events"
	"github.com/goshield/services/ai-service-go/internal/bridge"
)

// ClaimConsumer processes claim.created events from Kafka.
type ClaimConsumer struct {
	aiClient *bridge.PythonAIClient
	producer *events.Producer
	topics   topicConfig
	logger   *zap.Logger
}

type topicConfig struct {
	analyzed string
	flagged  string
	failed   string
}

// New creates a ClaimConsumer with all required dependencies.
func New(
	aiClient *bridge.PythonAIClient,
	producer *events.Producer,
	analyzedTopic, flaggedTopic, failedTopic string,
	logger *zap.Logger,
) *ClaimConsumer {
	return &ClaimConsumer{
		aiClient: aiClient,
		producer: producer,
		topics: topicConfig{
			analyzed: analyzedTopic,
			flagged:  flaggedTopic,
			failed:   failedTopic,
		},
		logger: logger,
	}
}

// Handle processes a single Kafka event from the claims.new topic.
func (c *ClaimConsumer) Handle(ctx context.Context, evt events.Event) error {
	if evt.Type != events.EventClaimCreated {
		return nil // ignore non-claim events on this topic
	}

	var payload events.ClaimCreatedPayload
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal claim.created payload: %w", err)
	}

	if payload.ClaimID == "" {
		return fmt.Errorf("claim.created event missing claim_id")
	}

	c.logger.Info("processing claim for AI analysis",
		zap.String("claim_id", payload.ClaimID),
		zap.Float64("amount", payload.Amount),
		zap.String("claim_type", payload.ClaimType),
	)

	// Wait briefly for doc upload to settle (production: use event ordering / doc.uploaded event)
	if payload.DocURL == "" {
		time.Sleep(500 * time.Millisecond)
	}

	// Call Python AI service
	inferReq := bridge.InferenceRequest{
		ClaimID:          payload.ClaimID,
		Amount:           payload.Amount,
		ClaimType:        payload.ClaimType,
		Description:      payload.Description,
		PolicyNumber:     payload.PolicyNumber,
		UserID:           payload.UserID,
		CompanyID:        payload.CompanyID,
		AccountAgeDays:   payload.AccountAgeDays,
		PriorClaimsCount: payload.PriorClaims,
		IncidentDate:     payload.IncidentDate,
		DocURL:           payload.DocURL,
	}

	inferCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	result, err := c.aiClient.Analyze(inferCtx, inferReq)
	if err != nil {
		c.logger.Error("AI analysis failed — publishing to DLQ",
			zap.String("claim_id", payload.ClaimID),
			zap.Error(err),
		)
		_ = c.publishFailed(ctx, payload, err)
		return fmt.Errorf("ai analysis for %s: %w", payload.ClaimID, err)
	}

	// Publish claim.analyzed event (include CompanyID for downstream routing)
	analyzedPayload := events.ClaimAnalyzedPayload{
		ClaimID:     result.ClaimID,
		CompanyID:   payload.CompanyID,
		FraudScore:  result.FraudScore,
		Reason:      result.Reason,
		RiskFactors: result.RiskFactors,
	}
	if err := c.producer.PublishPayload(
		ctx,
		c.topics.analyzed,
		events.EventClaimAnalyzed,
		uuid.New().String(),
		payload.CompanyID,
		analyzedPayload,
	); err != nil {
		c.logger.Error("failed to publish claim.analyzed",
			zap.String("claim_id", payload.ClaimID),
			zap.Error(err),
		)
		// Non-fatal: result is already computed; we log and continue.
	}

	// If fraud score is above flagging threshold, also publish claim.flagged
	if result.FraudScore >= 0.80 {
		flaggedPayload := events.ClaimFlaggedPayload{
			ClaimID:    result.ClaimID,
			FraudScore: result.FraudScore,
			CompanyID:  payload.CompanyID,
			Amount:     payload.Amount,
			Reason:     result.Reason,
		}
		if err := c.producer.PublishPayload(
			ctx,
			c.topics.flagged,
			events.EventClaimFlagged,
			uuid.New().String(),
			payload.CompanyID,
			flaggedPayload,
		); err != nil {
			c.logger.Warn("failed to publish claim.flagged",
				zap.String("claim_id", payload.ClaimID),
				zap.Error(err),
			)
		}
	}

	c.logger.Info("claim analysis complete",
		zap.String("claim_id", result.ClaimID),
		zap.Float64("fraud_score", result.FraudScore),
		zap.String("risk_level", result.RiskLevel),
	)
	return nil
}

func (c *ClaimConsumer) publishFailed(ctx context.Context, p events.ClaimCreatedPayload, cause error) error {
	type failedPayload struct {
		ClaimID string `json:"claim_id"`
		Reason  string `json:"reason"`
	}
	return c.producer.PublishPayload(
		ctx,
		c.topics.failed,
		events.EventClaimFailed,
		uuid.New().String(),
		p.CompanyID,
		failedPayload{ClaimID: p.ClaimID, Reason: cause.Error()},
	)
}
