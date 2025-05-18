# Groups and Nesting in MoraRouter

Route groups are a powerful feature in MoraRouter that help you organize related routes under a common prefix. This guide covers how to use groups effectively, including middleware scoping, nesting, and other advanced patterns.

## Basic Route Groups

Create a group with a common prefix:

```go
r := router.New()

// Create a group for API routes
api := r.Group("/api")

// Add routes to the API group
api.Get("/users", listUsersHandler)      // /api/users
api.Post("/users", createUserHandler)    // /api/users
api.Get("/posts", listPostsHandler)      // /api/posts
api.Post("/posts", createPostHandler)    // /api/posts
```

## Middleware Scoping

Apply middleware to all routes in a group:

```go
// Create a group for admin routes
admin := r.Group("/admin")

// Apply auth middleware to all admin routes
admin.Use(AuthMiddleware)

// All these routes require authentication
admin.Get("/dashboard", dashboardHandler)
admin.Get("/users", adminListUsersHandler)
admin.Get("/settings", settingsHandler)

// Routes outside the group are unaffected
r.Get("/public", publicHandler) // No auth required
```

You can apply multiple middleware to a group:

```go
admin := r.Group("/admin")

// Apply multiple middleware in order
admin.Use(LoggingMiddleware, AuthMiddleware, AdminRoleMiddleware)

// These routes have all middleware applied
admin.Get("/dashboard", dashboardHandler)
admin.Get("/users", adminListUsersHandler)
```

## Nested Groups

Groups can be nested to create hierarchical route structures:

```go
// Create an API group
api := r.Group("/api")

// Create nested groups for API versions
v1 := api.Group("/v1")  // /api/v1
v2 := api.Group("/v2")  // /api/v2

// Add routes to each version
v1.Get("/users", v1ListUsersHandler)     // /api/v1/users
v1.Get("/posts", v1ListPostsHandler)     // /api/v1/posts

v2.Get("/users", v2ListUsersHandler)     // /api/v2/users
v2.Get("/posts", v2ListPostsHandler)     // /api/v2/posts
```

Middleware inheritance in nested groups:

```go
// API group with rate limiting middleware
api := r.Group("/api")
api.Use(RateLimitMiddleware(100, time.Minute))

// V1 group with logging middleware
v1 := api.Group("/v1")
v1.Use(LoggingMiddleware)

// V2 group with different logging middleware
v2 := api.Group("/v2")
v2.Use(DetailedLoggingMiddleware)

// Routes in v1 have both rate limiting and basic logging
v1.Get("/users", v1ListUsersHandler)

// Routes in v2 have rate limiting and detailed logging
v2.Get("/users", v2ListUsersHandler)
```

## Advanced Group Patterns

### Resource Groups

Group related resources:

```go
// Create an API group
api := r.Group("/api/v1")

// User resources
users := api.Group("/users")
users.Get("", listUsersHandler)              // /api/v1/users
users.Get("/:id", getUserHandler)            // /api/v1/users/:id
users.Post("", createUserHandler)            // /api/v1/users

// Post resources
posts := api.Group("/posts")
posts.Get("", listPostsHandler)              // /api/v1/posts
posts.Get("/:id", getPostHandler)            // /api/v1/posts/:id
posts.Post("", createPostHandler)            // /api/v1/posts
```

### Nested Resources

Handle nested resources with groups:

```go
// Create an API group
api := r.Group("/api/v1")

// User resources
users := api.Group("/users")
users.Get("", listUsersHandler)              // /api/v1/users
users.Get("/:userId", getUserHandler)        // /api/v1/users/:userId

// User posts as nested resources
userPosts := users.Group("/:userId/posts")
userPosts.Get("", listUserPostsHandler)      // /api/v1/users/:userId/posts
userPosts.Get("/:postId", getUserPostHandler) // /api/v1/users/:userId/posts/:postId
userPosts.Post("", createUserPostHandler)    // /api/v1/users/:userId/posts
```

### Group with Authentication Levels

Different authentication levels for different groups:

```go
// Public API routes
public := r.Group("/api/public")
public.Get("/products", listPublicProductsHandler)

// Protected API routes (require authentication)
protected := r.Group("/api/protected")
protected.Use(AuthMiddleware)
protected.Get("/user-profile", userProfileHandler)

// Admin API routes (require admin role)
admin := r.Group("/api/admin")
admin.Use(AuthMiddleware, AdminRoleMiddleware)
admin.Get("/dashboard", adminDashboardHandler)
```

### Feature-Based Groups

Organize routes by feature:

```go
// Authentication routes
auth := r.Group("/auth")
auth.Post("/login", loginHandler)
auth.Post("/register", registerHandler)
auth.Post("/forgot-password", forgotPasswordHandler)
auth.Post("/reset-password", resetPasswordHandler)

// User profile routes
profile := r.Group("/profile")
profile.Use(AuthMiddleware)
profile.Get("", getProfileHandler)
profile.Put("", updateProfileHandler)
profile.Delete("", deleteProfileHandler)

// Product routes
products := r.Group("/products")
products.Get("", listProductsHandler)
products.Get("/:id", getProductHandler)
```

### Conditional Middleware

Apply middleware conditionally:

```go
// Create API group
api := r.Group("/api")

// Apply rate limiting only in production
if environment == "production" {
    api.Use(RateLimitMiddleware(100, time.Minute))
}

// Always apply authentication for protected routes
protected := api.Group("/protected")
protected.Use(AuthMiddleware)
```

## Named Routes in Groups

Give names to routes in groups for URL generation:

```go
// Create a group
users := r.Group("/users")

// Add named routes
users.Get("", listUsersHandler).Name("users.list")
users.Get("/:id", getUserHandler).Name("users.show")
users.Post("", createUserHandler).Name("users.create")
users.Put("/:id", updateUserHandler).Name("users.update")
users.Delete("/:id", deleteUserHandler).Name("users.delete")

// Generate URLs from named routes
listURL, _ := r.URL("users.list")                // /users
showURL, _ := r.URL("users.show", "123")         // /users/123
```

## Group with Custom Options

Create a group with custom options:

```go
// Create a group with options
admin := r.GroupWithOptions("/admin", router.GroupOptions{
    Middleware: []router.Middleware{AuthMiddleware, AdminRoleMiddleware},
    NotFound:   customNotFoundHandler,
})

// Add routes to the group
admin.Get("/dashboard", dashboardHandler)
```

## Domain-Specific Groups

Group routes by domain (advanced):

```go
// API subdomain
api := r.Domain("api.example.com")
api.Get("/users", apiListUsersHandler)

// Admin subdomain
admin := r.Domain("admin.example.com")
admin.Get("/dashboard", adminDashboardHandler)

// Main domain
r.Get("/", homeHandler)
```

## Method-Specific Groups

Group routes by HTTP method:

```go
// Only GET methods
getRoutes := r.Methods("GET")
getRoutes.Route("/users", listUsersHandler)
getRoutes.Route("/posts", listPostsHandler)

// Only POST methods
postRoutes := r.Methods("POST")
postRoutes.Route("/users", createUserHandler)
postRoutes.Route("/posts", createPostHandler)
```

## Group with Resource Controllers

Use resource controllers with groups:

```go
// API group
api := r.Group("/api")

// Register resources in the group
api.Resource("/users", UserController{})
api.Resource("/posts", PostController{})

// Nested resources
users := api.Group("/users")
users.Resource("/:userId/posts", UserPostController{})
```

## Practical Example

Here's a complete example showing how to organize a larger application with groups:

```go
package main

import (
    "net/http"
    
    "github.com/yourusername/mora-router/router"
)

func main() {
    r := router.New(router.WithLogging(), router.WithRecovery())
    
    // Static files
    r.Static("/assets", "./public")
    
    // Public routes
    r.Get("/", homeHandler)
    r.Get("/about", aboutHandler)
    r.Get("/contact", contactHandler)
    
    // Authentication routes
    auth := r.Group("/auth")
    auth.Post("/login", loginHandler)
    auth.Post("/register", registerHandler)
    auth.Post("/logout", logoutHandler)
    auth.Post("/forgot-password", forgotPasswordHandler)
    auth.Post("/reset-password/:token", resetPasswordHandler)
    
    // API routes
    api := r.Group("/api")
    api.Use(APIMiddleware) // Apply to all API routes
    
    // API v1
    v1 := api.Group("/v1")
    
    // Public API endpoints
    v1.Get("/products", listProductsHandler)
    v1.Get("/products/:id", getProductHandler)
    v1.Get("/categories", listCategoriesHandler)
    
    // Protected API endpoints
    protected := v1.Group("")
    protected.Use(JWTAuthMiddleware)
    
    // User profile
    protected.Get("/profile", getProfileHandler)
    protected.Put("/profile", updateProfileHandler)
    
    // User orders
    orders := protected.Group("/orders")
    orders.Get("", listOrdersHandler)
    orders.Get("/:id", getOrderHandler)
    orders.Post("", createOrderHandler)
    
    // Admin routes
    admin := r.Group("/admin")
    admin.Use(JWTAuthMiddleware, AdminRoleMiddleware)
    
    // Admin dashboard
    admin.Get("/dashboard", adminDashboardHandler)
    
    // Admin resources
    admin.Resource("/users", AdminUserController{})
    admin.Resource("/products", AdminProductController{})
    admin.Resource("/orders", AdminOrderController{})
    admin.Resource("/categories", AdminCategoryController{})
    
    // Start the server
    http.ListenAndServe(":8080", r)
}
```

## Best Practices

When working with groups, consider these best practices:

1. **Group by feature**: Organize routes based on features or resources.
2. **Use meaningful prefixes**: Choose clear, descriptive names for your route groups.
3. **Keep nesting shallow**: Avoid deeply nested groups to maintain readability.
4. **Apply middleware at the appropriate level**: Add middleware only where it's needed.
5. **Name your routes**: Use consistent naming conventions for URL generation.
6. **Consider versioning**: Use groups for API versioning.
7. **Separate public and protected routes**: Clearly distinguish routes that require authentication.

## Conclusion

Route groups in MoraRouter provide a flexible way to organize your routes and apply middleware selectively. By using groups effectively, you can create clean, maintainable route structures for even the largest applications.

Next, explore [Middleware](middleware.md) to learn more about extending your routes with custom functionality.
