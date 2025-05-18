# Core Concepts in MoraRouter

This guide covers the fundamental concepts behind MoraRouter, helping you understand how the router works and how to use it effectively.

## Router Architecture

MoraRouter is designed with a flexible, layered architecture:

1. **Route Matching**: The core routing system that matches HTTP requests to handlers
2. **Middleware Pipeline**: A chain of functions that process requests before they reach handlers
3. **Parameter Extraction**: A system for extracting and validating route parameters
4. **Response Helpers**: Utilities for sending responses in various formats
5. **Extensions**: Optional components like validation, template rendering, and WebSockets

## Route Matching

At its core, MoraRouter is responsible for matching HTTP requests to the appropriate handlers. When a request comes in, the router:

1. Extracts the HTTP method and path from the request
2. Compares the path against registered routes
3. If a match is found, extracts any parameters
4. If no match is found, responds with a 404 Not Found
5. If the path matches but the method doesn't, responds with a 405 Method Not Allowed

### Route Patterns

MoraRouter supports several types of route patterns:

- **Static Routes**: Exact path matches (`/users`)
- **Parameter Routes**: Routes with named parameters (`/users/:id`)
- **Regex Routes**: Parameters with regex validation (`/users/:id(\\d+)`)
- **Wildcard Routes**: Routes that capture the rest of the path (`/files/*path`)
- **Mixed Routes**: Combinations of the above (`/users/:id/posts/:postId(\\d+)`)

### Route Registration

Routes are registered with HTTP method functions:

```go
// Basic routes
r.Get("/users", listUsersHandler)
r.Post("/users", createUserHandler)
r.Put("/users/:id", updateUserHandler)
r.Delete("/users/:id", deleteUserHandler)

// Generic method
r.Handle("PATCH", "/users/:id/partial", partialUpdateHandler)
```

## Handler Function Signature

All handler functions in MoraRouter follow this signature:

```go
func(http.ResponseWriter, *http.Request, router.Params)
```

- `http.ResponseWriter`: The standard Go interface for writing HTTP responses
- `*http.Request`: The standard Go HTTP request object
- `router.Params`: A map of route parameters extracted from the URL

Example handler:

```go
func getUserHandler(w http.ResponseWriter, r *http.Request, p router.Params) {
    id := p["id"] // Extract the "id" parameter from the route
    
    user, err := fetchUser(id)
    if err != nil {
        router.Error(w, http.StatusNotFound, "User not found")
        return
    }
    
    router.JSON(w, http.StatusOK, user)
}
```

## Middleware Concept

Middleware are functions that wrap handlers to add functionality. They follow this pattern:

```go
type Middleware func(HandlerFunc) HandlerFunc
```

Middleware can perform operations before and after a handler executes, modify the request or response, or even short-circuit the request pipeline.

Example middleware:

```go
func LoggingMiddleware(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        start := time.Now()
        
        // Call the next handler
        next(w, r, p)
        
        // Log after the handler returns
        log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
    }
}
```

## Middleware Pipeline

When a request comes in, it passes through a series of middleware before reaching the handler:

```
Request → Middleware 1 → Middleware 2 → ... → Handler → ... → Middleware 2 → Middleware 1 → Response
```

Each middleware can:
- Process the request before passing it to the next middleware
- Call the next middleware in the chain
- Process the response after the next middleware returns
- Short-circuit the chain by not calling the next middleware

## Options Pattern

MoraRouter uses the options pattern for configuration:

```go
r := router.New(
    router.WithLogging(),
    router.WithRecovery(),
    router.WithCORS("*"),
)
```

This approach provides a clean, extensible way to configure the router without changing its core API.

## Resource Controllers

MoraRouter supports RESTful resource controllers that implement standard CRUD operations:

```go
type ResourceController interface {
    Index(http.ResponseWriter, *http.Request, Params)  // GET /resources
    Show(http.ResponseWriter, *http.Request, Params)   // GET /resources/:id
    Create(http.ResponseWriter, *http.Request, Params) // POST /resources
    Update(http.ResponseWriter, *http.Request, Params) // PUT /resources/:id
    Delete(http.ResponseWriter, *http.Request, Params) // DELETE /resources/:id
}
```

## Route Groups

Route groups provide a way to organize routes under a common prefix and apply middleware to a set of routes:

```go
admin := r.Group("/admin")
admin.Use(AuthMiddleware)

admin.Get("/dashboard", dashboardHandler)
admin.Get("/users", adminUsersHandler)
```

## Data Binding and Validation

MoraRouter provides a robust system for binding and validating request data:

```go
r.Post("/users", router.BindJSON(func(w http.ResponseWriter, r *http.Request, p router.Params, input CreateUserRequest) {
    // input has been validated and parsed
}))
```

## Response Helpers

A set of helper functions simplify sending responses:

```go
router.JSON(w, http.StatusOK, data)
router.XML(w, http.StatusOK, data)
router.Error(w, http.StatusBadRequest, "Invalid input")
router.Redirect(w, r, "/new-location", http.StatusSeeOther)
```

## The Context Paradigm

MoraRouter uses Go's `context.Context` to store request-scoped data:

```go
// Middleware that adds a value to the context
func UserMiddleware(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        // Add user to context
        ctx := context.WithValue(r.Context(), "user", getCurrentUser(r))
        
        // Call next middleware with updated context
        next(w, r.WithContext(ctx), p)
    }
}

// Handler that reads from context
func profileHandler(w http.ResponseWriter, r *http.Request, p router.Params) {
    // Get user from context
    user := r.Context().Value("user").(User)
    
    router.JSON(w, http.StatusOK, user)
}
```

## Named Routes and URL Generation

MoraRouter supports named routes for URL generation:

```go
// Name a route
r.Get("/users/:id", userHandler).Name("user.show")

// Generate a URL
url, err := r.URL("user.show", "123")
// url is "/users/123"
```

## Not Found and Method Not Allowed

MoraRouter handles 404 Not Found and 405 Method Not Allowed responses automatically, but you can customize them:

```go
// Custom not found handler
r.NotFound(func(w http.ResponseWriter, r *http.Request, p router.Params) {
    router.RenderTemplate(w, r, "errors/404.html", nil)
})

// Custom method not allowed handler
r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request, p router.Params, allowedMethods []string) {
    w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
    router.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
})
```

## Router Lifecycle

The lifecycle of a request in MoraRouter is:

1. Server receives a request
2. Router matches the request to a registered route
3. Router extracts parameters from the URL
4. Router executes the middleware chain
5. If no middleware short-circuits, the handler is executed
6. The response is sent to the client

## ServeHTTP Implementation

MoraRouter implements the `http.Handler` interface, which allows it to be used with the standard Go HTTP server:

```go
func main() {
    r := router.New()
    // Configure router...
    
    http.ListenAndServe(":8080", r)
}
```

The `ServeHTTP` method is the entry point for all HTTP requests.

## Concurrency Model

MoraRouter is designed to be thread-safe and can handle concurrent requests. The route matching is performed using immutable data structures, so it's safe for concurrent use.

Routes are registered during setup and cannot be modified during request handling, which prevents race conditions.

## Middleware Execution Order

Middleware is executed in the order it's added to the router:

```go
r := router.New(
    router.WithLogging(),   // First middleware
    router.WithRecovery(),  // Second middleware
)

r.Use(AuthMiddleware)       // Third middleware

r.With(RoleMiddleware).Get("/admin", adminHandler)  // Fourth middleware, only for this route
```

## Options vs. Middleware

In MoraRouter:
- **Options** configure the router during creation
- **Middleware** processes requests at runtime

Both can add functionality, but they operate at different times:

```go
// Option: Sets up logging during router creation
r := router.New(router.WithLogging())

// Middleware: Applied to requests at runtime
r.Use(AuthMiddleware)
```

## Error Handling Patterns

MoraRouter provides several approaches for error handling:

1. **Recovery middleware**: Catches panics and responds with a 500 error
2. **Error helper**: `router.Error(w, status, message)`
3. **Custom error types**: Create your own error responses
4. **Error middleware**: Create middleware that handles specific errors

## Extending MoraRouter

MoraRouter is designed to be extended through:
- Custom middleware
- Custom options
- Custom renderer implementations
- Custom validator rules
- Plugins (via options)

## Core Components

Understanding the core components of MoraRouter:

1. **Router**: The main entry point that implements `http.Handler`
2. **Route**: Contains the pattern, HTTP method, and handler
3. **Segment**: A part of a route pattern (static or dynamic)
4. **Params**: A map of route parameters extracted from the URL
5. **HandlerFunc**: The function signature for request handlers
6. **Middleware**: Functions that wrap handlers
7. **Group**: A collection of routes with a common prefix
8. **Render**: Helpers for response generation
9. **Option**: Configuration functions for the router

## Conclusion

MoraRouter provides a flexible, powerful foundation for building web applications in Go. By understanding these core concepts, you'll be able to effectively use the router and extend it to fit your needs.

Next, explore [Routing](routing.md) to learn more about defining routes in MoraRouter.
