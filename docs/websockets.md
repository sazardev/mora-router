# WebSockets with MoraRouter

Building real-time applications is a breeze with MoraRouter's WebSocket support. This guide covers everything from basic WebSocket setup to advanced features like chat rooms and message broadcasting.

## Basic WebSocket Setup

MoraRouter provides a simple way to add WebSocket functionality to your Go application:

```go
import (
    "log"
    "net/http"
    
    "github.com/yourusername/mora-router/router"
)

func main() {
    // Create router with WebSocket support
    r := router.New(router.WithGorillaWebSocket())
    
    // Basic WebSocket endpoint
    r.WebSocket("/ws", func(conn *router.WebSocketConnection, msg []byte) {
        // Echo the message back to the client
        conn.Send(msg)
    })
    
    log.Println("WebSocket server started on ws://localhost:8080/ws")
    http.ListenAndServe(":8080", r)
}
```

## Client-Side Implementation

Here's a simple JavaScript client to connect to your WebSocket endpoint:

```html
<!DOCTYPE html>
<html>
<head>
    <title>MoraRouter WebSocket Demo</title>
</head>
<body>
    <h1>WebSocket Echo Test</h1>
    <input id="message" type="text" placeholder="Type a message...">
    <button onclick="sendMessage()">Send</button>
    <div id="output"></div>
    
    <script>
        const ws = new WebSocket('ws://localhost:8080/ws');
        
        ws.onopen = function() {
            document.getElementById('output').innerHTML += '<p>Connection opened</p>';
        };
        
        ws.onmessage = function(e) {
            document.getElementById('output').innerHTML += '<p>Received: ' + e.data + '</p>';
        };
        
        ws.onclose = function() {
            document.getElementById('output').innerHTML += '<p>Connection closed</p>';
        };
        
        function sendMessage() {
            const msg = document.getElementById('message').value;
            ws.send(msg);
            document.getElementById('output').innerHTML += '<p>Sent: ' + msg + '</p>';
            document.getElementById('message').value = '';
        }
    </script>
</body>
</html>
```

## Advanced WebSocket Configuration

MoraRouter allows fine-grained control over your WebSocket connections:

```go
r := router.New(
    router.WithGorillaWebSocket(),
    router.WithWebSocketHandler(router.WebSocketConfig{
        Path: "/ws",
        // Optional configuration
        PingInterval: 30, // seconds
        MaxMessageSize: 4096, // bytes
        AllowedOrigins: []string{"example.com"}, // CORS for WebSockets
        // Event handlers
        OnConnect: func(conn *router.WebSocketConnection, req *http.Request, p router.Params) {
            log.Printf("New connection: %s from %s", conn.ID, req.RemoteAddr)
        },
        OnDisconnect: func(conn *router.WebSocketConnection) {
            log.Printf("Connection closed: %s", conn.ID)
        },
        MessageHandler: func(conn *router.WebSocketConnection, msg []byte) {
            // Process incoming message
            log.Printf("Received message from %s: %s", conn.ID, string(msg))
            
            // Send response
            conn.SendJSON(map[string]interface{}{
                "echo": string(msg),
                "time": time.Now().Format(time.RFC3339),
            })
        },
    })
)
```

## Implementing Chat Rooms

MoraRouter makes it easy to create chat applications with room functionality:

```go
r := router.New(
    router.WithGorillaWebSocket(),
    // Built-in chat room with UI
    router.WithChatRoom("/chat")
)
```

This automatically creates a WebSocket endpoint at `/chat` and serves a simple chat UI at `/chat-ui`.

### Custom Chat Implementation

For more control over your chat functionality:

```go
// Create a hub to manage connections
hub := router.NewWebSocketHub()

r := router.New(
    router.WithGorillaWebSocket()
)

// Handle new WebSocket connections
r.WebSocket("/chat/:room", func(conn *router.WebSocketConnection, msg []byte) {
    // Get room name from URL parameters
    roomName := conn.Params["room"]
    
    // Join room on connect
    if !conn.HasMetadata("joined") {
        hub.JoinRoom(roomName, conn)
        conn.SetMetadata("joined", true)
        conn.SetMetadata("room", roomName)
        
        // Notify room about new user
        hub.BroadcastToRoom(roomName, []byte("New user joined the chat"))
        return
    }
    
    // Broadcast message to room
    hub.BroadcastToRoom(roomName, msg)
})
```

## Dynamic Room Creation

MoraRouter supports dynamic room creation and management:

```go
r := router.New(
    router.WithGorillaWebSocket(),
    router.WithRoomProvider("/api/rooms", router.WebSocketRoomOption{
        MaxConnections: 100,
        MessageHandler: func(conn *router.WebSocketConnection, msg []byte) {
            // Broadcast message to all in the room
            conn.Hub.Broadcast(msg)
        },
    })
)
```

This creates:
- `POST /api/rooms` - Creates a new room and returns its ID
- `GET /api/rooms` - Lists all active rooms
- `DELETE /api/rooms/:id` - Closes a room and disconnects all clients
- `WS /api/rooms/:id` - WebSocket endpoint for connecting to a specific room

## Working with Binary Data

MoraRouter handles both text and binary WebSocket messages:

```go
r.WebSocket("/binary", func(conn *router.WebSocketConnection, msg []byte) {
    // Check message type
    if conn.MessageType() == websocket.BinaryMessage {
        // Process binary data
        processImage(msg)
    } else {
        // Process text data
        processText(string(msg))
    }
})
```

## Broadcasting to Multiple Connections

```go
// Global broadcaster
broadcaster := router.NewBroadcaster()

r.WebSocket("/notifications", func(conn *router.WebSocketConnection, msg []byte) {
    // Add connection to broadcaster
    broadcaster.Add(conn)
})

// Later, in your notification system:
broadcaster.Broadcast([]byte("System maintenance in 5 minutes"))
```

## Authentication for WebSockets

Secure your WebSocket endpoints with authentication middleware:

```go
authMiddleware := func(next router.WebSocketHandlerFunc) router.WebSocketHandlerFunc {
    return func(conn *router.WebSocketConnection, msg []byte) {
        // Check if authenticated
        if !conn.HasMetadata("authenticated") {
            // First message should be auth token
            token := string(msg)
            
            // Validate token (simplified example)
            if token == "secret-token" {
                conn.SetMetadata("authenticated", true)
                conn.Send([]byte("Authentication successful"))
                return
            }
            
            // Authentication failed
            conn.Send([]byte("Authentication failed"))
            conn.Close()
            return
        }
        
        // Already authenticated, proceed to handler
        next(conn, msg)
    }
}

// Apply middleware to WebSocket handler
r.WebSocket("/secure", authMiddleware(func(conn *router.WebSocketConnection, msg []byte) {
    conn.Send([]byte("Secure message received: " + string(msg)))
}))
```

## Testing WebSocket Endpoints

MoraRouter includes utilities for testing WebSocket endpoints:

```go
func TestWebSocketEcho(t *testing.T) {
    r := router.New(router.WithGorillaWebSocket())
    r.WebSocket("/echo", func(conn *router.WebSocketConnection, msg []byte) {
        conn.Send(msg)
    })
    
    // Create test WebSocket client
    client := router.NewTestClient(r)
    wsClient := client.WebSocket("/echo")
    
    // Send message
    wsClient.Send([]byte("Hello, WebSocket!"))
    
    // Wait for response
    msg, err := wsClient.Receive()
    if err != nil {
        t.Fatalf("Failed to receive message: %v", err)
    }
    
    if string(msg) != "Hello, WebSocket!" {
        t.Errorf("Expected 'Hello, WebSocket!', got '%s'", string(msg))
    }
    
    wsClient.Close()
}
```

## Performance Considerations

For high-performance WebSocket applications:

1. **Connection Limits**: Set appropriate limits for your environment
   ```go
   router.WithWebSocketHandler(router.WebSocketConfig{
       Path: "/ws",
       MaxConnections: 10000,
       OverflowAction: router.RejectConnection,
   })
   ```

2. **Message Size Limits**: Prevent memory issues with large messages
   ```go
   router.WithWebSocketHandler(router.WebSocketConfig{
       Path: "/ws",
       MaxMessageSize: 64 * 1024, // 64KB
   })
   ```

3. **Compression**: Enable WebSocket compression for bandwidth savings
   ```go
   router.WithWebSocketHandler(router.WebSocketConfig{
       Path: "/ws",
       EnableCompression: true,
   })
   ```

## Best Practices

1. **Always handle connection errors** on both server and client
2. **Implement reconnection logic** in your client applications
3. **Use ping/pong frames** to detect dead connections (MoraRouter handles this automatically)
4. **Secure your WebSocket endpoints** with authentication
5. **Rate limit connection attempts** to prevent DoS attacks

## Sample Applications

Check out these comprehensive WebSocket examples in the examples directory:
- `examples/websocket-chat` - Full-featured chat application
- `examples/websocket-dashboard` - Real-time dashboard with metrics
- `examples/websocket-game` - Simple multiplayer game using WebSockets

## Conclusion

MoraRouter makes WebSocket implementation straightforward while providing the flexibility needed for advanced real-time applications. Whether you're building a simple chat app or a complex real-time system, these tools will help you get the job done efficiently.

Have fun building real-time applications with MoraRouter! 🚀
