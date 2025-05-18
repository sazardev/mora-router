# Frequently Asked Questions (FAQ)

This document answers common questions about MoraRouter and provides solutions to common problems you might encounter.

## General Questions

### What is MoraRouter?

MoraRouter is a powerful HTTP router for Go applications, inspired by Django's routing capabilities but designed specifically for Go's concurrency model and performance characteristics. It provides advanced routing, middleware, resource controllers, and many other features to simplify web application development.

### Why create another Go router when there are already plenty of options?

MoraRouter was created to address specific pain points we experienced with existing routers:

1. **Feature completeness** - Most routers are either too minimalist (requiring lots of boilerplate) or too opinionated (forcing specific patterns)
2. **Developer experience** - We wanted something with Django-like ergonomics but with Go's performance
3. **Advanced patterns** - Support for RESTful resources, validations, and API versioning was often missing or required complex setup
4. **Testing tools** - Built-in utilities for testing routes were absent in most routers
5. **Documentation generation** - OpenAPI/Swagger support was usually an afterthought

MoraRouter aims to provide a "batteries included but removable" approach, where you get a rich feature set without sacrificing performance or flexibility.

### How does MoraRouter compare to other Go routers?

Here's a feature comparison with popular Go routers:

| Feature | MoraRouter | Gin | Chi | Echo | HttpRouter |
|---------|------------|-----|-----|------|-----------|
| **Pattern Matching** | Advanced (regex, typed) | Basic | Moderate | Basic | Efficient |
| **Middleware** | Rich ecosystem | Good | Good | Good | Minimal |
| **Groups** | Deep nesting | Yes | Yes | Yes | Limited |
| **REST Resources** | Built-in | Manual | Manual | Manual | Manual |
| **Validation** | Built-in | Plugins | Manual | Manual | Manual |
| **Testing Tools** | Comprehensive | Basic | Minimal | Good | None |
| **Websockets** | Built-in | Plugins | Plugins | Built-in | Plugins |
| **Documentation** | OpenAPI built-in | Plugins | Plugins | Plugins | None |
| **Hot Reload** | Yes | No | No | No | No |
| **Performance** | Very Good | Excellent | Very Good | Excellent | Outstanding |
| **Learning Curve** | Moderate | Low | Low | Low | Low |

### Is MoraRouter production-ready?

Yes! MoraRouter has been used in production environments handling substantial traffic. Its core routing engine is stable and well-tested. As with any framework, we recommend thorough testing before deploying to production.

### What's the performance impact of all these features?

MoraRouter is designed with performance in mind. The core routing engine uses a radix tree algorithm similar to other high-performance routers. Many features are opt-in, so you only pay the performance cost for what you use.

In our benchmarks, basic routing performance is comparable to Chi and within 15-20% of bare-metal HttpRouter. When using advanced features, there is a small overhead, but it's typically negligible compared to the actual application logic.

See the [Performance](performance.md) guide for detailed benchmarks and optimization tips.

### What does the name "MoraRouter" mean?

"Mora" is a linguistic term for a unit of sound length. We chose this name to symbolize the router's focus on precision and elegant structure. Also, it sounded cool and the domain name was available! 😎

## Installation & Setup

### What are the minimum Go version requirements?

MoraRouter works with Go 1.18 and above. Some features (like generics-based handlers) require Go 1.18+.

### Do I need to install any dependencies?

No external dependencies are required for core functionality. Some advanced features may have optional dependencies, which are clearly documented.

### Can I use MoraRouter with existing http.Handler middleware?

Yes! MoraRouter provides adapters to use standard `http.Handler` and `http.HandlerFunc` middleware:

```go
// Convert standard middleware
stdMiddleware := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Middleware logic...
        next.ServeHTTP(w, r)
    })
}

// Use with MoraRouter
r := router.New()
r.Use(router.WrapHTTPMiddleware(stdMiddleware))
```

## Routing Questions

### How do I create optional URL parameters?

MoraRouter supports optional parameters using the `?` suffix:

```go
// Optional "format" parameter
r.Get("/users/:id.:format?", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    id := p["id"]
    format := p["format"] // Will be empty if not provided
    
    if format == "" {
        // Default to JSON
        format = "json"
    }
    
    // Respond in requested format
})
```

### How do I handle different HTTP methods for the same URL?

You can use the `.Match()` method to handle multiple HTTP methods with different handlers:

```go
r.Match([]string{"GET", "HEAD"}, "/users/:id", getUser)
r.Match([]string{"PUT", "PATCH"}, "/users/:id", updateUser)
```

Or use a resource controller to handle all RESTful methods:

```go
r.Resource("/users", UserController{})
```

### How can I generate URLs from my routes?

Use named routes and the URL generation functions:

```go
// Define a named route
r.Get("/users/:id", getUser).Name("user.show")

// Generate a URL
url := r.URL("user.show", map[string]string{"id": "42"})
// url = "/users/42"
```

## Middleware Questions

### What order are middleware executed in?

Middleware are executed in the order they are added, from outermost to innermost:

1. Global middleware (added with `r.Use()`)
2. Group middleware (added with `group.Use()`)
3. Route-specific middleware (added with `r.With()`)
4. The route handler itself

### How do I share data between middleware and handlers?

MoraRouter provides a context-based mechanism for sharing data:

```go
// In middleware
func AuthMiddleware(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        user := authenticateUser(r)
        // Store user in context
        ctx := router.WithValue(r.Context(), "user", user)
        // Pass updated context
        next(w, r.WithContext(ctx), p)
    }
}

// In handler
func profileHandler(w http.ResponseWriter, r *http.Request, p router.Params) {
    // Retrieve user from context
    user := router.ValueFromContext(r.Context(), "user")
    // Use user data...
}
```

### Are there built-in middleware for common tasks?

Yes, MoraRouter includes several built-in middleware:

- Logging
- Recovery (panic handling)
- CORS
- Rate limiting
- Request ID
- Timeout handling
- Authentication (Basic and JWT)
- Compression
- Cache control

## Resource and Controller Questions

### How do I implement a custom action in a resource controller?

You can add custom actions to your resource controllers:

```go
type ProductController struct {
    router.DefaultController
}

// Standard RESTful methods...

// Custom action
func (c ProductController) Archive(w http.ResponseWriter, r *http.Request, p router.Params) {
    // Archive logic...
}

// In your router setup
r.Resource("/products", ProductController{})

// Register custom action
r.Get("/products/:id/archive", ProductController{}.Archive)
```

### Can I customize the URL parameters for resources?

Yes, you can customize the resource configuration:

```go
r.Resource("/users", UserController{}, 
    router.WithResourceName("user"),
    router.WithResourceID("user_id"),
    router.WithoutResourceMethod("delete"),
)
```

## WebSocket Questions

### How do I handle binary data in WebSockets?

MoraRouter's WebSocket utilities support both text and binary messages:

```go
r.WebSocket("/binary", func(conn *router.WebSocketConnection, msg []byte) {
    // Check message type
    if conn.MessageType() == websocket.BinaryMessage {
        // Process binary data
        processImage(msg)
        
        // Send binary response
        conn.SendBinary(processedData)
    }
})
```

### How many concurrent WebSocket connections can MoraRouter handle?

The number of concurrent WebSocket connections depends on your server's resources. MoraRouter itself doesn't impose strict limits, but you can configure connection limits:

```go
r := router.New(
    router.WithWebSocketConfig(router.WebSocketConfig{
        MaxConnections: 10000,
        MaxMessageSize: 32 * 1024, // 32KB
    })
)
```

In our testing, a moderately-sized server can handle thousands of concurrent connections with proper tuning.

## Performance Questions

### How can I optimize MoraRouter for high throughput?

See our [Performance](performance.md) guide for detailed optimization strategies. Some quick tips:

1. Use the route cache: `router.WithRouteCache(1000)`
2. Optimize middleware order (put frequently short-circuiting middleware first)
3. Use parameter type conversion helpers to avoid repetitive parsing
4. For static routes, consider using route pre-compilation
5. For heavily loaded production environments, consider using `router.WithOptimizationLevel(router.OptimizationProduction)`

### Does MoraRouter work well with Go's concurrency model?

Yes! MoraRouter is designed to work seamlessly with Go's concurrency. The router itself is thread-safe, and you can use goroutines within your handlers as needed:

```go
r.Get("/async", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    // Create a channel for results
    resultCh := make(chan string)
    
    // Spawn a goroutine
    go func() {
        // Do some async work
        time.Sleep(100 * time.Millisecond)
        resultCh <- "Async result!"
    }()
    
    // Wait for result
    result := <-resultCh
    
    // Send response
    router.JSON(w, http.StatusOK, map[string]string{
        "message": result,
    })
})
```

## Testing Questions

### How do I test my routes and handlers?

MoraRouter provides testing utilities to simplify API testing:

```go
func TestUserAPI(t *testing.T) {
    // Create router with test configuration
    r := setupRouter()
    
    // Create test client
    client := router.NewTestClient(r)
    
    // Test GET request
    resp := client.Get("/users/42")
    
    // Verify response
    if !resp.IsOK() {
        t.Errorf("Expected 200 status, got %d", resp.Status())
    }
    
    // Parse JSON response
    var user map[string]interface{}
    resp.JSON(&user)
    
    if user["id"] != "42" {
        t.Errorf("Expected user ID 42, got %v", user["id"])
    }
}
```

### How can I mock authentication in tests?

The test client allows you to set default headers for all requests:

```go
// Create test client with auth token
client := router.NewTestClient(r)
client.WithHeader("Authorization", "Bearer test-token")

// All requests will include the Authorization header
resp := client.Get("/protected-resource")
```

## Deployment Questions

### What's the best way to deploy a MoraRouter application?

MoraRouter applications are standard Go HTTP applications and can be deployed like any Go web service:

1. Build for your target platform: `GOOS=linux GOARCH=amd64 go build -o server main.go`
2. Transfer the binary to your server
3. Run the binary, optionally with a process manager like systemd or supervisor

For containerized deployment, a simple Dockerfile works well:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/server /app/
EXPOSE 8080
CMD ["./server"]
```

### How do I handle graceful shutdown?

MoraRouter supports graceful shutdown to ensure in-flight requests complete:

```go
func main() {
    r := router.New()
    // Setup routes...
    
    srv := &http.Server{
        Addr:    ":8080",
        Handler: r,
    }
    
    // Start server in goroutine
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()
    
    // Setup signal handling
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    // Shutdown with 5s timeout for in-flight requests
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    log.Println("Shutting down server...")
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced shutdown: %v", err)
    }
    
    log.Println("Server gracefully stopped")
}
```

## Troubleshooting

### My route isn't matching. What could be wrong?

Common issues with route matching:

1. **Order matters**: More specific routes should come before wildcard routes
2. **Leading/trailing slashes**: Check if you're handling these consistently
3. **Parameter syntax**: Ensure you're using the correct syntax (`:param` vs `{param}`)
4. **HTTP method**: Verify you're registering and calling the correct HTTP method

You can debug route matching using the inspector:

```go
r := router.New(router.WithDebug())
// Access route inspector at /_mora/inspector
```

### Why am I getting "handler not found" errors with hot reload?

When using hot reload, ensure that:

1. Your handler registration matches the names in your config file
2. You've registered all required handlers and middleware
3. The package paths are correctly specified

If problems persist, enable debug logging:

```go
r := router.New(
    router.WithHotReload("routes.json", 5*time.Second),
    router.WithDebugLog(),
)
```

### How do I troubleshoot middleware issues?

For middleware debugging:

1. Add logging at the beginning and end of each middleware
2. Use middleware-specific debug flags: `router.WithMiddlewareDebug()`
3. Check middleware execution order with the debug inspector

## Extension and Contribution

### How can I create custom extensions for MoraRouter?

MoraRouter is designed to be extensible. You can create custom:

1. **Middleware**: Implement the `router.MiddlewareFunc` type
2. **Renderers**: Extend `router.Renderer` interface
3. **Validators**: Implement custom validation functions
4. **Parameter matchers**: Create custom parameter matching patterns

See our [Contributing Guide](contributing.md) for more details.

### I found a bug or have a feature request. What should I do?

Please open an issue on our GitHub repository with:

1. A clear description of the bug or feature
2. Steps to reproduce (for bugs)
3. Expected vs. actual behavior (for bugs)
4. Use case and benefits (for features)

We welcome pull requests for bugfixes and new features!

While there are several excellent routers in the Go ecosystem (such as gorilla/mux, httprouter, chi, and gin), MoraRouter aims to provide a comprehensive set of features without sacrificing performance:

- **Feature Rich**: Built-in support for advanced routing patterns, middleware, validation, templates, and more
- **Developer Experience**: Designed to reduce boilerplate and make common tasks simple
- **Performance**: Optimized path matching and minimal allocations for high performance
- **No External Dependencies**: Core functionality has minimal dependencies
- **Flexible API**: Easy to extend and customize

### Why was MoraRouter created?

MoraRouter was created to combine the best aspects of different router approaches:

1. The expressiveness and flexibility of Django's URL routing
2. The performance of trie-based routers like httprouter
3. The middleware approach of libraries like negroni
4. The resource-oriented design of frameworks like Laravel
5. The type safety and concurrency model of Go

The result is a router that makes it easy to build robust web applications in Go without unnecessary complexity.

## Installation and Setup

### How do I install MoraRouter?

```bash
go get -u github.com/yourusername/mora-router
```

### What Go version is required?

MoraRouter requires Go 1.18 or later to support generics for the data binding features.

### How do I create a basic project with MoraRouter?

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/yourusername/mora-router/router"
)

func main() {
    r := router.New(router.WithLogging())
    
    r.Get("/", func(w http.ResponseWriter, req *http.Request, p router.Params) {
        w.Write([]byte("Hello, World!"))
    })
    
    log.Println("Server started on :8080")
    http.ListenAndServe(":8080", r)
}
```

### Does MoraRouter work with standard Go HTTP handlers?

Yes, you can mount standard `http.Handler` instances with the `Mount` method:

```go
fileServer := http.FileServer(http.Dir("./public"))
r.Mount("/static", fileServer)
```

## Routing

### How do I create routes with parameters?

```go
r.Get("/users/:id", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    id := p["id"]
    // Use the id parameter
})
```

### Can I use regular expressions in route parameters?

Yes, MoraRouter supports regex validation for parameters:

```go
r.Get("/users/:id(\\d+)", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    id := p["id"] // Guaranteed to be numeric
})
```

### How do I handle different HTTP methods for the same path?

Simply register handlers for different methods on the same path:

```go
r.Get("/users/:id", getUserHandler)
r.Put("/users/:id", updateUserHandler)
r.Delete("/users/:id", deleteUserHandler)
```

### How can I generate URLs from routes?

Use named routes and the `URL` method:

```go
r.Get("/users/:id", getUserHandler).Name("user.show")

// Generate URL
url, _ := r.URL("user.show", "123") // "/users/123"
```

## Middleware

### How do I use middleware in MoraRouter?

There are several ways to apply middleware:

```go
// Global middleware (applied to all routes)
r := router.New(router.WithLogging())
r.Use(CustomMiddleware)

// Route-specific middleware
r.With(AuthMiddleware).Get("/admin", adminHandler)

// Group middleware
admin := r.Group("/admin")
admin.Use(AuthMiddleware)
```

### How do I create custom middleware?

```go
func CustomMiddleware(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        // Do something before the handler
        
        next(w, r, p) // Call the next handler
        
        // Do something after the handler
    }
}
```

### Can I control the order of middleware execution?

Yes, middleware is executed in the order it's added:

```go
r.Use(FirstMiddleware)
r.Use(SecondMiddleware)
// FirstMiddleware runs before SecondMiddleware
```

### How do I share data between middleware and handlers?

Use Go's `context.Context`:

```go
func AuthMiddleware(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        user := getCurrentUser(r)
        ctx := context.WithValue(r.Context(), "user", user)
        next(w, r.WithContext(ctx), p)
    }
}

func profileHandler(w http.ResponseWriter, r *http.Request, p router.Params) {
    user := r.Context().Value("user").(User)
    // Use the user
}
```

## Data Binding and Validation

### How do I parse and validate JSON requests?

```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=3"`
    Email string `json:"email" validate:"required,email"`
}

r.Post("/users", router.BindJSON(func(w http.ResponseWriter, r *http.Request, p router.Params, req CreateUserRequest) {
    // req is already parsed and validated
}))
```

### What validation rules are supported?

MoraRouter supports many validation rules, including:
- `required`: Field must not be empty
- `min=n`, `max=n`: Minimum/maximum length for strings, min/max value for numbers
- `email`: Must be a valid email
- `url`: Must be a valid URL
- `oneof=a b c`: Must be one of the provided values
- `gt=n`, `lt=n`: Greater/less than for numbers
- And many more

### How do I handle file uploads?

```go
type UploadRequest struct {
    Title string           `form:"title" validate:"required"`
    File  *router.FormFile `form:"file" validate:"required"`
}

r.Post("/upload", router.BindForm(func(w http.ResponseWriter, r *http.Request, p router.Params, form router.Form, req UploadRequest) {
    if form.HasErrors() {
        router.JSON(w, http.StatusBadRequest, form.GetErrors())
        return
    }
    
    filePath, err := form.SaveFile("file", "./uploads")
    if err != nil {
        router.Error(w, http.StatusInternalServerError, "Failed to save file")
        return
    }
    
    router.JSON(w, http.StatusCreated, map[string]string{
        "message": "File uploaded successfully",
        "path": filePath,
    })
}))
```

## Resource Controllers

### What is a resource controller?

A resource controller is a struct that implements handler methods for standard CRUD operations (Index, Show, Create, Update, Delete) on a resource.

### How do I create a resource controller?

```go
type UserController struct {
    router.DefaultController
    service UserService
}

// Override only the methods you need
func (c UserController) Index(w http.ResponseWriter, r *http.Request, p router.Params) {
    users, _ := c.service.ListUsers()
    router.JSON(w, http.StatusOK, users)
}

func (c UserController) Show(w http.ResponseWriter, r *http.Request, p router.Params) {
    user, _ := c.service.GetUser(p["id"])
    router.JSON(w, http.StatusOK, user)
}

// Register the controller
r.Resource("/users", UserController{service: userService})
```

### Can I customize the routes created by resources?

Yes, you can override specific routes or add custom routes to resources:

```go
// Register the resource
r.Resource("/users", UserController{})

// Add custom route
r.Post("/users/:id/reset-password", userController.ResetPassword)
```

## Performance

### Is MoraRouter performant?

Yes, MoraRouter is designed with performance in mind. While it offers more features than minimalist routers, it maintains excellent performance characteristics:

- Efficient route matching algorithm
- Minimal allocations during request handling
- Optional response caching
- Ability to pre-compile templates

### How can I improve the performance of my MoraRouter application?

1. Use `router.WithCache()` for GET requests that don't change frequently
2. Pre-compile templates at startup
3. Use the built-in response helpers that set proper Content-Type headers
4. Enable gzip compression for responses
5. Consider using server-side rendering instead of large JSON payloads

### Does MoraRouter support HTTP/2?

MoraRouter works with Go's standard `http.Server`, which supports HTTP/2 when configured with TLS:

```go
server := &http.Server{
    Addr:    ":8443",
    Handler: r,
}
server.ListenAndServeTLS("cert.pem", "key.pem")
```

## Testing

### How do I test routes with MoraRouter?

MoraRouter includes a test client that makes it easy to test your routes:

```go
func TestUserAPI(t *testing.T) {
    r := router.New()
    r.Get("/users/:id", getUserHandler)
    
    client := router.NewTestClient(r)
    
    resp := client.Get("/users/123")
    if !resp.IsOK() {
        t.Error("Expected 200 OK")
    }
    
    var user User
    resp.DecodeJSON(&user)
    
    if user.ID != "123" {
        t.Errorf("Expected user ID 123, got %s", user.ID)
    }
}
```

### How do I mock dependencies in tests?

You can use interfaces and dependency injection to make your handlers testable:

```go
// Interface for the service
type UserService interface {
    GetUser(id string) (User, error)
}

// Handler that uses the service
func getUserHandler(service UserService) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        user, err := service.GetUser(p["id"])
        if err != nil {
            router.Error(w, http.StatusNotFound, "User not found")
            return
        }
        router.JSON(w, http.StatusOK, user)
    }
}

// Mock implementation for testing
type MockUserService struct {
    GetUserFunc func(id string) (User, error)
}

func (m MockUserService) GetUser(id string) (User, error) {
    return m.GetUserFunc(id)
}

// Test with mock
func TestGetUserHandler(t *testing.T) {
    mock := MockUserService{
        GetUserFunc: func(id string) (User, error) {
            return User{ID: id, Name: "Test User"}, nil
        },
    }
    
    r := router.New()
    r.Get("/users/:id", getUserHandler(mock))
    
    client := router.NewTestClient(r)
    resp := client.Get("/users/123")
    
    // Assert response
}
```

## Deployment

### How do I deploy a MoraRouter application?

MoraRouter applications can be deployed like any Go HTTP server:

1. Build your application: `go build -o myapp`
2. Run it on your server: `./myapp`

For production, consider:
- Using a process manager like systemd
- Setting up a reverse proxy (Nginx/Caddy/Traefik)
- Using environment variables for configuration
- Implementing graceful shutdown

### How do I implement graceful shutdown?

```go
func main() {
    r := router.New()
    // Configure router
    
    server := &http.Server{
        Addr:    ":8080",
        Handler: r,
    }
    
    // Start server in a goroutine
    go func() {
        log.Println("Starting server on :8080")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()
    
    // Wait for interrupt signal
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop
    
    // Gracefully shutdown
    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("Server shutdown error: %v", err)
    }
    
    log.Println("Server gracefully stopped")
}
```

## Templates and UI

### How do I use HTML templates with MoraRouter?

```go
// Enable templates
r := router.New(router.WithTemplates("views"))

// Render a template
r.Get("/", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    data := map[string]interface{}{
        "Title": "Welcome",
    }
    router.RenderTemplate(w, r, "home.html", data)
})
```

### How do I serve a Single Page Application (SPA)?

```go
// Serve a SPA with HTML5 history API support
r.SPA("/app", "./dist", "index.html")
```

### Can I use MoraRouter with frontend frameworks like React or Vue?

Yes, there are several approaches:

1. Use `r.SPA()` to serve the built frontend files
2. Create a separate API with MoraRouter and serve the frontend separately
3. Use server-side rendering with Go templates for initial load and JavaScript for interactivity

## Error Handling

### How do I handle errors in MoraRouter?

```go
r.Get("/users/:id", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    user, err := fetchUser(p["id"])
    
    if err == ErrNotFound {
        router.Error(w, http.StatusNotFound, "User not found")
        return
    }
    
    if err != nil {
        router.Error(w, http.StatusInternalServerError, "Failed to fetch user")
        return
    }
    
    router.JSON(w, http.StatusOK, user)
})
```

### How do I customize error responses?

```go
// Custom JSON error response
type ErrorResponse struct {
    Status  int    `json:"status"`
    Message string `json:"message"`
    Code    string `json:"code,omitempty"`
}

func CustomError(w http.ResponseWriter, status int, message string, code string) {
    router.JSON(w, status, ErrorResponse{
        Status:  status,
        Message: message,
        Code:    code,
    })
}

// Use in handlers
r.Get("/protected", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    if !isAuthorized(req) {
        CustomError(w, http.StatusForbidden, "Access denied", "FORBIDDEN")
        return
    }
    // ...
})
```

## Advanced Features

### How do I implement API versioning?

```go
// Option 1: URL-based versioning
v1 := r.Group("/api/v1")
v1.Get("/users", v1ListUsersHandler)

v2 := r.Group("/api/v2")
v2.Get("/users", v2ListUsersHandler)

// Option 2: Header-based versioning
r := router.New(router.WithAPIVersioning("X-API-Version", "1"))
// Routes will be matched based on the header value
```

### How do I implement WebSockets?

```go
r := router.New(router.WithGorillaWebSocket())

r.Get("/ws", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    conn, err := router.UpgradeWebSocket(w, req)
    if err != nil {
        return
    }
    defer conn.Close()
    
    // Handle WebSocket connection
    for {
        messageType, message, err := conn.ReadMessage()
        if err != nil {
            break
        }
        conn.WriteMessage(messageType, message)
    }
})
```

### How do I use internationalization (i18n) with routes?

```go
translations := map[string]map[string]string{
    "es": {
        "/users": "/usuarios",
        "/products": "/productos",
    },
}

r := router.New(router.WithI18n(translations))

// Define a route once
r.Get("/users", listUsersHandler)
// It will also respond to /usuarios when Accept-Language is es
```

## Troubleshooting

### My route isn't matching as expected

Check these common issues:

1. **Path Format**: Ensure your route pattern starts with a slash (`/`)
2. **Parameter Names**: Parameter names must be unique in a route
3. **Regex Patterns**: Make sure to escape backslashes in regex patterns (`\\d+` not `\d+`)
4. **Order Matters**: More specific routes should be defined before wildcards
5. **HTTP Method**: Verify you're using the correct HTTP method

### Middleware isn't executing as expected

Check the middleware order and ensure you're calling `next(w, r, p)` to continue the chain:

```go
func MyMiddleware(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        // Do something
        
        // IMPORTANT: Call the next handler in the chain
        next(w, r, p)
        
        // After handler code runs here
    }
}
```

### I'm getting a 404 Not Found for a route I defined

Possible causes:

1. Route defined after server started
2. Route pattern doesn't match the request path exactly
3. Group prefix issues (double slashes or missing slashes)
4. The handler is mounted under a different prefix

Try debugging with the route inspector:

```go
r := router.New(router.WithDebug())
// Then visit /_mora/routes in your browser
```

### How do I debug performance issues?

1. Enable the metrics middleware: `router.WithMetrics()`
2. Visit the `/_mora/metrics` endpoint to see request counts and latencies
3. Use Go's built-in profiling tools (pprof) for deeper analysis
4. Consider adding custom timing middleware to isolate slow components

## Integration

### Can I use MoraRouter with other Go web frameworks?

Yes, MoraRouter implements the `http.Handler` interface, so it can be:

1. Used as the main handler in a Go HTTP server
2. Mounted inside other routers or frameworks
3. Wrapped by other middleware or handlers

### How do I integrate with a database?

The recommended approach is to:

1. Create a service layer that handles database operations
2. Inject services into your handlers or controllers
3. Use a dependency injection pattern for testability

```go
type UserService struct {
    db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
    return &UserService{db: db}
}

func (s *UserService) GetUser(id string) (User, error) {
    // Database operations
}

func getUserHandler(service *UserService) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        user, err := service.GetUser(p["id"])
        // Handle response
    }
}

// In main.go
db, _ := sql.Open("postgres", "connection_string")
userService := NewUserService(db)
r.Get("/users/:id", getUserHandler(userService))
```

## Contributing

### How can I contribute to MoraRouter?

1. Report issues on GitHub
2. Suggest features or improvements
3. Submit pull requests
4. Improve documentation
5. Share examples and best practices
6. Write blog posts or tutorials

See [Contributing](contributing.md) for more details.

## Miscellaneous

### Is MoraRouter production-ready?

Yes, MoraRouter is designed for production use with features like logging, recovery from panics, and comprehensive testing utilities.

### Does MoraRouter support SSL/TLS?

MoraRouter works with Go's standard `http.Server`, which supports TLS:

```go
server := &http.Server{
    Addr:    ":8443",
    Handler: r,
}
server.ListenAndServeTLS("cert.pem", "key.pem")
```

### Can I use MoraRouter with Docker?

Yes, here's a simple Dockerfile:

```dockerfile
FROM golang:1.18-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY views/ /app/views/
EXPOSE 8080
CMD ["./main"]
```

## Further Help

If your question isn't answered here, check:

1. The [Documentation](index.md) for detailed guides
2. The [Examples](examples.md) for real-world usage
3. The [API Reference](api-reference.md) for complete API details
4. Open an issue on GitHub for specific questions
