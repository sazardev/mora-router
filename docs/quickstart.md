# Quick Start Guide

This guide will help you get started with MoraRouter in just a few minutes, covering installation, basic setup, and your first endpoints.

## Installation

First, make sure you have Go installed (version 1.18 or newer recommended).

```bash
go get -u github.com/yourusername/mora-router
```

## Creating Your First Router

Create a new file named `main.go`:

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/yourusername/mora-router/router"
)

func main() {
    // Create a new router with logging middleware
    r := router.New(router.WithLogging())
    
    // Define a simple route
    r.Get("/hello", func(w http.ResponseWriter, req *http.Request, p router.Params) {
        w.Header().Set("Content-Type", "text/plain")
        w.Write([]byte("Hello, World!"))
    })
    
    // Route with parameter
    r.Get("/hello/:name", func(w http.ResponseWriter, req *http.Request, p router.Params) {
        name := p["name"] // Get the parameter value
        w.Header().Set("Content-Type", "text/plain")
        w.Write([]byte("Hello, " + name + "!"))
    })
    
    // Start the server
    log.Println("Server started on :8080")
    http.ListenAndServe(":8080", r)
}
```

Run your application:

```bash
go run main.go
```

Visit `http://localhost:8080/hello` and `http://localhost:8080/hello/YourName` in your browser.

## Add JSON Response

Let's modify our example to use JSON responses:

```go
r.Get("/hello/:name", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    // Use the built-in JSON helper
    router.JSON(w, http.StatusOK, map[string]interface{}{
        "message": "Hello, " + p["name"] + "!",
        "timestamp": time.Now().Unix(),
    })
})
```

## Adding Middleware

MoraRouter comes with several built-in middleware options:

```go
// Create a router with multiple middleware options
r := router.New(
    router.WithLogging(),        // Log all requests
    router.WithRecovery(),       // Recover from panics
    router.WithCORS("*"),        // Enable CORS
    router.WithCache(time.Minute), // Cache responses
)
```

## Creating a REST Resource

MoraRouter makes it easy to create RESTful API endpoints:

```go
// Define a simple controller
type UserController struct {
    router.DefaultController
}

// Override the Show method
func (c UserController) Show(w http.ResponseWriter, r *http.Request, p router.Params) {
    id := p["id"]
    router.JSON(w, http.StatusOK, map[string]interface{}{
        "id": id,
        "name": "User " + id,
        "email": "user" + id + "@example.com",
    })
}

// Register the resource
r.Resource("/users", UserController{})

// This automatically creates:
// GET    /users         -> Index()  - List all users
// POST   /users         -> Create() - Create a new user
// GET    /users/:id     -> Show()   - Get a single user
// PUT    /users/:id     -> Update() - Update a user
// DELETE /users/:id     -> Delete() - Delete a user
```

## Data Validation

Add automatic request validation:

```go
// Define a struct for validation
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=3"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=18"`
}

// Use the binding middleware
r.Post("/users", router.BindJSON(func(w http.ResponseWriter, r *http.Request, p router.Params, input CreateUserRequest) {
    // If we get here, input has been validated and parsed
    router.JSON(w, http.StatusCreated, map[string]interface{}{
        "message": "User created",
        "user": input,
    })
}))
```

## Route Groups

Organize routes with groups:

```go
// Create an API group
api := r.Group("/api/v1")

// Add routes to the group
api.Get("/users", listUsersHandler)
api.Post("/users", createUserHandler)
api.Get("/users/:id", getUserHandler)

// Apply middleware to a group
admin := r.Group("/admin")
admin.Use(authMiddleware)
admin.Get("/dashboard", dashboardHandler)
```

## Next Steps

- Check the [Core Concepts](core-concepts.md) guide to understand MoraRouter's architecture
- See [Routing](routing.md) for more advanced routing patterns
- Explore [Middleware](middleware.md) to enhance your application

## Complete Example

Here's a more complete example combining many features:

```go
package main

import (
    "log"
    "net/http"
    "time"
    
    "github.com/yourusername/mora-router/router"
)

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

type UserController struct {
    router.DefaultController
}

func (c UserController) Index(w http.ResponseWriter, r *http.Request, p router.Params) {
    users := []User{
        {ID: "1", Name: "Alice", Email: "alice@example.com"},
        {ID: "2", Name: "Bob", Email: "bob@example.com"},
    }
    router.JSON(w, http.StatusOK, users)
}

func (c UserController) Show(w http.ResponseWriter, r *http.Request, p router.Params) {
    id := p["id"]
    user := User{ID: id, Name: "User " + id, Email: "user" + id + "@example.com"}
    router.JSON(w, http.StatusOK, user)
}

func main() {
    r := router.New(
        router.WithLogging(),
        router.WithRecovery(),
        router.WithCORS("*"),
    )
    
    // Home page
    r.Get("/", func(w http.ResponseWriter, r *http.Request, p router.Params) {
        router.JSON(w, http.StatusOK, map[string]interface{}{
            "message": "Welcome to MoraRouter API",
            "version": "1.0.0",
            "documentation": "/docs",
        })
    })
    
    // API routes
    api := r.Group("/api/v1")
    
    // Resource routes
    api.Resource("/users", UserController{})
    
    // Static files
    r.Static("/assets", "./public")
    
    // SPA fallback
    r.SPA("/app", "./web/dist", "index.html")
    
    // Start server
    log.Println("Server started on :8080")
    http.ListenAndServe(":8080", r)
}
```
