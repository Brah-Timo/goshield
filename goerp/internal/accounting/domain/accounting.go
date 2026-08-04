package domain
import "time"
type Account struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	AccountType string    `json:"account_type"`
	ParentID    string    `json:"parent_id"`
	IsActive    bool      `json:"is_active"`
	Balance     float64   `json:"balance"`
	CreatedAt   time.Time `json:"created_at"`
}
type JournalEntry struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	EntryNumber string         `json:"entry_number"`
	EntryDate   string         `json:"entry_date"`
	Description string         `json:"description"`
	Reference   string         `json:"reference"`
	State       string         `json:"state"`
	TotalDebit  float64        `json:"total_debit"`
	TotalCredit float64        `json:"total_credit"`
	Lines       []*JournalLine `json:"lines,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}
type JournalLine struct {
	ID          string  `json:"id"`
	EntryID     string  `json:"entry_id"`
	AccountID   string  `json:"account_id"`
	AccountName string  `json:"account_name,omitempty"`
	Description string  `json:"description"`
	Debit       float64 `json:"debit"`
	Credit      float64 `json:"credit"`
}
type AccountingFilter struct { TenantID string; State string; Page int }
