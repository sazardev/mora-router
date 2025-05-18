# API Versioning

Versioning your API is crucial for maintaining backward compatibility while introducing new features. MoraRouter provides several elegant approaches to API versioning that are both developer and client-friendly.

## URL-Based Versioning

URL-based versioning is the most explicit and widely-used approach, where the version is embedded directly in the URL path.

### Basic URL-Based Versioning

```go
package main

import (
    "net/http"
    
    "github.com/yourusername/mora-router/router"
)

func main() {
    r := router.New()
    
    // API version 1
    v1 := r.Group("/api/v1")
    v1.Get("/users", listUsersV1)
    v1.Get("/products", listProductsV1)
    
    // API version 2
    v2 := r.Group("/api/v2")
    v2.Get("/users", listUsersV2)
    v2.Get("/products", listProductsV2)
    
    http.ListenAndServe(":8080", r)
}
```

This creates endpoints like:
- `GET /api/v1/users`
- `GET /api/v1/products`
- `GET /api/v2/users`
- `GET /api/v2/products`

### Dynamic Version Groups

You can dynamically create version groups for more flexibility:

```go
func setupVersions(r *router.Router, versions []int) {
    for _, v := range versions {
        version := fmt.Sprintf("v%d", v)
        vGroup := r.Group("/api/" + version)
        
        // Register version-specific routes
        setupVersionRoutes(vGroup, v)
    }
}

func setupVersionRoutes(g *router.Group, version int) {
    // Common routes across all versions
    g.Get("/status", statusHandler)
    
    // Version-specific implementations
    switch version {
    case 1:
        g.Get("/users", usersV1Handler)
    case 2:
        g.Get("/users", usersV2Handler)
        g.Get("/analytics", analyticsV2Handler)  // New in v2
    case 3:
        g.Get("/users", usersV3Handler)
        g.Get("/analytics", analyticsV3Handler)
        g.Get("/metrics", metricsV3Handler)  // New in v3
    }
}
```

## Header-Based Versioning

MoraRouter supports API versioning based on request headers, which keeps URLs clean while still allowing version-specific endpoints.

```go
r := router.New(router.WithAPIVersioning("X-API-Version", "1"))

// Define routes without version in URL
r.Get("/users", userHandler)

// Client requests with different headers get different handlers
// X-API-Version: 1 → userHandlerV1
// X-API-Version: 2 → userHandlerV2
```

### Custom Header Version Selection

```go
r := router.New(router.WithCustomVersioning(func(req *http.Request) string {
    // Get version from header, with fallback to query parameter
    version := req.Header.Get("X-API-Version")
    if version == "" {
        version = req.URL.Query().Get("version")
    }
    if version == "" {
        return "1" // Default version
    }
    return version
}))

// Register version-specific handlers
r.Version("1").Get("/users", usersV1Handler)
r.Version("2").Get("/users", usersV2Handler)
```

## Content Negotiation Versioning

Content negotiation versioning uses the `Accept` header to determine the API version:

```go
r := router.New(router.WithAcceptHeaderVersioning("application/vnd.myapi.v"))

// Client requests with different Accept headers get different handlers
// Accept: application/vnd.myapi.v1+json → userHandlerV1
// Accept: application/vnd.myapi.v2+json → userHandlerV2

// Define routes with content type versions
r.Get("/users", router.ContentTypeHandler(map[string]router.HandlerFunc{
    "v1": usersV1Handler,
    "v2": usersV2Handler,
}))
```

## Query Parameter Versioning

Query parameter versioning allows clients to specify the version in the URL query string:

```go
r := router.New(router.WithQueryVersioning("version", "1"))

// These routes automatically handle version query parameters
r.Get("/users", router.VersionHandler(map[string]router.HandlerFunc{
    "1": usersV1Handler,
    "2": usersV2Handler,
}))
```

## Mixed Versioning Strategy

For maximum flexibility, you can combine multiple versioning strategies:

```go
r := router.New(router.WithMixedVersioning(
    // Check header first
    router.HeaderVersionExtractor("X-API-Version"),
    // Then check Accept header
    router.AcceptHeaderVersionExtractor("application/vnd.myapi.v"),
    // Finally try query parameter
    router.QueryVersionExtractor("version"),
    // Default version if none specified
    "1",
))

// Register handlers in a version-agnostic way
r.Version("1").Get("/users", usersV1Handler)
r.Version("2").Get("/users", usersV2Handler)
```

## Route Deprecation

MoraRouter provides tools for gracefully managing deprecated API routes:

```go
// Mark route as deprecated in version 2
r.Get("/legacy/endpoint", legacyHandler).Deprecated("2", "Use /new/endpoint instead")

// Add warning header on deprecated routes
r.Use(router.DeprecationWarnings())
```

This will automatically add a `Warning` header to responses when deprecated endpoints are accessed.

## Automatic Documentation

Versioned APIs automatically include version information in generated OpenAPI/Swagger documentation:

```go
r := router.New(
    router.WithSwagger(),
    router.WithAPIVersioning("X-API-Version", "1"),
)

// Configure Swagger info for each version
r.SwaggerInfoForVersion("1", &router.SwaggerInfo{
    Title: "My API v1",
    Version: "1.0.0",
})

r.SwaggerInfoForVersion("2", &router.SwaggerInfo{
    Title: "My API v2",
    Version: "2.0.0",
})
```

## Version Fallback

MoraRouter can automatically fall back to earlier API versions if a requested version doesn't support an endpoint:

```go
r := router.New(
    router.WithVersionFallback(true),
    router.WithAPIVersioning("X-API-Version", "3"),
)

// v1 and v2 both have this endpoint
r.Version("1").Get("/common", commonHandlerV1)
r.Version("2").Get("/common", commonHandlerV2)

// Only v2 has this endpoint
r.Version("2").Get("/feature", featureHandler)

// A request to /feature with X-API-Version: 1 will get a 404
// A request to /feature with X-API-Version: 3 will use the v2 handler
```

## Version-based Middleware

Apply middleware only for specific API versions:

```go
// Apply rate limiting only on v1 API
r.Version("1").Use(router.RateLimit(100, time.Minute))

// Apply different authentication for different versions
r.Version("1").Use(basicAuthMiddleware)
r.Version("2").Use(jwtAuthMiddleware)
```

## API Migration Assistant

For larger APIs, MoraRouter provides a migration assistant to help track and implement version changes:

```go
// Create migration plan
migration := router.NewVersionMigration("1", "2")

// Register endpoints to be changed
migration.RouteChanged("/users/:id", "Modified response format")
migration.RouteAdded("/users/:id/settings", "New endpoint for user settings")
migration.RouteRemoved("/legacy/endpoint", "Use /new/endpoint instead")

// Generate migration report
report := migration.GenerateReport()
fmt.Println(report)
```

## Best Practices for API Versioning

1. **Choose a consistent versioning strategy** that suits your application and clients
2. **Document version changes thoroughly** for API consumers
3. **Support at least one previous version** when making breaking changes
4. **Use semantic versioning principles** when deciding when to increment versions
5. **Consider version lifespan** and communicate deprecation timelines
6. **Test all supported versions** to ensure backward compatibility
7. **Gradually phase out older versions** rather than removing them abruptly

## Conclusion

With MoraRouter's flexible versioning options, you can create robust, evolvable APIs that preserve backward compatibility while still allowing your API to grow and improve over time. Choose the versioning approach that best fits your application's needs and client expectations.
