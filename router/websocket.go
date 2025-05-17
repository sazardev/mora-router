package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// WebSocketConnection represents a client connection
type WebSocketConnection struct {
	// Standard websocket connection
	Conn        *http.ResponseWriter
	Request     *http.Request
	Params      Params
	ID          string
	SendChan    chan []byte
	ReceiveChan chan []byte
	closeChan   chan struct{}
	closed      bool
	mu          sync.RWMutex
	Hub         *WebSocketHub
	Metadata    map[string]interface{}
}

// WebSocketHub manages multiple WebSocket connections
type WebSocketHub struct {
	// Registered connections
	connections map[string]*WebSocketConnection
	// Inbound messages from connections
	broadcast chan []byte
	// Register requests from connections
	register chan *WebSocketConnection
	// Unregister requests from connections
	unregister chan *WebSocketConnection
	// Custom message handler
	messageHandler func(*WebSocketConnection, []byte)
	// Hub lock
	mu sync.RWMutex
}

// WebSocketHandlerFunc defines a function that handles WebSocket connections
type WebSocketHandlerFunc func(*WebSocketConnection, *http.Request, Params)

// WebSocketConfig contains configuration for the WebSocket handler
type WebSocketConfig struct {
	// Path for the WebSocket endpoint
	Path string
	// Message handler function
	MessageHandler func(*WebSocketConnection, []byte)
	// Connection handler function called when a new connection is established
	OnConnect WebSocketHandlerFunc
	// Disconnection handler function called when a connection is closed
	OnDisconnect WebSocketHandlerFunc
	// Ping interval in seconds (0 to disable)
	PingInterval int
	// Allowed origins for WebSocket connections (empty for all)
	AllowedOrigins []string
	// Maximum message size in bytes
	MaxMessageSize int64
	// Hub for broadcasting messages
	Hub *WebSocketHub
}

// WebSocketMessage represents a structured message
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// NewWebSocketHub creates a new hub for WebSocket connections
func NewWebSocketHub(messageHandler func(*WebSocketConnection, []byte)) *WebSocketHub {
	return &WebSocketHub{
		connections:    make(map[string]*WebSocketConnection),
		broadcast:      make(chan []byte),
		register:       make(chan *WebSocketConnection),
		unregister:     make(chan *WebSocketConnection),
		messageHandler: messageHandler,
	}
}

// Run starts the hub's main loop
func (h *WebSocketHub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.connections[conn.ID] = conn
			h.mu.Unlock()
		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.connections[conn.ID]; ok {
				delete(h.connections, conn.ID)
				close(conn.SendChan)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for _, conn := range h.connections {
				select {
				case conn.SendChan <- message:
				default:
					close(conn.SendChan)
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.connections, conn.ID)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHub) Broadcast(message []byte) {
	h.broadcast <- message
}

// GetConnection returns a WebSocket connection by ID
func (h *WebSocketHub) GetConnection(id string) (*WebSocketConnection, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conn, ok := h.connections[id]
	return conn, ok
}

// GetConnectionCount returns the number of active connections
func (h *WebSocketHub) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}

// ForEachConnection iterates over all connections and applies a function
func (h *WebSocketHub) ForEachConnection(f func(*WebSocketConnection)) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, conn := range h.connections {
		f(conn)
	}
}

// Send sends data to the WebSocket connection
func (c *WebSocketConnection) Send(data []byte) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return
	}
	c.SendChan <- data
}

// SendJSON marshals and sends JSON data
func (c *WebSocketConnection) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.Send(data)
	return nil
}

// Close closes the WebSocket connection
func (c *WebSocketConnection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	close(c.closeChan)
	// Unregister from hub
	if c.Hub != nil {
		c.Hub.unregister <- c
	}
}

// WithWebSocketHandler adds a WebSocket handler to the router
func WithWebSocketHandler(config WebSocketConfig) Option {
	return func(r *MoraRouter) {
		// Create default hub if none provided
		if config.Hub == nil && config.MessageHandler != nil {
			config.Hub = NewWebSocketHub(config.MessageHandler)
			go config.Hub.Run()
		}

		r.Get(config.Path, func(w http.ResponseWriter, req *http.Request, p Params) {
			// WebSocket upgrade and connection handling will go here
			// This is a placeholder - the real implementation would use gorilla/websocket
			wsConn := &WebSocketConnection{
				Conn:        &w,
				Request:     req,
				Params:      p,
				ID:          fmt.Sprintf("%s-%d", req.RemoteAddr, time.Now().UnixNano()),
				SendChan:    make(chan []byte, 256),
				ReceiveChan: make(chan []byte, 256),
				closeChan:   make(chan struct{}),
				Hub:         config.Hub,
				Metadata:    make(map[string]interface{}),
			}

			// Register with hub if available
			if config.Hub != nil {
				config.Hub.register <- wsConn
			}

			// Call OnConnect handler if provided
			if config.OnConnect != nil {
				config.OnConnect(wsConn, req, p)
			}

			// Write upgrade error
			log.Println("WebSocket upgrade would happen here - implement with gorilla/websocket")
			http.Error(w, "WebSocket support requires the gorilla/websocket package", http.StatusNotImplemented)
		})
	}
}

// WithWebSockets adds support for WebSocket connections
func WithWebSockets(paths map[string]WebSocketHandlerFunc) Option {
	return func(r *MoraRouter) {
		for path, handler := range paths {
			config := WebSocketConfig{
				Path:      path,
				OnConnect: handler,
			}
			WithWebSocketHandler(config)(r)
		}
	}
}

// WebSocketRoomOption defines options for creating a WebSocket room
type WebSocketRoomOption struct {
	// Authentication function
	Auth func(r *http.Request) bool
	// Maximum connections per room
	MaxConnections int
	// Message types handled by the room
	MessageTypes []string
	// Custom message handler
	MessageHandler func(*WebSocketConnection, []byte)
}

// WithRoomProvider adds a WebSocket room provider
func WithRoomProvider(pathPrefix string, options WebSocketRoomOption) Option {
	return func(r *MoraRouter) {
		// Map to store room hubs, keyed by room ID
		rooms := make(map[string]*WebSocketHub)
		var roomsMu sync.RWMutex

		// Handler for room WebSocket connections
		r.Get(pathPrefix+"/:roomID/ws", func(w http.ResponseWriter, req *http.Request, p Params) {
			roomID := p["roomID"]

			// Authentication check
			if options.Auth != nil && !options.Auth(req) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Find or create the room
			roomsMu.Lock()
			hub, exists := rooms[roomID]
			if !exists {
				hub = NewWebSocketHub(options.MessageHandler)
				rooms[roomID] = hub
				go hub.Run()
			}
			roomsMu.Unlock()

			// Check room capacity
			if options.MaxConnections > 0 && hub.GetConnectionCount() >= options.MaxConnections {
				http.Error(w, "Room is full", http.StatusServiceUnavailable)
				return
			}

			// WebSocket connection placeholder
			wsConn := &WebSocketConnection{
				Conn:        &w,
				Request:     req,
				Params:      p,
				ID:          fmt.Sprintf("%s-%d", req.RemoteAddr, time.Now().UnixNano()),
				SendChan:    make(chan []byte, 256),
				ReceiveChan: make(chan []byte, 256),
				closeChan:   make(chan struct{}),
				Hub:         hub,
				Metadata:    make(map[string]interface{}),
			}

			// Register connection with hub
			hub.register <- wsConn

			// Write upgrade error (placeholder for real implementation)
			http.Error(w, "WebSocket support requires the gorilla/websocket package", http.StatusNotImplemented)
		})

		// API endpoints for room management
		r.Get(pathPrefix, func(w http.ResponseWriter, req *http.Request, p Params) {
			roomsMu.RLock()
			roomList := make([]map[string]interface{}, 0, len(rooms))
			for id, hub := range rooms {
				roomList = append(roomList, map[string]interface{}{
					"id":              id,
					"connectionCount": hub.GetConnectionCount(),
				})
			}
			roomsMu.RUnlock()

			JSON(w, http.StatusOK, roomList)
		})

		r.Get(pathPrefix+"/:roomID", func(w http.ResponseWriter, req *http.Request, p Params) {
			roomID := p["roomID"]

			roomsMu.RLock()
			hub, exists := rooms[roomID]
			roomsMu.RUnlock()

			if !exists {
				http.Error(w, "Room not found", http.StatusNotFound)
				return
			}

			connections := make([]string, 0)
			hub.ForEachConnection(func(conn *WebSocketConnection) {
				connections = append(connections, conn.ID)
			})

			JSON(w, http.StatusOK, map[string]interface{}{
				"id":              roomID,
				"connectionCount": hub.GetConnectionCount(),
				"connections":     connections,
			})
		})

		r.Delete(pathPrefix+"/:roomID", func(w http.ResponseWriter, req *http.Request, p Params) {
			roomID := p["roomID"]

			roomsMu.Lock()
			hub, exists := rooms[roomID]
			if exists {
				delete(rooms, roomID)
				// Close all connections
				hub.ForEachConnection(func(conn *WebSocketConnection) {
					conn.Close()
				})
			}
			roomsMu.Unlock()

			w.WriteHeader(http.StatusNoContent)
		})
	}
}
