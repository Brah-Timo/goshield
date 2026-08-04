package events

import (
	"sync"
)

// Event represents a domain event
type Event struct {
	Name    string
	Payload interface{}
}

// Handler is a function that handles an event
type Handler func(event Event)

// Bus is a simple in-process event bus
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

var Default = NewBus()

func NewBus() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe registers a handler for an event name
func (b *Bus) Subscribe(eventName string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], handler)
}

// Publish dispatches an event to all subscribed handlers
func (b *Bus) Publish(eventName string, payload interface{}) {
	b.mu.RLock()
	handlers := b.handlers[eventName]
	b.mu.RUnlock()

	event := Event{Name: eventName, Payload: payload}
	for _, h := range handlers {
		go h(event) // async dispatch
	}
}

// Publish to default bus
func Publish(eventName string, payload interface{}) {
	Default.Publish(eventName, payload)
}

// Subscribe to default bus
func Subscribe(eventName string, handler Handler) {
	Default.Subscribe(eventName, handler)
}

// GetBus returns the default event bus
func GetBus() *Bus {
	return Default
}

// Common event names
const (
	StockMoved            = "stock.moved"
	StockLow              = "stock.low"
	SalesOrderConfirmed   = "sales_order.confirmed"
	SalesOrderCancelled   = "sales_order.cancelled"
	InvoiceCreated        = "invoice.created"
	InvoicePaid           = "invoice.paid"
	PurchaseOrderCreated  = "purchase_order.created"
	PurchaseOrderReceived = "purchase_order.received"
	JournalEntryPosted    = "journal_entry.posted"
	UserLoggedIn          = "user.logged_in"
	NotificationRequested = "notification.requested"
	LeadCreated           = "lead.created"
	OpportunityWon        = "opportunity.won"
	EmployeeCreated       = "employee.created"
	LeaveApproved         = "leave.approved"
	PayrollGenerated      = "payroll.generated"
)
