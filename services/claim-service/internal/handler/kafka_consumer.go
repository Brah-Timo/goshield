// Package handler provides the Kafka consumer handler for the claim-service.
// It listens on the claims.analyzed topic for AI results.
package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/goshield/pkg/events"
	"github.com/goshield/services/claim-service/internal/service"
)

// KafkaHandler handles incoming Kafka events for the claim-service.
type KafkaHandler struct {
	svc    service.ClaimService
	logger *zap.Logger
}

// NewKafkaHandler creates a new KafkaHandler.
func NewKafkaHandler(svc service.ClaimService, logger *zap.Logger) *KafkaHandler {
	return &KafkaHandler{svc: svc, logger: logger}
}

// HandleEvent routes an incoming Kafka event to the correct handler function.
func (h *KafkaHandler) HandleEvent(ctx context.Context, evt events.Event) error {
	h.logger.Debug("received kafka event",
		zap.String("type", evt.Type),
		zap.String("id", evt.ID),
	)

	switch evt.Type {
	case events.EventClaimAnalyzed:
		return h.handleClaimAnalyzed(ctx, evt)
	default:
		// Ignore other event types on this topic.
		return nil
	}
}

func (h *KafkaHandler) handleClaimAnalyzed(ctx context.Context, evt events.Event) error {
	var payload events.ClaimAnalyzedPayload
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal claim.analyzed payload: %w", err)
	}

	if payload.ClaimID == "" {
		return fmt.Errorf("claim.analyzed event missing claim_id")
	}

	h.logger.Info("processing claim.analyzed event",
		zap.String("claim_id", payload.ClaimID),
		zap.Float64("fraud_score", payload.FraudScore),
	)

	if err := h.svc.HandleAnalysisResult(ctx, payload); err != nil {
		return fmt.Errorf("handle analysis result for %s: %w", payload.ClaimID, err)
	}

	h.logger.Info("claim analysis applied",
		zap.String("claim_id", payload.ClaimID),
		zap.Float64("fraud_score", payload.FraudScore),
		zap.String("reason", payload.Reason),
	)

	return nil
}
