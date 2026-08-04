package usecase
import (
	"github.com/goerp/goerp/internal/accounting/domain"
	"github.com/goerp/goerp/internal/accounting/repository"
)
type AccountingUsecase struct{ repo *repository.AccountingRepository }
func NewAccountingUsecase(repo *repository.AccountingRepository) *AccountingUsecase { return &AccountingUsecase{repo: repo} }
func (u *AccountingUsecase) ListAccounts(tenantID string) ([]*domain.Account, error) { return u.repo.ListAccounts(tenantID) }
func (u *AccountingUsecase) ListJournalEntries(f domain.AccountingFilter) ([]*domain.JournalEntry, int, error) { return u.repo.ListJournalEntries(f) }
func (u *AccountingUsecase) CreateJournalEntry(e *domain.JournalEntry) error { return u.repo.CreateJournalEntry(e) }
func (u *AccountingUsecase) PostJournalEntry(id, tenantID string) error { return u.repo.PostJournalEntry(id, tenantID) }
