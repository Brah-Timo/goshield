// Package consumer processes Kafka events for the notification-service.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/goshield/pkg/events"
	"github.com/goshield/services/notification-service/internal/email"
	"github.com/goshield/services/notification-service/internal/hub"
	"github.com/goshield/services/notification-service/internal/slack"
)

// NotificationConsumer handles Kafka events and dispatches notifications.
type NotificationConsumer struct {
	wsHub          *hub.Hub
	mailer         *email.Mailer
	slackNotifier  *slack.Notifier
	alertThreshold float64
	logger         *zap.Logger
}

// New creates a NotificationConsumer.
func New(
	wsHub *hub.Hub,
	mailer *email.Mailer,
	slackNotifier *slack.Notifier,
	alertThreshold float64,
	logger *zap.Logger,
) *NotificationConsumer {
	return &NotificationConsumer{
		wsHub:          wsHub,
		mailer:         mailer,
		slackNotifier:  slackNotifier,
		alertThreshold: alertThreshold,
		logger:         logger,
	}
}

// Handle routes events to the appropriate notification channels.
func (c *NotificationConsumer) Handle(ctx context.Context, evt events.Event) error {
	switch evt.Type {
	case events.EventClaimFlagged:
		return c.handleClaimFlagged(ctx, evt)
	case events.EventClaimAnalyzed:
		return c.handleClaimAnalyzed(ctx, evt)
	case events.EventClaimApproved, events.EventClaimRejected:
		return c.handleClaimStatusChange(ctx, evt)
	default:
		return nil
	}
}

func (c *NotificationConsumer) handleClaimFlagged(ctx context.Context, evt events.Event) error {
	var p events.ClaimFlaggedPayload
	if err := json.Unmarshal(evt.Payload, &p); err != nil {
		return fmt.Errorf("unmarshal claim.flagged: %w", err)
	}

	c.logger.Info("fraud alert triggered",
		zap.String("claim_id", p.ClaimID),
		zap.Float64("fraud_score", p.FraudScore),
		zap.String("company_id", p.CompanyID),
	)

	// 1. WebSocket broadcast to all connected dashboards for this company
	c.wsHub.Broadcast(hub.Message{
		CompanyID: p.CompanyID,
		Type:      "claim.flagged",
		Payload: map[string]any{
			"claim_id":    p.ClaimID,
			"fraud_score": p.FraudScore,
			"amount":      p.Amount,
			"reason":      p.Reason,
		},
	})

	// 2. Email alert (non-blocking — errors are logged, not returned)
	if p.FraudScore >= c.alertThreshold {
		analysts := []string{}
		if p.AnalystEmail != "" {
			analysts = append(analysts, p.AnalystEmail)
		}
		if len(analysts) > 0 {
			_ = c.mailer.SendFraudAlert(analysts, email.FraudAlertData{
				ClaimID:    p.ClaimID,
				FraudScore: p.FraudScore,
				RiskLevel:  riskLevel(p.FraudScore),
				Amount:     p.Amount,
				Reason:     p.Reason,
				ReviewURL:  fmt.Sprintf("https://app.goshield.io/claims/%s", p.ClaimID),
			})
		}
	}

	// 3. Slack alert
	_ = c.slackNotifier.SendFraudAlert(
		ctx,
		p.ClaimID,
		p.FraudScore,
		riskLevel(p.FraudScore),
		p.Reason,
		fmt.Sprintf("$%.2f", p.Amount),
	)

	return nil
}

func (c *NotificationConsumer) handleClaimAnalyzed(ctx context.Context, evt events.Event) error {
	var p events.ClaimAnalyzedPayload
	if err := json.Unmarshal(evt.Payload, &p); err != nil {
		return fmt.Errorf("unmarshal claim.analyzed: %w", err)
	}

	// Broadcast real-time update to dashboard
	c.wsHub.Broadcast(hub.Message{
		CompanyID: evt.CompanyID,
		Type:      "claim.analyzed",
		Payload: map[string]any{
			"claim_id":     p.ClaimID,
			"fraud_score":  p.FraudScore,
			"risk_factors": p.RiskFactors,
		},
	})
	return nil
}

func (c *NotificationConsumer) handleClaimStatusChange(_ context.Context, evt events.Event) error {
	type statusPayload struct {
		ClaimID string `json:"claim_id"`
		Status  string `json:"status"`
	}
	var p statusPayload
	if err := json.Unmarshal(evt.Payload, &p); err != nil {
		return nil // best-effort
	}

	c.wsHub.Broadcast(hub.Message{
		CompanyID: evt.CompanyID,
		Type:      evt.Type,
		Payload:   map[string]any{"claim_id": p.ClaimID, "status": p.Status},
	})
	return nil
}

func riskLevel(score float64) string {
	switch {
	case score >= 0.90:
		return "CRITICAL"
	case score >= 0.75:
		return "HIGH"
	case score >= 0.50:
		return "MEDIUM"
	default:
		return "LOW"
	}
}
