package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sazardev/mora-router/router"
)

func main() {
	// Check if we should run the hub example
	if len(os.Args) > 1 && os.Args[1] == "hub" {
		runHubExample()
		return
	}
	// Create router with WebSocket support
	r := router.New(router.WithGorillaWebSocket(), router.WithLogging())

	// Basic WebSocket endpoint - echo server
	r.WebSocket("/ws", func(conn *router.WebSocketConnection, msg []byte) {
		log.Printf("Received message: %s", string(msg))
		// Echo the message back to the client
		conn.SendText(string(msg))
	})

	// WebSocket endpoint with JSON handling
	r.WebSocket("/ws-json", func(conn *router.WebSocketConnection, msg []byte) {
		// Example of sending JSON response
		conn.SendJSON(map[string]interface{}{
			"echo": string(msg),
			"time": time.Now().Format(time.RFC3339),
		})
	})
	// Add a chat room with built-in UI
	router.WithChatRoom("/chat")(r)

	// Serve a simple HTML demo page for the WebSocket examples
	r.Get("/", func(w http.ResponseWriter, r *http.Request, p router.Params) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>MoraRouter WebSocket Demo</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .demo-box { border: 1px solid #ccc; padding: 20px; margin-bottom: 20px; }
        .output { height: 200px; overflow-y: scroll; border: 1px solid #eee; padding: 10px; margin-top: 10px; }
        button, input { padding: 5px; }
    </style>
</head>
<body>
    <h1>MoraRouter WebSocket Demos</h1>
    
    <div class="demo-box">
        <h2>Echo WebSocket</h2>
        <input id="echo-message" type="text" placeholder="Type a message...">
        <button onclick="sendEchoMessage()">Send</button>
        <div id="echo-output" class="output"></div>
    </div>

    <div class="demo-box">
        <h2>JSON WebSocket</h2>
        <input id="json-message" type="text" placeholder="Type a message...">
        <button onclick="sendJsonMessage()">Send</button>
        <div id="json-output" class="output"></div>
    </div>
    
    <div class="demo-box">
        <h2>Chat Room Demo</h2>
        <p>The chat demo is available at <a href="/chat-ui" target="_blank">/chat-ui</a></p>
    </div>
    
    <script>
        // Echo WebSocket
        const echoWs = new WebSocket('ws://' + location.host + '/ws');
        
        echoWs.onopen = function() {
            logEcho('Connection opened');
        };
        
        echoWs.onmessage = function(e) {
            logEcho('Received: ' + e.data);
        };
        
        echoWs.onclose = function() {
            logEcho('Connection closed');
        };
        
        function sendEchoMessage() {
            const msg = document.getElementById('echo-message').value;
            if (msg) {
                echoWs.send(msg);
                logEcho('Sent: ' + msg);
                document.getElementById('echo-message').value = '';
            }
        }
        
        function logEcho(text) {
            const output = document.getElementById('echo-output');
            const div = document.createElement('div');
            div.textContent = text;
            output.appendChild(div);
            output.scrollTop = output.scrollHeight;
        }
        
        // JSON WebSocket
        const jsonWs = new WebSocket('ws://' + location.host + '/ws-json');
        
        jsonWs.onopen = function() {
            logJson('Connection opened');
        };
        
        jsonWs.onmessage = function(e) {
            try {
                const data = JSON.parse(e.data);
                logJson('Received: ' + JSON.stringify(data, null, 2));
            } catch (err) {
                logJson('Received (non-JSON): ' + e.data);
            }
        };
        
        jsonWs.onclose = function() {
            logJson('Connection closed');
        };
        
        function sendJsonMessage() {
            const msg = document.getElementById('json-message').value;
            if (msg) {
                jsonWs.send(msg);
                logJson('Sent: ' + msg);
                document.getElementById('json-message').value = '';
            }
        }
        
        function logJson(text) {
            const output = document.getElementById('json-output');
            const div = document.createElement('div');
            div.textContent = text;
            output.appendChild(div);
            output.scrollTop = output.scrollHeight;
        }
        
        // Handle Enter key
        document.getElementById('echo-message').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') sendEchoMessage();
        });
        
        document.getElementById('json-message').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') sendJsonMessage();
        });
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})

	// Start the server
	addr := ":8080"
	fmt.Printf("WebSocket server started at http://localhost%s\n", addr)
	fmt.Printf("Available endpoints:\n")
	fmt.Printf("- Echo WebSocket: ws://localhost%s/ws\n", addr)
	fmt.Printf("- JSON WebSocket: ws://localhost%s/ws-json\n", addr)
	fmt.Printf("- Chat Room UI: http://localhost%s/chat-ui\n", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
