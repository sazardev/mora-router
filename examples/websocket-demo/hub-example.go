package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sazardev/mora-router/router"
)

func runHubExample() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Println("Starting Hub Example...")

	// Create router with WebSocket support
	r := router.New(router.WithGorillaWebSocket())

	// Create a WebSocket config with handlers for the hub
	hubConfig := router.WebSocketConfig{
		Path:           "/hub",
		MaxMessageSize: 1024 * 64,
		PingInterval:   30 * time.Second,
		MessageHandler: func(conn *router.WebSocketConnection, msg []byte) {
			// Log the message
			log.Printf("Hub received message from %s: %s", conn.ID, string(msg))

			// Format the message with sender ID
			formattedMsg := fmt.Sprintf("User %s: %s", conn.ID[len(conn.ID)-4:], string(msg))

			// Immediately echo back to sender for confirmation
			conn.SendText(fmt.Sprintf("Message sent: %s", string(msg)))

			// Broadcast formatted message to all clients
			log.Printf("Broadcasting message to hub: %s", formattedMsg)
			conn.Hub.BroadcastMessage([]byte(formattedMsg))
		},
		OnConnect: func(conn *router.WebSocketConnection) {
			log.Printf("New connection: %s", conn.ID)

			// Send welcome message to the new client
			conn.SendText("¡Bienvenido al hub de chat!")

			// Notify all clients about new user
			userID := conn.ID[len(conn.ID)-4:] // Use last 4 digits as user identifier
			conn.Hub.BroadcastMessage([]byte(fmt.Sprintf("🟢 Usuario %s se ha conectado", userID)))
		},
		OnDisconnect: func(conn *router.WebSocketConnection) {
			log.Printf("Connection closed: %s", conn.ID)

			// Notify all clients about user disconnection
			userID := conn.ID[len(conn.ID)-4:]
			conn.Hub.BroadcastMessage([]byte(fmt.Sprintf("🔴 Usuario %s se ha desconectado", userID)))
		},
	}

	// Register the WebSocket handler with the router
	r.Get("/hub", router.WebSocketHandler(hubConfig))
	// Setup connection handlers using middleware
	r.Use(func(next router.HandlerFunc) router.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, params router.Params) {
			// Only apply to WebSocket requests
			if strings.HasPrefix(req.URL.Path, "/hub") && strings.ToLower(req.Header.Get("Upgrade")) == "websocket" {
				log.Printf("WebSocket middleware: new connection attempt to %s", req.URL.Path)
			}
			next(w, req, params)
		}
	})
	// Add simple ping endpoint for testing
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request, p router.Params) {
		w.Write([]byte("pong"))
	})

	// Serve a simple HTML demo page
	r.Get("/hub-demo", func(w http.ResponseWriter, r *http.Request, p router.Params) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>MoraRouter WebSocket Hub Demo</title>    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; background: #f5f5f5; }
        .container { border: 1px solid #ddd; border-radius: 8px; padding: 20px; background: white; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .output { height: 400px; overflow-y: scroll; border: 1px solid #ddd; padding: 10px; margin-top: 10px; background: #fafafa; border-radius: 4px; }
        .message-form { display: flex; margin-top: 10px; }
        .message-input { flex: 1; padding: 10px; border: 1px solid #ddd; border-radius: 4px 0 0 4px; font-size: 16px; }
        button { padding: 10px 20px; background: #0066ff; color: white; border: none; border-radius: 0 4px 4px 0; cursor: pointer; font-weight: bold; }
        button:hover { background: #0052cc; }
        .system-msg { color: #666; font-style: italic; padding: 5px; border-left: 3px solid #ccc; margin: 5px 0; }
        .user-msg { padding: 8px 12px; margin: 5px 0; border-radius: 10px; background: #e1f5fe; }
        h1 { color: #333; }
        h2 { color: #0066ff; margin-bottom: 5px; }
        .status { font-weight: bold; }
        .status.online { color: #4caf50; }
        .status.offline { color: #f44336; }
        .user-count { font-size: 14px; color: #666; float: right; }
        .info-bar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
    </style>
</head>
<body>
    <h1>Chat en Tiempo Real con MoraRouter</h1>
    
    <div class="container">
        <div class="info-bar">
            <h2>Chat Público</h2>
            <span class="status online" id="connection-status">Conectado</span>
        </div>
        <p>Todos los mensajes se transmiten a todos los clientes conectados</p>
        
        <div id="hub-output" class="output"></div>
        
        <div class="message-form">
            <input id="hub-message" class="message-input" type="text" placeholder="Escribe un mensaje..." autocomplete="off">
            <button onclick="sendHubMessage()">Enviar</button>
        </div>
    </div>
      <script>
        console.log("Initializing WebSocket connection...");
        
        // Hub WebSocket
        const hubWs = new WebSocket('ws://' + location.host + '/hub');
        let reconnectAttempts = 0;
        
        hubWs.onopen = function(event) {
            console.log("WebSocket connection opened:", event);
            document.getElementById('connection-status').className = 'status online';
            document.getElementById('connection-status').textContent = 'Conectado';
            logHub('Conectado al servidor de chat', true);
            reconnectAttempts = 0;
        };
        
        hubWs.onmessage = function(e) {
            console.log("WebSocket message received:", e.data);
            // Check if it's a system message
            const isSystem = e.data.startsWith('🟢') || e.data.startsWith('🔴') || e.data.startsWith('¡Bienvenido');
            logHub(e.data, isSystem);
        };
          hubWs.onclose = function() {
            document.getElementById('connection-status').className = 'status offline';
            document.getElementById('connection-status').textContent = 'Desconectado';
            logHub('Desconectado del servidor de chat. Intentando reconectar...', true);
            
            // Try to reconnect with exponential backoff
            setTimeout(function() {
                reconnectAttempts++;
                const timeout = Math.min(30000, Math.pow(2, reconnectAttempts) * 1000);
                logHub("Intento de reconexión " + reconnectAttempts + " en " + (timeout/1000) + " segundos...", true);
                
                // Reconnect
                setTimeout(function() {
                    window.hubWs = new WebSocket('ws://' + location.host + '/hub');
                }, timeout);
            }, 1000);
        };        function sendHubMessage() {
            const msg = document.getElementById('hub-message').value.trim();
            if (msg) {
                console.log("Sending message:", msg);
                try {
                    hubWs.send(msg);
                    logHub("Mensaje enviado: " + msg, true);
                    document.getElementById('hub-message').value = '';
                } catch (err) {
                    console.error("Error sending message:", err);
                    logHub("Error al enviar mensaje: " + err.message, true);
                }
            }
        }
        
        function logHub(text, isSystem = false) {
            const output = document.getElementById('hub-output');
            const div = document.createElement('div');
            
            if (isSystem) {
                div.className = 'system-msg';
                div.textContent = text;
            } else {
                div.className = 'user-msg';
                
                // Check if it's a user message
                if (text.startsWith('User ')) {
                    // Format user messages better
                    const colonIndex = text.indexOf(':');
                    if (colonIndex > 0) {
                        const user = text.substring(0, colonIndex);
                        const message = text.substring(colonIndex + 1).trim();
                        
                        // Create username element
                        const userSpan = document.createElement('strong');
                        userSpan.textContent = user + ': ';
                        div.appendChild(userSpan);
                        
                        // Add message text
                        const msgText = document.createTextNode(message);
                        div.appendChild(msgText);
                    } else {
                        div.textContent = text;
                    }
                } else {
                    div.textContent = text;
                }
            }
            
            // Add timestamp
            const timestamp = new Date().toLocaleTimeString();
            const timeSpan = document.createElement('span');
            timeSpan.className = 'timestamp';
            timeSpan.textContent = ' [' + timestamp + ']';
            timeSpan.style.fontSize = '11px';
            timeSpan.style.color = '#999';
            div.appendChild(timeSpan);
            
            // Add to chat and scroll
            output.appendChild(div);
            output.scrollTop = output.scrollHeight;
        }
        
        // Handle Enter key
        document.getElementById('hub-message').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') sendHubMessage();
        });
    </script>
</body>
</html>`

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})

	// Start the server
	addr := ":8081"
	fmt.Printf("Hub example started at http://localhost%s/hub-demo\n", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
