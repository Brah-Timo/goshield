// Package hub implements the WebSocket broadcast hub.
package hub

import (
	"encoding/json"
	"sync"

	"go.uber.org/zap"
)

// Message is broadcast to all subscribers for a company.
type Message struct {
	CompanyID string `json:"company_id"`
	Type      string `json:"type"`
	Payload   any    `json:"payload"`
}

// envelope is the JSON wire format sent to WebSocket clients.
type envelope struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// Client represents a single WebSocket connection.
type Client struct {
	ID        string
	CompanyID string
	Send      chan []byte
}

// Hub manages active WebSocket clients and broadcasts messages.
type Hub struct {
	mu         sync.RWMutex
	clients    map[string]map[string]*Client // companyID -> clientID -> *Client
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	logger     *zap.Logger
}

// New creates a Hub. Call Run() in a goroutine.
func New(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]map[string]*Client),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client, 16),
		unregister: make(chan *Client, 16),
		logger:     logger,
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(c *Client) { h.register <- c }

// Unregister removes a client from the hub.
func (h *Hub) Unregister(c *Client) { h.unregister <- c }

// Broadcast sends a message to all clients in the company.
func (h *Hub) Broadcast(msg Message) { h.broadcast <- msg }

// Run starts the hub event loop. Must be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			if h.clients[c.CompanyID] == nil {
				h.clients[c.CompanyID] = make(map[string]*Client)
			}
			h.clients[c.CompanyID][c.ID] = c
			h.mu.Unlock()
			h.logger.Debug("ws client registered",
				zap.String("client_id", c.ID),
				zap.String("company_id", c.CompanyID),
			)

		case c := <-h.unregister:
			h.mu.Lock()
			if co, ok := h.clients[c.CompanyID]; ok {
				if _, ok := co[c.ID]; ok {
					delete(co, c.ID)
					close(c.Send)
				}
				if len(co) == 0 {
					delete(h.clients, c.CompanyID)
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			company := h.clients[msg.CompanyID]
			h.mu.RUnlock()
			if len(company) == 0 {
				continue
			}
			data, _ := json.Marshal(envelope{Type: msg.Type, Payload: msg.Payload})
			for _, c := range company {
				select {
				case c.Send <- data:
				default:
					h.logger.Warn("dropped message for slow client", zap.String("client_id", c.ID))
				}
			}
		}
	}
}

// ClientCount returns number of connected clients for a company.
func (h *Hub) ClientCount(companyID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[companyID])
}
