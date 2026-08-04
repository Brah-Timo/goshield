package usecase

import (
	"fmt"

	"github.com/goerp/goerp/internal/sales/domain"
	"github.com/goerp/goerp/internal/sales/repository"
	"github.com/goerp/goerp/internal/shared/events"
)

type SalesUsecase struct {
	repo *repository.SalesRepository
}

func NewSalesUsecase(repo *repository.SalesRepository) *SalesUsecase {
	return &SalesUsecase{repo: repo}
}

func (u *SalesUsecase) ListCustomers(tenantID, search string, page int) ([]*domain.Customer, int, error) {
	return u.repo.ListCustomers(tenantID, search, page)
}

func (u *SalesUsecase) GetCustomer(id, tenantID string) (*domain.Customer, error) {
	return u.repo.GetCustomer(id, tenantID)
}

func (u *SalesUsecase) CreateCustomer(c *domain.Customer) error {
	if c.Name == "" {
		return fmt.Errorf("customer name is required")
	}
	c.IsActive = true
	if c.Currency == "" {
		c.Currency = "USD"
	}
	return u.repo.CreateCustomer(c)
}

func (u *SalesUsecase) ListSalesOrders(f domain.SalesFilter) ([]*domain.SalesOrder, int, error) {
	return u.repo.ListSalesOrders(f)
}

func (u *SalesUsecase) CreateSalesOrder(o *domain.SalesOrder) error {
	if o.CustomerID == "" {
		return fmt.Errorf("customer is required")
	}
	// Calculate totals from lines
	var subtotal, taxAmt float64
	for _, l := range o.Lines {
		lineTotal := l.Quantity * l.UnitPrice * (1 - l.DiscountPct/100)
		l.Subtotal = lineTotal
		l.TaxAmount = lineTotal * l.TaxRate / 100
		l.Total = lineTotal + l.TaxAmount
		subtotal += lineTotal
		taxAmt += l.TaxAmount
	}
	o.Subtotal = subtotal
	o.TaxAmount = taxAmt
	o.Total = subtotal + taxAmt
	return u.repo.CreateSalesOrder(o)
}

func (u *SalesUsecase) ConfirmOrder(id, tenantID string) error {
	if err := u.repo.UpdateOrderState(id, tenantID, "confirmed"); err != nil {
		return err
	}
	events.Publish(events.SalesOrderConfirmed, map[string]string{"order_id": id, "tenant_id": tenantID})
	return nil
}

func (u *SalesUsecase) CancelOrder(id, tenantID string) error {
	return u.repo.UpdateOrderState(id, tenantID, "cancelled")
}

func (u *SalesUsecase) ListInvoices(f domain.SalesFilter) ([]*domain.Invoice, int, error) {
	return u.repo.ListInvoices(f)
}

func (u *SalesUsecase) CreateInvoice(inv *domain.Invoice) error {
	if inv.CustomerID == "" {
		return fmt.Errorf("customer is required")
	}
	if inv.InvoiceType == "" {
		inv.InvoiceType = "invoice"
	}
	if err := u.repo.CreateInvoice(inv); err != nil {
		return err
	}
	events.Publish(events.InvoiceCreated, inv)
	return nil
}
