package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sazardev/mora-router/router"
)

func runHubExample() {
	// Create router with WebSocket support
	r := router.New(router.WithGorillaWebSocket())

	// Create a WebSocket hub for broadcasting messages
	hub := router.NewWebSocketHub("global", router.WebSocketConfig{
		MaxMessageSize: 1024 * 64,
		PingInterval:   30 * time.Second,
		OnConnect: func(conn *router.WebSocketConnection) {
			log.Printf("New connection: %s", conn.ID)
			conn.SendText("Welcome to the hub example!")
			conn.Hub.BroadcastMessage([]byte("New user connected!"))
		},
		OnDisconnect: func(conn *router.WebSocketConnection) {
			log.Printf("Connection closed: %s", conn.ID)
			conn.Hub.BroadcastMessage([]byte("A user has disconnected"))
		},
	})

	// Start the hub's event loop
	go hub.Run()

	// WebSocket endpoint that uses the hub
	r.WebSocket("/hub", func(conn *router.WebSocketConnection, msg []byte) {
		log.Printf("Received message from %s: %s", conn.ID, string(msg))

		// Broadcast the message to all clients
		hub.BroadcastMessage(msg)
	})

	// Serve a simple HTML demo page
	r.Get("/hub-demo", func(w http.ResponseWriter, r *http.Request, p router.Params) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>MoraRouter WebSocket Hub Demo</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .container { border: 1px solid #ccc; padding: 20px; margin-bottom: 20px; }
        .output { height: 300px; overflow-y: scroll; border: 1px solid #eee; padding: 10px; margin-top: 10px; }
        .message-form { display: flex; margin-top: 10px; }
        .message-input { flex: 1; padding: 8px; }
        button { padding: 8px 16px; background: #0066ff; color: white; border: none; cursor: pointer; }
        .system-msg { color: #777; font-style: italic; }
    </style>
</head>
<body>
    <h1>WebSocket Hub Example</h1>
    
    <div class="container">
        <h2>Broadcast Hub</h2>
        <p>All connected clients will receive messages</p>
        
        <div id="hub-output" class="output"></div>
        
        <div class="message-form">
            <input id="hub-message" class="message-input" type="text" placeholder="Type a message..." autocomplete="off">
            <button onclick="sendHubMessage()">Send</button>
        </div>
    </div>
    
    <script>
        // Hub WebSocket
        const hubWs = new WebSocket('ws://' + location.host + '/hub');
        
        hubWs.onopen = function() {
            logHub('Connected to hub', true);
        };
        
        hubWs.onmessage = function(e) {
            logHub('Broadcast: ' + e.data);
        };
        
        hubWs.onclose = function() {
            logHub('Disconnected from hub', true);
        };
        
        function sendHubMessage() {
            const msg = document.getElementById('hub-message').value;
            if (msg) {
                hubWs.send(msg);
                document.getElementById('hub-message').value = '';
            }
        }
        
        function logHub(text, isSystem = false) {
            const output = document.getElementById('hub-output');
            const div = document.createElement('div');
            if (isSystem) div.className = 'system-msg';
            div.textContent = text;
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
