package usecase

import (
	"fmt"
	"github.com/goerp/goerp/internal/purchases/domain"
	"github.com/goerp/goerp/internal/purchases/repository"
)

type PurchasesUsecase struct{ repo *repository.PurchasesRepository }
func NewPurchasesUsecase(repo *repository.PurchasesRepository) *PurchasesUsecase { return &PurchasesUsecase{repo: repo} }
func (u *PurchasesUsecase) ListSuppliers(tenantID string, page int) ([]*domain.Supplier, int, error) { return u.repo.ListSuppliers(tenantID, page) }
func (u *PurchasesUsecase) CreateSupplier(s *domain.Supplier) error {
	if s.Name == "" { return fmt.Errorf("supplier name required") }
	s.IsActive = true
	if s.Currency == "" { s.Currency = "USD" }
	return u.repo.CreateSupplier(s)
}
func (u *PurchasesUsecase) ListPurchaseOrders(f domain.PurchaseFilter) ([]*domain.PurchaseOrder, int, error) { return u.repo.ListPurchaseOrders(f) }
func (u *PurchasesUsecase) CreatePurchaseOrder(o *domain.PurchaseOrder) error {
	if o.SupplierID == "" { return fmt.Errorf("supplier required") }
	if o.Currency == "" { o.Currency = "USD" }
	return u.repo.CreatePurchaseOrder(o)
}
