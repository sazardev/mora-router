# Route Macros and Advanced Routing Patterns

One of MoraRouter's most powerful features is the ability to define and use route macros - reusable routing patterns that make your code more maintainable and expressive. This guide covers everything you need to know about working with route macros and advanced routing techniques.

## What Are Route Macros?

Route macros are named, reusable routing patterns that can be applied across your application. They help you:

- Keep your routing code DRY (Don't Repeat Yourself)
- Standardize URL patterns across your application
- Apply consistent middleware to similar routes
- Simplify complex routing logic

## Basic Usage

### Registering a Custom Macro

Register custom macros early in your application setup:

```go
package main

import (
    "net/http"
    "time"
    
    "github.com/yourusername/mora-router/router"
)

func main() {
    // Register a custom macro
    router.RegisterMacro(
        "paginated",               // Macro name
        "/:page(\\d+)",            // URL pattern
        []string{"GET"},           // Allowed HTTP methods
        rateLimitMiddleware(100, time.Minute) // Optional middleware
    )
    
    r := router.New()
    
    // Use the macro
    r.UseMacro("/users", "paginated", func(w http.ResponseWriter, req *http.Request, p router.Params) {
        page := p["page"] // The page comes from the :page parameter
        router.JSON(w, http.StatusOK, map[string]string{"page": page})
    })
    
    // This is equivalent to:
    // r.Get("/users/:page(\\d+)", rateLimitMiddleware(100, time.Minute)(func(w http.ResponseWriter, req *http.Request, p router.Params) {
    //     page := p["page"]
    //     router.JSON(w, http.StatusOK, map[string]string{"page": page})
    // }))
    
    http.ListenAndServe(":8080", r)
}
```

### Using MoraRouter's Built-in Macros

MoraRouter comes with several built-in macros for common patterns:

```go
// Create a resource with CRUD endpoints
r.UseMacro("/products", "resource", ProductController{})

// Create a read-only resource 
r.UseMacro("/categories", "readonly-resource", CategoryController{})

// Create API endpoints with versioning
r.UseMacro("/users", "api-resource", UserController{})
```

## Built-in Macros Reference

MoraRouter provides these built-in macros:

| Macro Name | URL Pattern | HTTP Methods | Description |
|------------|-------------|--------------|-------------|
| `resource` | Various | GET, POST, PUT, DELETE | Complete CRUD resource |
| `readonly-resource` | `/, /:id` | GET | Read-only resource (index, show) |
| `api-resource` | Various | GET, POST, PUT, DELETE | API-style resource with JSON responses |
| `paginated` | `/:page(\d+)` | GET | Resource with page parameter |
| `api-version` | `/v:version(\d+)/*` | All | Versioned API endpoints |
| `cached` | `/*` | GET | Cacheable responses |
| `authenticated` | `/*` | All | Routes requiring authentication |
| `form` | `/` | GET, POST | Form handling with CSRF protection |

## Creating Advanced Macros

You can create complex macros with multiple parameters:

```go
// Advanced paginated search macro
router.RegisterMacro(
    "paginated-search",
    "/:page(\\d+)/q/:query",
    []string{"GET"},
    searchRateLimiter,
    searchLogger,
)

// Usage
r.UseMacro("/products", "paginated-search", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    page := p["page"]
    query := p["query"]
    // Search products with pagination
})
```

## Macro Parameters and URL Building

Macros can be used to build URLs consistently:

```go
// Register URL builder
router.RegisterMacroURL("product-page", "/products/:page(\\d+)")

// Use in template or code
url := router.BuildURL("product-page", map[string]string{
    "page": "5",
})
// url = "/products/5"
```

This ensures consistent URL structure throughout your application.

## Combining Multiple Macros

You can stack macros for complex routing patterns:

```go
// First register the macros
router.RegisterMacro("api", "/api/v:version(\\d+)", []string{"GET", "POST"}, apiKeyAuth)
router.RegisterMacro("json", "", []string{"GET", "POST"}, jsonContentType)

// Then use them together
r.UseMacro("/users", "api+json", userHandler)
// This creates routes like: /api/v1/users, /api/v2/users
// With both apiKeyAuth and jsonContentType middleware applied
```

## Dynamic Macro Registration

You can register macros dynamically at runtime:

```go
// Dynamically create tenant-specific macros
func registerTenantMacro(tenantID string) {
    macroName := "tenant-" + tenantID
    router.RegisterMacro(
        macroName,
        "/:resource",
        []string{"GET", "POST", "PUT", "DELETE"},
        tenantAuthMiddleware(tenantID),
    )
}

// Later in your code
for _, tenant := range getTenants() {
    registerTenantMacro(tenant.ID)
    r.UseMacro("/tenant/" + tenant.ID, "tenant-" + tenant.ID, tenantHandler)
}
```

## Macro Groups

Organize related macros into logical groups:

```go
// Register a group of related macros
router.RegisterMacroGroup("admin", map[string]router.MacroDefinition{
    "list":    {Pattern: "", Methods: []string{"GET"}, Middleware: adminAuth},
    "create":  {Pattern: "/create", Methods: []string{"GET", "POST"}, Middleware: adminAuth},
    "edit":    {Pattern: "/:id/edit", Methods: []string{"GET", "PUT"}, Middleware: adminAuth},
    "delete":  {Pattern: "/:id/delete", Methods: []string{"GET", "DELETE"}, Middleware: adminAuth},
})

// Use any macro from the group
r.UseMacro("/users", "admin.list", listUsersHandler)
r.UseMacro("/users", "admin.create", createUserHandler)
```

## CRUD Resource Macros

Create complete RESTful resources with a single line:

```go
// Register a CRUD resource macro
r.UseMacro("/products", "crud", ProductController{})

// This creates:
// GET    /products          -> Index()  - List all
// GET    /products/:id      -> Show()   - Show one
// GET    /products/create   -> New()    - Show create form
// POST   /products          -> Create() - Create new
// GET    /products/:id/edit -> Edit()   - Show edit form
// PUT    /products/:id      -> Update() - Update existing
// DELETE /products/:id      -> Delete() - Delete existing
```

## Customizing Built-in Macros

Override or extend built-in macros to match your application's needs:

```go
// Extend the resource macro with extra routes
router.ExtendMacro("resource", map[string]router.MacroRoute{
    "archive": {
        Pattern: "/:id/archive",
        Methods: []string{"PUT"},
    },
    "restore": {
        Pattern: "/:id/restore", 
        Methods: []string{"PUT"},
    },
})

// Now the resource macro includes archive and restore routes
r.UseMacro("/products", "resource", ProductController{})
```

## Conditional Macros

Apply macros conditionally based on runtime factors:

```go
// Conditionally use different macros
if appConfig.EnableReadOnly {
    r.UseMacro("/products", "readonly-resource", ProductController{})
} else {
    r.UseMacro("/products", "resource", ProductController{})
}
```

## Advanced URL Pattern Matching

MoraRouter supports sophisticated URL patterns within macros:

```go
// Complex parameter validation
router.RegisterMacro(
    "product-code",
    "/:code([A-Z]{3}\\d{4})",
    []string{"GET"},
    nil,
)

// Wildcard patterns with constraints
router.RegisterMacro(
    "nested-resources",
    "/:parentId(\\d+)/:resource/:childId(\\d+)",
    []string{"GET", "PUT", "DELETE"},
    nil,
)

// Optional parameters
router.RegisterMacro(
    "optional-format",
    ".:format?",
    []string{"GET"},
    nil,
)
```

## Best Practices for Route Macros

1. **Consistent Naming**: Establish naming conventions for your macros
2. **Documentation**: Document each macro's purpose and parameters
3. **Logical Grouping**: Group related macros together
4. **Reuse Middleware**: Use macros to standardize middleware application
5. **Version Control**: Consider versioning important macros when APIs evolve

## Debugging Macros

MoraRouter provides tools to debug and inspect your macros:

```go
// Print all registered macros
router.PrintMacros()

// Check if a specific path matches a macro
matches, params := router.TestMacroMatch("paginated", "/products/5")
// matches = true, params = map[string]string{"page": "5"}

// Debug a specific macro's expansion
expanded := router.ExpandMacro("/products", "resource")
fmt.Println(expanded)
// Prints all the expanded routes and handlers
```

## Case Study: API Versioning with Macros

Here's a real-world example of using macros for API versioning:

```go
// Register API version macros
for i := 1; i <= 3; i++ {
    version := fmt.Sprintf("v%d", i)
    router.RegisterMacro(
        "api-"+version,
        "/api/"+version+"/:resource",
        []string{"GET", "POST", "PUT", "DELETE"},
        apiVersionMiddleware(i),
    )
}

// V1 API uses one controller
r.UseMacro("/users", "api-v1", UserControllerV1{})

// V2 API uses another controller
r.UseMacro("/users", "api-v2", UserControllerV2{})

// V3 API uses the latest controller
r.UseMacro("/users", "api-v3", UserControllerV3{})
```

## Conclusion

Route macros are a powerful way to keep your routing code clean, maintainable, and consistent. By leveraging macros effectively, you can create expressive, DRY routing that scales with your application complexity.

Whether you're building a simple website or a complex API, MoraRouter's macro system gives you the tools to organize your routes logically and efficiently.

Have fun building with MoraRouter macros! 🚀
