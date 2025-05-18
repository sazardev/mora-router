# Routing with MoraRouter

Routing is at the heart of MoraRouter, with powerful pattern matching capabilities that go far beyond basic path parameters.

## Basic Routing

Setting up basic routes in MoraRouter is straightforward:

```go
r := router.New()

// Basic routes for different HTTP methods
r.Get("/users", listUsersHandler)
r.Post("/users", createUserHandler)
r.Put("/users/:id", updateUserHandler)
r.Delete("/users/:id", deleteUserHandler)

// Generic method
r.Handle("PATCH", "/users/:id/partial", patchUserHandler)
```

## Route Parameters

Routes can include named parameters that capture values from segments of the URL:

```go
// Basic parameter
r.Get("/users/:id", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    id := p["id"]  // Access the parameter value
    // ...
})

// Multiple parameters
r.Get("/posts/:year/:month/:slug", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    year := p["year"]
    month := p["month"]
    slug := p["slug"]
    // ...
})
```

## Advanced Parameter Types

MoraRouter supports various parameter validation patterns:

### Regular Expression Validation

```go
// Only match if id is numeric
r.Get("/users/:id(\\d+)", userHandler)

// Match year as 4 digits
r.Get("/archive/:year(\\d{4})", archiveHandler)

// Match product code with format XXX-999
r.Get("/products/:code([A-Z]{3}-\\d{3})", productHandler)
```

### Alternative Syntax

```go
// Alternative syntax with {}
r.Get("/users/{id}", userHandler)
r.Get("/products/{code:[A-Z]{3}-\\d{3}}", productHandler)
```

### Wildcard Parameters

Capture the rest of the path with a wildcard parameter:

```go
// Match any path under /files/
r.Get("/files/*filepath", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    filePath := p["filepath"]  // Contains the entire path after "/files/"
    // ...
})
```

## Named Routes

Name your routes for easier URL generation:

```go
// Name a route
r.Get("/users/:id", userHandler).Name("user.show")

// Generate a URL from a named route
url, err := r.URL("user.show", "42")
// url is "/users/42"
```

## HTTP Method Handling

MoraRouter automatically handles OPTIONS requests and provides a 405 Method Not Allowed response for unallowed methods:

```go
// Define specific methods for the same path
r.Get("/api/resource", getResourceHandler)
r.Post("/api/resource", createResourceHandler)
r.Put("/api/resource", updateResourceHandler)

// OPTIONS /api/resource would return:
// Allow: GET, POST, PUT
```

## Route Groups

Organize related routes under a common prefix:

```go
// Create a group with prefix
admin := r.Group("/admin")

// Add routes to the group
admin.Get("/dashboard", dashboardHandler)
admin.Get("/users", adminListUsersHandler)
admin.Post("/users", adminCreateUserHandler)

// Nested groups
api := r.Group("/api")
v1 := api.Group("/v1")  // /api/v1
v2 := api.Group("/v2")  // /api/v2
```

## Middleware on Routes

Apply middleware to specific routes:

```go
// Apply middleware to a single route
r.With(authMiddleware).Get("/protected", protectedHandler)

// Apply multiple middleware to a route
r.With(authMiddleware, loggingMiddleware).Get("/admin", adminHandler)

// Apply middleware to a group
admin := r.Group("/admin")
admin.Use(authMiddleware)
admin.Get("/dashboard", dashboardHandler)
```

## Mount External Handlers

Mount any `http.Handler` under a prefix:

```go
// Mount a standard http.Handler
fileServer := http.FileServer(http.Dir("./public"))
r.Mount("/static", fileServer)

// Mount a third-party handler
r.Mount("/metrics", promhttp.Handler())
```

## Static Files and SPAs

Serve static files or single-page applications:

```go
// Serve static files
r.Static("/assets", "./public/assets")

// Serve a SPA with HTML5 history API support
r.SPA("/app", "./web/dist", "index.html")
```

## Route Macros

Define reusable route patterns:

```go
// Register a custom macro
router.RegisterMacro("paginated", "/:page(\\d+)", []string{"GET"}, rateLimitMiddleware(100, time.Minute))

// Use the macro
r.UseMacro("/users", "paginated", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    page := p["page"]
    // ...
})

// UseMacro also works with API resources
r.UseMacro("/products", "list", listProductsHandler)
r.UseMacro("/products", "create", createProductHandler)
```

## Internationalized Routes

Set up routes that respond to the user's language:

```go
translations := map[string]map[string]string{
    "es": {
        "/users": "/usuarios",
        "/products": "/productos",
    },
    "fr": {
        "/users": "/utilisateurs",
        "/products": "/produits",
    },
}

r := router.New(router.WithI18n(translations))

// Define a route once
r.Get("/users", listUsersHandler)

// It will also respond to /usuarios (Spanish) and /utilisateurs (French)
// based on the Accept-Language header
```

## API Versioning

MoraRouter supports automatic API versioning:

```go
r := router.New(router.WithAPIVersioning("X-API-Version", "1"))

// Define routes without version prefix
r.Get("/users", v1ListUsersHandler)

// Define v2 routes
r.Get("/v2/users", v2ListUsersHandler)

// With X-API-Version: 2 header, /users will route to v2ListUsersHandler
```

## WebSockets

Handle WebSocket connections:

```go
r := router.New(router.WithGorillaWebSocket())

r.Get("/ws/chat", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    // Handle WebSocket connection
    conn, err := upgrader.Upgrade(w, req, nil)
    if err != nil {
        return
    }
    defer conn.Close()
    
    // Handle WebSocket messages
    for {
        messageType, message, err := conn.ReadMessage()
        if err != nil {
            break
        }
        conn.WriteMessage(messageType, message)
    }
})
```

## Hot Reload

Configure routes to reload automatically:

```go
r := router.New(router.WithHotReload("routes.json", 5 * time.Second))
```

## Next Steps

- Check [Middleware](middleware.md) to learn about extending your routes
- See [Controllers](controllers.md) for organizing your handlers
- Explore [Data Binding](data-binding.md) for request validation
