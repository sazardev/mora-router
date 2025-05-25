package router

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// WebSocketConnection represents a client connection
type WebSocketConnection struct {
	// Standard websocket connection
	Conn        http.ResponseWriter
	Request     *http.Request
	ID          string
	Hub         *WebSocketHub
	Send        chan []byte
	isConnected bool
	closeMutex  sync.Mutex

	// Hijacked connection components
	netConn net.Conn
	bufrw   *bufio.ReadWriter
}

// SendText sends a text message to the client
func (c *WebSocketConnection) SendText(msg string) error {
	if !c.isConnected {
		return fmt.Errorf("connection closed")
	}
	frame := newTextFrame([]byte(msg))
	_, err := c.netConn.Write(frame)
	return err
}

// SendJSON marshals and sends a JSON message to the client
func (c *WebSocketConnection) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.SendText(string(data))
}

// Send binary data to the client
func (c *WebSocketConnection) SendBinary(data []byte) error {
	if !c.isConnected {
		return fmt.Errorf("connection closed")
	}
	frame := newBinaryFrame(data)
	_, err := c.netConn.Write(frame)
	return err
}

// Close the connection with normal closure
func (c *WebSocketConnection) Close() {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	if !c.isConnected {
		return
	}

	// Send close frame
	closeFrame := []byte{0x88, 0x02, 0x03, 0xE8} // Normal closure (1000)
	if c.netConn != nil {
		c.netConn.Write(closeFrame)
		c.netConn.Close()
	}
	c.isConnected = false

	// Remove from hub if present
	if c.Hub != nil {
		c.Hub.Unregister <- c
	}
}

// WebSocketHub manages a collection of connections
type WebSocketHub struct {
	// Registered connections
	Connections map[*WebSocketConnection]bool

	// Register requests
	Register chan *WebSocketConnection

	// Unregister requests
	Unregister chan *WebSocketConnection

	// Inbound messages to broadcast
	Broadcast chan []byte

	// Room identifier if in room mode
	Room string

	// Configuration
	Config WebSocketConfig
}

// NewWebSocketHub creates a new hub
func NewWebSocketHub(room string, cfg WebSocketConfig) *WebSocketHub {
	return &WebSocketHub{
		Connections: make(map[*WebSocketConnection]bool),
		Register:    make(chan *WebSocketConnection),
		Unregister:  make(chan *WebSocketConnection),
		Broadcast:   make(chan []byte),
		Room:        room,
		Config:      cfg,
	}
}

// Run starts the hub's event loop
func (h *WebSocketHub) Run() {
	for {
		select {
		case conn := <-h.Register:
			h.Connections[conn] = true
			if h.Config.OnConnect != nil {
				h.Config.OnConnect(conn)
			}

		case conn := <-h.Unregister:
			if _, ok := h.Connections[conn]; ok {
				delete(h.Connections, conn)
				close(conn.Send)
				if h.Config.OnDisconnect != nil {
					h.Config.OnDisconnect(conn)
				}
			}

		case msg := <-h.Broadcast:
			for conn := range h.Connections {
				select {
				case conn.Send <- msg:
				default:
					close(conn.Send)
					delete(h.Connections, conn)
				}
			}
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHub) BroadcastMessage(msg []byte) {
	h.Broadcast <- msg
}

// Count returns the number of active connections
func (h *WebSocketHub) Count() int {
	return len(h.Connections)
}

// WebSocketConfig contains the configuration for a WebSocket endpoint
type WebSocketConfig struct {
	Path           string
	MaxMessageSize int
	PingInterval   time.Duration
	AllowedOrigins []string
	MessageHandler func(conn *WebSocketConnection, msg []byte)
	OnConnect      func(conn *WebSocketConnection)
	OnDisconnect   func(conn *WebSocketConnection)
}

// WebSocketHandler handles a WebSocket connection
func WebSocketHandler(config WebSocketConfig) HandlerFunc {
	if config.MaxMessageSize == 0 {
		config.MaxMessageSize = 4096 // 4KB default
	}

	if config.PingInterval == 0 {
		config.PingInterval = 30 * time.Second
	}

	hub := NewWebSocketHub("", config)
	go hub.Run()

	return func(w http.ResponseWriter, r *http.Request, params Params) {
		// Check origin if configured
		if len(config.AllowedOrigins) > 0 {
			origin := r.Header.Get("Origin")
			allowed := false
			for _, o := range config.AllowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}
		}

		// Verify it's a websocket upgrade request
		if !isWebSocketUpgrade(r) {
			http.Error(w, "Expected WebSocket Upgrade", http.StatusBadRequest)
			return
		} // Get the underlying connection using hijack before doing the handshake
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "WebSocket error: connection doesn't support hijacking", http.StatusInternalServerError)
			return
		}

		netConn, bufrw, err := hijacker.Hijack()
		if err != nil {
			http.Error(w, fmt.Sprintf("WebSocket hijack failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Perform handshake by writing directly to the hijacked connection
		if err := writeHandshake(netConn, r); err != nil {
			netConn.Close()
			return
		}

		conn := &WebSocketConnection{
			Conn:        w,
			Request:     r,
			ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
			Hub:         hub,
			Send:        make(chan []byte, 256),
			isConnected: true,
			netConn:     netConn,
			bufrw:       bufrw,
		}

		hub.Register <- conn

		// Handle the connection in the current goroutine - no need for 'go' here
		// since we already hijacked the connection
		handleWebSocketConnection(conn, config)
	}
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Upgrade")) == "websocket" &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// performHandshake completes the WebSocket opening handshake (deprecated, use writeHandshake instead)
func performHandshake(w http.ResponseWriter, r *http.Request) bool {
	// Get the WebSocket key
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return false
	}

	// Calculate accept key (per RFC6455)
	h := sha1.New()
	h.Write([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	acceptKey := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Set response headers
	headers := w.Header()
	headers.Set("Upgrade", "websocket")
	headers.Set("Connection", "Upgrade")
	headers.Set("Sec-WebSocket-Accept", acceptKey)

	// Write response status
	w.WriteHeader(http.StatusSwitchingProtocols)

	return true
}

// writeHandshake writes the WebSocket handshake directly to the connection
func writeHandshake(conn net.Conn, r *http.Request) error {
	// Get the WebSocket key
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return fmt.Errorf("missing Sec-WebSocket-Key header")
	}

	// Calculate accept key (per RFC6455)
	h := sha1.New()
	h.Write([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	acceptKey := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Write handshake response directly to the connection
	handshake := fmt.Sprintf(
		"HTTP/1.1 101 Switching Protocols\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n\r\n",
		acceptKey,
	)

	_, err := conn.Write([]byte(handshake))
	return err
}

// handleWebSocketConnection reads frames and dispatches them
func handleWebSocketConnection(conn *WebSocketConnection, config WebSocketConfig) {
	defer conn.netConn.Close()

	// Set deadlines
	readDeadline := time.Now().Add(config.PingInterval + 10*time.Second)
	conn.netConn.SetReadDeadline(readDeadline)
	// Read loop
	for {
		// Read frame header
		frameHeader := make([]byte, 2)
		if _, err := io.ReadFull(conn.bufrw, frameHeader); err != nil {
			break
		}

		// Parse first two bytes for opcode and mask bit
		fin := (frameHeader[0] & 0x80) != 0
		opcode := frameHeader[0] & 0x0F
		masked := (frameHeader[1] & 0x80) != 0
		payloadLen := int(frameHeader[1] & 0x7F)

		// Handle extended payload length
		if payloadLen == 126 {
			extLen := make([]byte, 2)
			if _, err := io.ReadFull(conn.bufrw, extLen); err != nil {
				break
			}
			payloadLen = int(binary.BigEndian.Uint16(extLen))
		} else if payloadLen == 127 {
			extLen := make([]byte, 8)
			if _, err := io.ReadFull(conn.bufrw, extLen); err != nil {
				break
			}
			payloadLen = int(binary.BigEndian.Uint64(extLen))
		}

		// Limit payload size
		if payloadLen > config.MaxMessageSize {
			log.Printf("WebSocket message too large: %d bytes", payloadLen)
			conn.Close()
			break
		}

		// Read masking key if present
		var maskKey []byte
		if masked {
			maskKey = make([]byte, 4)
			if _, err := io.ReadFull(conn.bufrw, maskKey); err != nil {
				break
			}
		}

		// Read payload
		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(conn.bufrw, payload); err != nil {
			break
		}

		// Unmask the payload if needed
		if masked {
			for i := 0; i < payloadLen; i++ {
				payload[i] ^= maskKey[i%4]
			}
		}

		// Handle based on opcode
		switch opcode {
		case 0x1: // Text frame
			if config.MessageHandler != nil {
				config.MessageHandler(conn, payload)
			}

		case 0x2: // Binary frame
			if config.MessageHandler != nil {
				config.MessageHandler(conn, payload)
			}

		case 0x8: // Close frame
			conn.Close()
			return

		case 0x9: // Ping frame, respond with pong
			pongFrame := newPongFrame(payload)
			conn.netConn.Write(pongFrame)

		case 0xA: // Pong frame, reset deadline
			conn.netConn.SetReadDeadline(time.Now().Add(config.PingInterval + 10*time.Second))
		}

		if !fin {
			// TODO: handle message fragmentation
			log.Println("WebSocket: fragmentation not supported yet")
		}
	}
}

// Helper functions for creating WebSocket frames
func newTextFrame(data []byte) []byte {
	return createFrame(0x1, data)
}

func newBinaryFrame(data []byte) []byte {
	return createFrame(0x2, data)
}

func newPingFrame(data []byte) []byte {
	return createFrame(0x9, data)
}

func newPongFrame(data []byte) []byte {
	return createFrame(0xA, data)
}

func createFrame(opcode byte, data []byte) []byte {
	length := len(data)
	var header []byte

	// First byte: FIN bit + opcode
	b0 := 0x80 | opcode // FIN=1, opcode=given

	// Second byte: MASK bit + payload length
	var b1 byte
	var extBytes []byte

	if length < 126 {
		b1 = byte(length)
		header = []byte{b0, b1}
	} else if length <= 65535 {
		b1 = 126
		extBytes = make([]byte, 2)
		binary.BigEndian.PutUint16(extBytes, uint16(length))
		header = []byte{b0, b1}
		header = append(header, extBytes...)
	} else {
		b1 = 127
		extBytes = make([]byte, 8)
		binary.BigEndian.PutUint64(extBytes, uint64(length))
		header = []byte{b0, b1}
		header = append(header, extBytes...)
	}

	// Add payload
	frame := append(header, data...)
	return frame
}

// WebSocket functions for the router

// WithGorillaWebSocket adds WebSocket support to the router (compatibility layer but implements natively)
func WithGorillaWebSocket() Option {
	return func(r *MoraRouter) {
		// This is just a placeholder for compatibility
		// Our implementation doesn't require gorilla/websocket
	}
}

// WithChatRoom adds a basic chat room at the given path
func WithChatRoom(path string) Option {
	return func(r *MoraRouter) {
		config := WebSocketConfig{
			Path:           path,
			MaxMessageSize: 1024 * 64, // 64KB
			MessageHandler: func(conn *WebSocketConnection, msg []byte) {
				// Broadcast message to all clients
				conn.Hub.BroadcastMessage(msg)
			},
			OnConnect: func(conn *WebSocketConnection) {
				// Notify that a new user has joined
				conn.Hub.BroadcastMessage([]byte(fmt.Sprintf("* User joined (Total: %d)", conn.Hub.Count())))
			},
			OnDisconnect: func(conn *WebSocketConnection) {
				// Notify that a user has left
				conn.Hub.BroadcastMessage([]byte(fmt.Sprintf("* User left (Total: %d)", conn.Hub.Count())))
			},
		}

		r.WebSocket(path, config.MessageHandler)

		// Also add a basic chat UI
		chatUI := `
<!DOCTYPE html>
<html>
<head>
    <title>MoraRouter Chat</title>
    <style>
        body { margin: 0; padding: 0; font-family: sans-serif; }
        #chat { max-width: 800px; margin: 0 auto; padding: 20px; }
        #messages { height: 300px; border: 1px solid #ccc; overflow-y: scroll; margin-bottom: 10px; padding: 10px; }
        #input-area { display: flex; }
        #message { flex: 1; padding: 8px; }
        button { padding: 8px 16px; background: #0066ff; color: white; border: none; cursor: pointer; }
        .system { color: #999; font-style: italic; }
    </style>
</head>
<body>
    <div id="chat">
        <h2>MoraRouter Chat</h2>
        <div id="messages"></div>
        <div id="input-area">
            <input id="message" type="text" placeholder="Type a message..." autocomplete="off">
            <button onclick="sendMessage()">Send</button>
        </div>
    </div>
    
    <script>
        const messages = document.getElementById('messages');
        const messageInput = document.getElementById('message');
        
        // Create WebSocket connection
        const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const ws = new WebSocket(protocol + '//' + location.host + '` + path + `');
        
        ws.onopen = function() {
            addMessage('Connected to chat server', true);
        };
        
        ws.onmessage = function(e) {
            const msg = e.data;
            if (msg.startsWith('* ')) {
                addMessage(msg, true);
            } else {
                addMessage(msg, false);
            }
        };
        
        ws.onclose = function() {
            addMessage('Disconnected from chat server', true);
        };
        
        function addMessage(text, isSystem) {
            const div = document.createElement('div');
            if (isSystem) div.className = 'system';
            div.textContent = text;
            messages.appendChild(div);
            messages.scrollTop = messages.scrollHeight;
        }
        
        function sendMessage() {
            const text = messageInput.value.trim();
            if (text) {
                ws.send(text);
                messageInput.value = '';
            }
        }
        
        // Handle Enter key
        messageInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });
    </script>
</body>
</html>
`
		r.Get(path+"-ui", func(w http.ResponseWriter, r *http.Request, p Params) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(chatUI))
		})
	}
}

// WebSocket adds a WebSocket handler for the given path
func (r *MoraRouter) WebSocket(path string, handler func(*WebSocketConnection, []byte)) {
	config := WebSocketConfig{
		Path:           path,
		MessageHandler: handler,
		MaxMessageSize: 1024 * 64, // 64KB default
		PingInterval:   30 * time.Second,
	}

	r.Get(path, WebSocketHandler(config))
}

// WithWebSocketHandler adds a WebSocket handler with custom configuration
func WithWebSocketHandler(config WebSocketConfig) Option {
	return func(r *MoraRouter) {
		r.Get(config.Path, WebSocketHandler(config))
	}
}

// WithWebSockets allows multiple WebSocket endpoints to be defined at once
func WithWebSockets(handlers map[string]func(*WebSocketConnection, []byte)) Option {
	return func(r *MoraRouter) {
		for path, handler := range handlers {
			r.WebSocket(path, handler)
		}
	}
}
