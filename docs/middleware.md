# Middleware in MoraRouter

Middleware provides a powerful way to inject functionality into your request-handling pipeline. MoraRouter's middleware system makes it easy to apply cross-cutting concerns like logging, authentication, and more.

## What is Middleware?

In MoraRouter, middleware is a function that takes a handler function and returns a new handler function:

```go
type Middleware func(HandlerFunc) HandlerFunc
```

Middleware can perform operations before the handler executes, modify the request, intercept the response, or perform operations after the handler executes.

## Built-in Middleware

MoraRouter comes with several built-in middleware options that you can enable via options:

### Logging Middleware

```go
r := router.New(router.WithLogging())
```

This middleware logs each request with method, path, status code, and execution time:

```
2023/06/15 12:30:45 [INFO] GET /users -> 200 (45.2ms)
```

### Recovery Middleware

```go
r := router.New(router.WithRecovery())
```

This middleware recovers from panics in your handlers, preventing your server from crashing and returning a 500 response.

### CORS Middleware

```go
// Allow all origins
r := router.New(router.WithCORS("*"))

// Allow specific origin
r := router.New(router.WithCORS("https://example.com"))
```

Configures Cross-Origin Resource Sharing (CORS) headers to allow browsers to make cross-origin requests.

### Cache Middleware

```go
r := router.New(router.WithCache(time.Minute))
```

Caches responses for GET requests to improve performance.

### Rate Limiting

```go
r := router.New(router.WithRateLimit(100, time.Minute))
```

Limits the number of requests from a single IP address to prevent abuse.

### JWT Authentication

```go
r := router.New(router.WithJWT("your-secret-key"))
```

Verifies JSON Web Tokens in the Authorization header and makes the claims available to handlers.

### Metrics

```go
r := router.New(router.WithMetrics())
```

Collects request metrics and exposes them on a `/metrics` endpoint, compatible with Prometheus.

### API Versioning

```go
r := router.New(router.WithAPIVersioning("X-API-Version", "1"))
```

Automatically handles API versioning based on a header or URL.

## Custom Middleware

Creating your own middleware is straightforward:

```go
// Simple auth middleware
func AuthMiddleware(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        token := r.Header.Get("Authorization")
        
        if token == "" {
            router.Error(w, http.StatusUnauthorized, "Authorization required")
            return
        }
        
        // Validate token...
        if !isValidToken(token) {
            router.Error(w, http.StatusForbidden, "Invalid token")
            return
        }
        
        // Call the next handler
        next(w, r, p)
    }
}

// Use it
r.With(AuthMiddleware).Get("/protected", protectedHandler)
```

## Middleware with Parameters

You can create middleware factories that accept parameters:

```go
// Middleware factory that requires a specific role
func RequireRole(role string) router.Middleware {
    return func(next router.HandlerFunc) router.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request, p router.Params) {
            // Get user from context (set by previous auth middleware)
            user := getUserFromContext(r.Context())
            
            if !user.HasRole(role) {
                router.Error(w, http.StatusForbidden, "Insufficient permissions")
                return
            }
            
            next(w, r, p)
        }
    }
}

// Use with specific roles
r.With(AuthMiddleware, RequireRole("admin")).Get("/admin", adminHandler)
r.With(AuthMiddleware, RequireRole("editor")).Get("/content", contentHandler)
```

## Response Modification Middleware

Middleware can also modify the response:

```go
// Add security headers to all responses
func SecurityHeadersMiddleware(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        
        next(w, r, p)
    }
}
```

## Request Timing Middleware

Measure execution time:

```go
func TimingMiddleware(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        start := time.Now()
        
        // Call the next handler
        next(w, r, p)
        
        // Calculate duration after handler returns
        duration := time.Since(start)
        
        // Add header to response
        w.Header().Set("X-Response-Time", duration.String())
    }
}
```

## Access Control Middleware

Control access based on IP address:

```go
func IPWhitelistMiddleware(allowedIPs []string) router.Middleware {
    // Convert to map for faster lookups
    allowed := make(map[string]bool)
    for _, ip := range allowedIPs {
        allowed[ip] = true
    }
    
    return func(next router.HandlerFunc) router.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request, p router.Params) {
            // Get client IP
            ip := getClientIP(r)
            
            if !allowed[ip] {
                router.Error(w, http.StatusForbidden, "Access denied")
                return
            }
            
            next(w, r, p)
        }
    }
}

// Use with a list of allowed IPs
r.With(IPWhitelistMiddleware([]string{"127.0.0.1", "192.168.1.10"})).
  Get("/admin", adminHandler)
```

## Applying Middleware

There are several ways to apply middleware in MoraRouter:

### Global Middleware

Applied to all routes:

```go
r := router.New(router.WithLogging(), router.WithRecovery())
```

### Route-Specific Middleware

Applied to a specific route:

```go
r.With(AuthMiddleware).Get("/protected", protectedHandler)
```

### Group Middleware

Applied to a group of routes:

```go
admin := r.Group("/admin")
admin.Use(AuthMiddleware, RequireRole("admin"))
admin.Get("/dashboard", dashboardHandler)
admin.Get("/users", usersHandler)
```

### Middleware Order

Middleware is applied in the order it's added:

```go
// Order: Logging -> Auth -> Handler
r.With(LoggingMiddleware, AuthMiddleware).Get("/path", handler)

// Order: Auth -> Logging -> Handler
r.With(AuthMiddleware, LoggingMiddleware).Get("/path", handler)
```

## Middleware Registry

MoraRouter also supports a middleware registry for organizing and applying middleware by name:

```go
// Register middleware by name
r := router.New()
r.RegisterMiddleware("auth", AuthMiddleware)
r.RegisterMiddleware("admin", RequireRole("admin"))

// Apply by name
r.WithNamed("auth", "admin").Get("/admin", adminHandler)

// Apply global middleware by name
r := router.New(router.UseMiddleware("logging", "recovery"))
```

## Common Middleware Patterns

### Request ID

Add a unique ID to each request:

```go
func RequestIDMiddleware(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        id := uuid.New().String()
        w.Header().Set("X-Request-ID", id)
        
        // Store in context for logging
        ctx := context.WithValue(r.Context(), "request-id", id)
        next(w, r.WithContext(ctx), p)
    }
}
```

### Response Compression

Compress response data:

```go
func CompressionMiddleware(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        // Check if client supports gzip
        if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
            next(w, r, p)
            return
        }
        
        // Create gzip writer
        gz := gzip.NewWriter(w)
        defer gz.Close()
        
        // Replace writer with gzip writer
        gzw := gzipResponseWriter{
            ResponseWriter: w,
            Writer:         gz,
        }
        
        // Set content encoding header
        w.Header().Set("Content-Encoding", "gzip")
        
        // Call next handler with gzip writer
        next(gzw, r, p)
    }
}

type gzipResponseWriter struct {
    http.ResponseWriter
    Writer *gzip.Writer
}

func (gzw gzipResponseWriter) Write(b []byte) (int, error) {
    return gzw.Writer.Write(b)
}
```

## Chaining Multiple Middleware

You can chain multiple middleware together:

```go
secureAPI := func(h router.HandlerFunc) router.HandlerFunc {
    return AuthMiddleware(
        LoggingMiddleware(
            RateLimitMiddleware(
                h
            )
        )
    )
}

// Or using the With method
r.With(
    AuthMiddleware,
    LoggingMiddleware,
    RateLimitMiddleware,
).Get("/secure", secureHandler)
```

## Conclusion

Middleware is a powerful tool for adding cross-cutting functionality to your application. MoraRouter's middleware system is flexible and easy to use, allowing you to create reusable components that can be applied to routes as needed.

For more examples of middleware, check the [examples](examples.md) section.
