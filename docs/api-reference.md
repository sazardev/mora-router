# API Reference

This document provides a comprehensive reference for the MoraRouter API, including all major types, functions, and methods.

## Core Types

### Router

The central type that manages HTTP routing in your application.

```go
type Router struct {
    // Contains all registered routes
    Routes []*Route
    
    // Global middleware stack
    Middleware []MiddlewareFunc
    
    // Options for router behavior
    Options *RouterOptions
    
    // ... other fields
}
```

### Route

Represents an individual route in your application.

```go
type Route struct {
    // HTTP method this route responds to
    Method string
    
    // URL pattern for matching requests
    Pattern string
    
    // Handler function for this route
    Handler HandlerFunc
    
    // Route-specific middleware stack
    Middleware []MiddlewareFunc
    
    // Name for this route (used in URL generation)
    Name string
    
    // ... other fields
}
```

### Params

Contains route parameters extracted from the URL.

```go
type Params map[string]string
```

### HandlerFunc

The signature for route handler functions.

```go
type HandlerFunc func(http.ResponseWriter, *http.Request, Params)
```

### MiddlewareFunc

The signature for middleware functions.

```go
type MiddlewareFunc func(HandlerFunc) HandlerFunc
```

### Group

A group of routes sharing a common prefix and middleware.

```go
type Group struct {
    // Prefix shared by all routes in this group
    Prefix string
    
    // Group-specific middleware stack
    Middleware []MiddlewareFunc
    
    // Parent router reference
    Router *Router
    
    // ... other fields
}
```

## Router Creation and Configuration

### Creating a New Router

```go
// Create a new router with default options
r := router.New()

// Create with configuration options
r := router.New(
    router.WithLogging(),
    router.WithRecovery(),
    router.WithCORS("*"),
    // ... other options
)
```

### Configuration Options

```go
// Enable request logging
router.WithLogging()

// Enable panic recovery
router.WithRecovery()

// Configure CORS
router.WithCORS(origins string, options ...CORSOption)

// Enable Swagger/OpenAPI documentation
router.WithSwagger()

// Enable debug features
router.WithDebug()

// Enable metrics collection
router.WithMetrics()

// Configure response caching
router.WithCache(duration time.Duration)

// Enable JWT authentication
router.WithJWT(secret string, options ...JWTOption)

// Enable WebSocket support
router.WithGorillaWebSocket()

// Set custom error handler
router.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
    // Custom error handling
})

// Set custom not found handler
router.WithNotFoundHandler(func(w http.ResponseWriter, r *http.Request) {
    // Custom 404 handling
})

// Configure hot reload
router.WithHotReload(configPath string, interval time.Duration)

// Create a router with options
r := router.New(
    router.WithLogging(),
    router.WithRecovery(),
    router.WithCORS("*"),
)
```

### Route Registration

```go
// Basic HTTP methods
r.Get(pattern string, handler HandlerFunc)
r.Post(pattern string, handler HandlerFunc)
r.Put(pattern string, handler HandlerFunc)
r.Delete(pattern string, handler HandlerFunc)
r.Patch(pattern string, handler HandlerFunc)
r.Head(pattern string, handler HandlerFunc)
r.Options(pattern string, handler HandlerFunc)

// Generic method
r.Handle(method string, pattern string, handler HandlerFunc)

// Method that returns the route for further configuration
route := r.Get(pattern, handler)
route.Name("route-name")  // Name the route for URL generation
```

### Middleware

```go
// Apply middleware to all routes
r.Use(middleware1, middleware2)

// Apply middleware to specific routes
r.With(middleware1, middleware2).Get(pattern, handler)
```

### Groups

```go
// Create a route group
group := r.Group("/prefix")

// Add routes to the group
group.Get("/path", handler)  // Maps to /prefix/path

// Apply middleware to a group
group.Use(middleware1, middleware2)

// Create nested groups
nestedGroup := group.Group("/nested")  // Maps to /prefix/nested
```

### Resources

```go
// Register a resource controller
r.Resource("/users", UserController{})

// This creates:
// GET /users -> Index()
// GET /users/:id -> Show()
// POST /users -> Create()
// PUT /users/:id -> Update()
// DELETE /users/:id -> Delete()
```

### Static Files

```go
// Serve static files from a directory
r.Static(prefix string, dir string)

// Serve a single file
r.StaticFile(pattern string, file string)

// Serve a SPA with HTML5 history API support
r.SPA(prefix string, dir string, indexFile string)
```

### URL Generation

```go
// Name a route
r.Get("/users/:id", handler).Name("user.show")

// Generate a URL from a named route
url, err := r.URL(name string, params ...string)
```

### Special Handlers

```go
// Set custom 404 handler
r.NotFound(handler HandlerFunc)

// Set custom 405 handler
r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request, p Params, methods []string) {
    // methods contains the allowed methods for this path
})
```

### Mounting

```go
// Mount an http.Handler under a prefix
r.Mount(prefix string, handler http.Handler)
```

## Types

### HandlerFunc

```go
type HandlerFunc func(http.ResponseWriter, *http.Request, Params)
```

### Middleware

```go
type Middleware func(HandlerFunc) HandlerFunc
```

### Params

```go
type Params map[string]string
```

### Option

```go
type Option func(*MoraRouter)
```

## Resource Controller Interface

```go
type ResourceController interface {
    Index(http.ResponseWriter, *http.Request, Params)
    Show(http.ResponseWriter, *http.Request, Params)
    Create(http.ResponseWriter, *http.Request, Params)
    Update(http.ResponseWriter, *http.Request, Params)
    Delete(http.ResponseWriter, *http.Request, Params)
}
```

## Response Helpers

### JSON Responses

```go
// Send JSON response
router.JSON(w http.ResponseWriter, status int, data interface{})

// Send pretty-printed JSON
router.PrettyJSON(w http.ResponseWriter, status int, data interface{})

// Send JSONP response
router.JSONP(w http.ResponseWriter, status int, callback string, data interface{})
```

### XML Responses

```go
// Send XML response
router.XML(w http.ResponseWriter, status int, data interface{})
```

### Plain Text and HTML

```go
// Send plain text
router.Text(w http.ResponseWriter, status int, text string)

// Send HTML
router.HTML(w http.ResponseWriter, status int, html string)
```

### Error Responses

```go
// Send error response
router.Error(w http.ResponseWriter, status int, message string)

// Common error helpers
router.BadRequest(w http.ResponseWriter, message string)
router.Unauthorized(w http.ResponseWriter, realm string)
router.Forbidden(w http.ResponseWriter, message string)
router.NotFound(w http.ResponseWriter, message string)
router.MethodNotAllowed(w http.ResponseWriter, message string, allowedMethods []string)
router.ServerError(w http.ResponseWriter, message string)
```

### Redirects

```go
// Redirect to another URL
router.Redirect(w http.ResponseWriter, r *http.Request, url string, status int)
```

### File Responses

```go
// Serve a file for download
router.FileDownload(w http.ResponseWriter, r *http.Request, filepath string)

// Force browser to download rather than display
router.ForceDownload(w http.ResponseWriter, filename string)
```

### Templates

```go
// Render a template with data
router.RenderTemplate(w http.ResponseWriter, r *http.Request, name string, data interface{})
```

## Data Binding and Validation

### JSON Binding

```go
// Bind and validate JSON
router.BindJSON[T any](func(w http.ResponseWriter, r *http.Request, p Params, input T) {
    // input is parsed and validated
})

// Parse JSON manually
err := router.ParseJSON(r *http.Request, v interface{})
```

### XML Binding

```go
// Bind and validate XML
router.BindXML[T any](func(w http.ResponseWriter, r *http.Request, p Params, input T) {
    // input is parsed and validated
})

// Parse XML manually
err := router.ParseXML(r *http.Request, v interface{})
```

### Form Binding

```go
// Bind and validate form data
router.BindForm[T any](func(w http.ResponseWriter, r *http.Request, p Params, form router.Form, input T) {
    // input is parsed and validated
})

// Parse form data manually
form, err := router.ParseForm(r *http.Request)
```

### Validation

```go
// Validate an object
err := router.Validate(v interface{})

// Get detailed validation errors
errs := router.ValidateDetailed(v interface{})
```

## Testing

### Test Client

```go
// Create a test client
client := router.NewTestClient(r *MoraRouter)

// Make requests
resp := client.Get(path string)
resp := client.Post(path string, contentType string, body io.Reader)
resp := client.Put(path string, contentType string, body io.Reader)
resp := client.Delete(path string)

// JSON requests
resp := client.PostJSON(path string, data interface{})
resp := client.PutJSON(path string, data interface{})
resp := client.PatchJSON(path string, data interface{})

// Form requests
resp := client.PostForm(path string, values map[string]string)
```

### Test Response

```go
// Check status
if resp.IsOK() { /* status is 200 */ }
if resp.IsCreated() { /* status is 201 */ }
if resp.IsNoContent() { /* status is 204 */ }
if resp.IsBadRequest() { /* status is 400 */ }
if resp.IsNotFound() { /* status is 404 */ }

// Get body
body := resp.Body()
text := resp.Text()

// Parse body
var data map[string]interface{}
resp.DecodeJSON(&data)

var xmlData MyXMLStruct
resp.DecodeXML(&xmlData)
```

## Options

### Logging and Recovery

```go
// Enable request logging
router.WithLogging()

// Enable panic recovery
router.WithRecovery()
```

### CORS

```go
// Enable CORS with allowed origins
router.WithCORS(allowedOrigins string)
```

### Rate Limiting

```go
// Enable rate limiting
router.WithRateLimit(max int, window time.Duration)
```

### Cache

```go
// Enable response caching
router.WithCache(ttl time.Duration)
```

### API Versioning

```go
// Enable API versioning
router.WithAPIVersioning(headerName string, defaultVersion string)
```

### JWT Authentication

```go
// Enable JWT authentication
router.WithJWT(secret string)
```

### Templates

```go
// Enable template rendering
router.WithTemplates(dir string)

// Configure template manager
router.WithTemplateManager(tm *TemplateManager)
```

### Metrics

```go
// Enable metrics collection
router.WithMetrics()
```

### WebSockets

```go
// Enable WebSocket support
router.WithGorillaWebSocket()

// Create a chat room
router.WithChatRoom(path string)
```

### OpenAPI/Swagger

```go
// Enable OpenAPI documentation
router.WithSwagger()
```

### Internationalization

```go
// Enable route translation
router.WithI18n(translations map[string]map[string]string)
```

### Hot Reload

```go
// Enable hot reload of routes
router.WithHotReload(filePath string, interval time.Duration)
```

## Middleware Registry

```go
// Register middleware by name
r.RegisterMiddleware(name string, middleware Middleware)

// Use middleware from registry
r.UseMiddleware(names ...string)

// Apply named middleware to a route
r.WithNamed(names ...string).Get(pattern, handler)
```

## Macros

```go
// Register a route macro
router.RegisterMacro(name string, pattern string, methods []string, middlewares ...Middleware)

// Use a registered macro
r.UseMacro(prefix string, name string, handler HandlerFunc)
```

## Template Management

```go
// Create a template manager
tm := router.NewTemplateManager(dir string)

// Configure the manager
tm.WithLayout(layoutFile string)
tm.WithPartials(partialFiles ...string)
tm.WithFuncs(funcs template.FuncMap)
tm.WithExtension(ext string)

// Parse templates
err := tm.Parse()

// Render a template
err := tm.Render(w io.Writer, name string, data interface{})
```

## Form Handling

```go
// Create a form from a request
form, err := router.NewForm(r *http.Request)

// Check if form has errors
if form.HasErrors() {
    errors := form.GetErrors()
}

// Get a form value
value := form.Get(name string)

// Get a file
file := form.File(name string)

// Save a file
path, err := form.SaveFile(name string, dir string)
```

## Code Generation

```go
// Create a route generator
gen := router.NewRouteGenerator(r *MoraRouter)

// Generate a controller
code, err := gen.GenerateController(name string)

// Generate a model
code, err := gen.GenerateModel(name string, fields map[string]string)

// Generate tests
code, err := gen.GenerateTests(name string, endpoints []string)
```

## WebSocket

```go
// Upgrade a connection to WebSocket
conn, err := router.UpgradeWebSocket(w http.ResponseWriter, r *http.Request)

// Send a message
err := conn.WriteMessage(messageType int, data []byte)

// Read a message
messageType, p, err := conn.ReadMessage()
```

## Context Helpers

```go
// Get a parameter from the request context
value := router.Param(r *http.Request, name string)

// Get JWT claims from context
claims := router.GetClaims(r *http.Request)

// Get current user from context (requires auth middleware)
user := router.GetUser(r *http.Request)
```

## Content Negotiation

```go
// Create a renderer
render := router.NewRender()

// Negotiate content type based on Accept header
render.Negotiate(w http.ResponseWriter, r *http.Request, status int, data interface{})
```

## URL Helpers

```go
// Join URL paths
path := router.JoinPath(segments ...string)

// Parse query parameters
query := router.ParseQuery(r *http.Request)

// Build a query string
queryString := router.BuildQuery(params map[string]string)
```

## Utility Functions

```go
// Generate a random string
str := router.RandomString(length int)

// Hash a password
hash := router.HashPassword(password string)

// Verify a password
valid := router.VerifyPassword(password string, hash string)

// Format a timestamp
formatted := router.FormatTime(t time.Time, format string)
```

## Conclusion

This API reference covers the main functionality of MoraRouter. For more detailed information on specific features, refer to the topic-specific guides.

For usage examples, check the [Examples](examples.md) section.
