package repository

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/goerp/goerp/internal/accounting/domain"
	"github.com/goerp/goerp/internal/shared/database"
)

type AccountingRepository struct{ db *database.DB }

func NewAccountingRepository(db *database.DB) *AccountingRepository {
	return &AccountingRepository{db: db}
}

// ---- helpers ----------------------------------------------------------------

func accParseTime(s string) time.Time {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05+00:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ---- ListAccounts -----------------------------------------------------------

func (r *AccountingRepository) ListAccounts(tenantID string) ([]*domain.Account, error) {
	rows, err := r.db.Query(`
		SELECT id, tenant_id, code, name, account_type,
		       COALESCE(parent_id,''), is_active, balance, created_at
		FROM chart_of_accounts
		WHERE tenant_id=?
		ORDER BY code`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Account
	for rows.Next() {
		a := &domain.Account{}
		var isActive int
		var createdStr string
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.Code, &a.Name, &a.AccountType,
			&a.ParentID, &isActive, &a.Balance, &createdStr,
		); err != nil {
			return nil, err
		}
		a.IsActive = isActive == 1
		a.CreatedAt = accParseTime(createdStr)
		list = append(list, a)
	}
	return list, nil
}

// ---- ListJournalEntries -----------------------------------------------------

func (r *AccountingRepository) ListJournalEntries(f domain.AccountingFilter) ([]*domain.JournalEntry, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	offset := (f.Page - 1) * 50

	rows, err := r.db.Query(`
		SELECT id, tenant_id, entry_number, entry_date,
		       COALESCE(description,''), COALESCE(reference,''),
		       state, total_debit, total_credit, created_at
		FROM journal_entries
		WHERE tenant_id=?
		ORDER BY created_at DESC
		LIMIT 50 OFFSET ?`, f.TenantID, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*domain.JournalEntry
	for rows.Next() {
		e := &domain.JournalEntry{}
		var createdStr string
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.EntryNumber, &e.EntryDate,
			&e.Description, &e.Reference,
			&e.State, &e.TotalDebit, &e.TotalCredit, &createdStr,
		); err != nil {
			return nil, 0, err
		}
		e.CreatedAt = accParseTime(createdStr)
		list = append(list, e)
	}

	var total int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM journal_entries WHERE tenant_id=?`, f.TenantID).Scan(&total)
	return list, total, nil
}

// ---- CreateJournalEntry -----------------------------------------------------

func (r *AccountingRepository) CreateJournalEntry(e *domain.JournalEntry) error {
	e.ID = uuid.New().String()
	e.CreatedAt = time.Now()
	if e.State == "" {
		e.State = "draft"
	}

	var count int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM journal_entries WHERE tenant_id=?`, e.TenantID).Scan(&count)
	e.EntryNumber = fmt.Sprintf("JE-%05d", count+1)

	var totalD, totalC float64
	for _, l := range e.Lines {
		totalD += l.Debit
		totalC += l.Credit
	}
	if totalD != totalC {
		return fmt.Errorf("debit (%.2f) must equal credit (%.2f)", totalD, totalC)
	}
	e.TotalDebit, e.TotalCredit = totalD, totalC

	today := time.Now().Format("2006-01-02")
	_, err := r.db.Exec(`
		INSERT INTO journal_entries
		  (id, tenant_id, entry_number, entry_date, description, reference,
		   state, total_debit, total_credit, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		e.ID, e.TenantID, e.EntryNumber, today,
		e.Description, e.Reference,
		e.State, e.TotalDebit, e.TotalCredit,
		e.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}

	// Insert journal lines
	for _, l := range e.Lines {
		lineID := uuid.New().String()
		_, _ = r.db.Exec(`
			INSERT INTO journal_lines
			  (id, journal_entry_id, account_id, description, debit, credit)
			VALUES (?,?,?,?,?,?)`,
			lineID, e.ID, l.AccountID, l.Description, l.Debit, l.Credit,
		)
	}
	return nil
}

// ---- PostJournalEntry -------------------------------------------------------

func (r *AccountingRepository) PostJournalEntry(id, tenantID string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec(`
		UPDATE journal_entries
		SET state='posted', posted_at=?
		WHERE id=? AND tenant_id=? AND state='draft'`,
		now, id, tenantID)
	return err
}
