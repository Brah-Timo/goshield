// Package email provides SMTP email delivery.
package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"go.uber.org/zap"

	"github.com/goshield/pkg/config"
)

// Mailer sends emails via SMTP.
type Mailer struct {
	cfg    config.NotificationConfig
	logger *zap.Logger
}

// New creates a Mailer.
func New(cfg config.NotificationConfig, logger *zap.Logger) *Mailer {
	return &Mailer{cfg: cfg, logger: logger}
}

// FraudAlertData holds template variables for the fraud alert email.
type FraudAlertData struct {
	ClaimID    string
	FraudScore float64
	RiskLevel  string
	Amount     float64
	Reason     string
	ReviewURL  string
}

var fraudAlertTpl = template.Must(template.New("fraud_alert").Parse(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>GoShield Fraud Alert</title></head>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#dc2626">⚠️ Fraud Alert — GoShield</h2>
  <p>A claim has been flagged with a <strong>high fraud probability</strong>.</p>
  <table style="border-collapse:collapse;width:100%">
    <tr><td style="padding:8px;border:1px solid #e5e7eb;font-weight:bold">Claim ID</td>
        <td style="padding:8px;border:1px solid #e5e7eb">{{.ClaimID}}</td></tr>
    <tr><td style="padding:8px;border:1px solid #e5e7eb;font-weight:bold">Fraud Score</td>
        <td style="padding:8px;border:1px solid #e5e7eb;color:#dc2626">{{printf "%.1f%%" (mul .FraudScore 100)}}</td></tr>
    <tr><td style="padding:8px;border:1px solid #e5e7eb;font-weight:bold">Risk Level</td>
        <td style="padding:8px;border:1px solid #e5e7eb">{{.RiskLevel}}</td></tr>
    <tr><td style="padding:8px;border:1px solid #e5e7eb;font-weight:bold">Amount</td>
        <td style="padding:8px;border:1px solid #e5e7eb">${{printf "%.2f" .Amount}}</td></tr>
    <tr><td style="padding:8px;border:1px solid #e5e7eb;font-weight:bold">AI Reason</td>
        <td style="padding:8px;border:1px solid #e5e7eb">{{.Reason}}</td></tr>
  </table>
  <p style="margin-top:24px">
    <a href="{{.ReviewURL}}" style="background:#2563eb;color:white;padding:10px 20px;border-radius:6px;text-decoration:none">
      Review Claim
    </a>
  </p>
  <p style="color:#6b7280;font-size:12px;margin-top:32px">
    This is an automated alert from GoShield Insurance Fraud Detection Platform.
  </p>
</body>
</html>
`))

// SendFraudAlert sends a fraud alert email to the analyst team.
func (m *Mailer) SendFraudAlert(to []string, data FraudAlertData) error {
	if m.cfg.SMTPHost == "" {
		m.logger.Warn("SMTP not configured — skipping email", zap.String("claim_id", data.ClaimID))
		return nil
	}

	var body bytes.Buffer
	tplFuncs := template.FuncMap{
		"mul": func(a, b float64) float64 { return a * b },
	}
	tpl, err := template.New("fraud_alert").Funcs(tplFuncs).Parse(fraudAlertTpl.Tree.Root.String())
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}
	if err := tpl.Execute(&body, data); err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	msg := buildMessage(
		m.cfg.SMTPFrom,
		to,
		fmt.Sprintf("⚠️ Fraud Alert: Claim %s — Score %.0f%%", data.ClaimID, data.FraudScore*100),
		body.String(),
	)

	addr := fmt.Sprintf("%s:%d", m.cfg.SMTPHost, m.cfg.SMTPPort)
	var auth smtp.Auth
	if m.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", m.cfg.SMTPUser, m.cfg.SMTPPassword, m.cfg.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, m.cfg.SMTPFrom, to, []byte(msg)); err != nil {
		m.logger.Error("SMTP send failed", zap.Strings("to", to), zap.Error(err))
		return fmt.Errorf("smtp send: %w", err)
	}

	m.logger.Info("fraud alert email sent",
		zap.String("claim_id", data.ClaimID),
		zap.Strings("to", to),
	)
	return nil
}

// SendRaw sends a plain text email.
func (m *Mailer) SendRaw(to []string, subject, body string) error {
	if m.cfg.SMTPHost == "" {
		return nil
	}
	msg := buildMessage(m.cfg.SMTPFrom, to, subject, body)
	addr := fmt.Sprintf("%s:%d", m.cfg.SMTPHost, m.cfg.SMTPPort)
	var auth smtp.Auth
	if m.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", m.cfg.SMTPUser, m.cfg.SMTPPassword, m.cfg.SMTPHost)
	}
	return smtp.SendMail(addr, auth, m.cfg.SMTPFrom, to, []byte(msg))
}

func buildMessage(from string, to []string, subject, htmlBody string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", from))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(htmlBody)
	return sb.String()
}
