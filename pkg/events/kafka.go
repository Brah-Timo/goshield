// Package events provides Kafka producer and consumer implementations.
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// EventType constants for all GoShield domain events.
const (
	EventClaimCreated  = "claim.created"
	EventClaimAnalyzed = "claim.analyzed"
	EventClaimFlagged  = "claim.flagged"
	EventClaimApproved = "claim.approved"
	EventClaimRejected = "claim.rejected"
	EventClaimFailed   = "claim.failed"
)

// Event is the envelope for all Kafka messages.
type Event struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	OccurredAt  time.Time       `json:"occurred_at"`
	CompanyID   string          `json:"company_id"`
	Payload     json.RawMessage `json:"payload"`
}

// ClaimCreatedPayload is the payload for EventClaimCreated.
type ClaimCreatedPayload struct {
	ClaimID       string  `json:"claim_id"`
	DocURL        string  `json:"doc_url"`
	Amount        float64 `json:"amount"`
	ClaimType     string  `json:"claim_type"`
	Description   string  `json:"description"`
	PolicyNumber  string  `json:"policy_number"`
	UserID        string  `json:"user_id"`
	CompanyID     string  `json:"company_id"`
	AccountAgeDays int32  `json:"account_age_days"`
	PriorClaims   int32   `json:"prior_claims"`
	IncidentDate  string  `json:"incident_date"`
}

// ClaimAnalyzedPayload is the payload for EventClaimAnalyzed.
type ClaimAnalyzedPayload struct {
	ClaimID     string   `json:"claim_id"`
	FraudScore  float64  `json:"fraud_score"`
	Reason      string   `json:"reason"`
	RiskFactors []string `json:"risk_factors"`
}

// ClaimFlaggedPayload is the payload for EventClaimFlagged.
type ClaimFlaggedPayload struct {
	ClaimID       string  `json:"claim_id"`
	FraudScore    float64 `json:"fraud_score"`
	AnalystEmail  string  `json:"analyst_email"`
	CompanyID     string  `json:"company_id"`
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
}

// Producer wraps kafka-go Writer.
type Producer struct {
	writer *kafka.Writer
	logger *zap.Logger
}

// NewProducer creates a Kafka producer.
func NewProducer(brokers []string, logger *zap.Logger) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		BatchTimeout: 5 * time.Millisecond,
		BatchSize:    100,
		Compression:  kafka.Snappy,
	}
	return &Producer{writer: w, logger: logger}
}

// Publish serializes and publishes an event to the given topic.
func (p *Producer) Publish(ctx context.Context, topic string, evt Event) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(evt.ID),
		Value: data,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(evt.Type)},
			{Key: "company_id", Value: []byte(evt.CompanyID)},
		},
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.logger.Error("kafka publish failed",
			zap.String("topic", topic),
			zap.String("event_type", evt.Type),
			zap.Error(err),
		)
		return fmt.Errorf("publish to %s: %w", topic, err)
	}

	p.logger.Debug("event published",
		zap.String("topic", topic),
		zap.String("event_id", evt.ID),
		zap.String("event_type", evt.Type),
	)
	return nil
}

// PublishPayload is a convenience wrapper that marshals a payload struct.
func (p *Producer) PublishPayload(ctx context.Context, topic, eventType, id, companyID string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	return p.Publish(ctx, topic, Event{
		ID:         id,
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		CompanyID:  companyID,
		Payload:    data,
	})
}

// Close flushes and closes the producer.
func (p *Producer) Close() error {
	return p.writer.Close()
}

// Consumer wraps kafka-go Reader with retry logic.
type Consumer struct {
	reader  *kafka.Reader
	logger  *zap.Logger
	maxRetries int
}

// NewConsumer creates a Kafka consumer for a given topic and group.
func NewConsumer(brokers []string, topic, groupID string, logger *zap.Logger) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
		RetentionTime:  7 * 24 * time.Hour,
		MaxWait:        500 * time.Millisecond,
	})
	return &Consumer{reader: r, logger: logger, maxRetries: 3}
}

// HandlerFunc processes a single event.
type HandlerFunc func(ctx context.Context, evt Event) error

// Consume starts consuming messages and calls handler for each.
// On failure after maxRetries, message is skipped (DLQ should be configured at Kafka level).
func (c *Consumer) Consume(ctx context.Context, handler HandlerFunc) error {
	c.logger.Info("kafka consumer started",
		zap.String("topic", c.reader.Config().Topic),
		zap.String("group", c.reader.Config().GroupID),
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.logger.Error("fetch message failed", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		var evt Event
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			c.logger.Error("unmarshal event failed",
				zap.ByteString("value", msg.Value),
				zap.Error(err),
			)
			_ = c.reader.CommitMessages(ctx, msg)
			continue
		}

		// Process with retries.
		var processErr error
		for attempt := 1; attempt <= c.maxRetries; attempt++ {
			processErr = handler(ctx, evt)
			if processErr == nil {
				break
			}
			c.logger.Warn("event processing failed, retrying",
				zap.String("event_id", evt.ID),
				zap.Int("attempt", attempt),
				zap.Error(processErr),
			)
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}

		if processErr != nil {
			c.logger.Error("event processing permanently failed — skipping",
				zap.String("event_id", evt.ID),
				zap.String("event_type", evt.Type),
				zap.Error(processErr),
			)
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("commit message failed", zap.Error(err))
		}
	}
}

// Close closes the consumer.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
