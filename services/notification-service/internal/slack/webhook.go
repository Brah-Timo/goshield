// Package slack provides Slack webhook notification delivery.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Notifier sends messages to a Slack incoming webhook.
type Notifier struct {
	webhookURL string
	httpClient *http.Client
	logger     *zap.Logger
}

// New creates a Notifier. If webhookURL is empty all sends are no-ops.
func New(webhookURL string, logger *zap.Logger) *Notifier {
	return &Notifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

type slackBlock struct {
	Type string      `json:"type"`
	Text *slackText  `json:"text,omitempty"`
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type slackPayload struct {
	Text   string       `json:"text,omitempty"`
	Blocks []slackBlock `json:"blocks,omitempty"`
}

// SendFraudAlert posts a fraud alert card to Slack.
func (n *Notifier) SendFraudAlert(ctx context.Context, claimID string, fraudScore float64, riskLevel, reason, amount string) error {
	if n.webhookURL == "" {
		n.logger.Debug("Slack webhook not configured — skipping")
		return nil
	}

	emoji := "🟡"
	if riskLevel == "CRITICAL" {
		emoji = "🔴"
	} else if riskLevel == "HIGH" {
		emoji = "🟠"
	}

	payload := slackPayload{
		Blocks: []slackBlock{
			{
				Type: "header",
				Text: &slackText{
					Type: "plain_text",
					Text: fmt.Sprintf("%s GoShield Fraud Alert — %s Risk", emoji, riskLevel),
				},
			},
			{
				Type: "section",
				Text: &slackText{
					Type: "mrkdwn",
					Text: fmt.Sprintf(
						"*Claim ID:* `%s`\n*Fraud Score:* %.1f%%\n*Risk Level:* %s\n*Amount:* %s\n*AI Reason:* %s",
						claimID, fraudScore*100, riskLevel, amount, reason,
					),
				},
			},
		},
	}

	return n.send(ctx, payload)
}

// SendMessage posts a plain text message to Slack.
func (n *Notifier) SendMessage(ctx context.Context, message string) error {
	if n.webhookURL == "" {
		return nil
	}
	return n.send(ctx, slackPayload{Text: message})
}

func (n *Notifier) send(ctx context.Context, payload slackPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send slack webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	n.logger.Debug("Slack notification sent")
	return nil
}
