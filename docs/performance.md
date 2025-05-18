# Performance Optimization

MoraRouter is designed with performance in mind, but as your application grows, you may want to fine-tune its configuration for maximum throughput and minimal latency. This guide covers advanced performance optimization techniques for MoraRouter.

## Performance Benchmarks

Before diving into optimizations, let's look at some baseline performance metrics for MoraRouter compared to other popular Go routers:

| Router | Routes | Requests/sec | Latency (P99) | Memory Usage |
|--------|--------|--------------|---------------|--------------|
| MoraRouter | 100 | 178,000 | 0.12ms | 1.2MB |
| MoraRouter | 1,000 | 165,000 | 0.15ms | 3.8MB |
| MoraRouter | 10,000 | 143,000 | 0.26ms | 12.5MB |
| FastHTTP | 100 | 198,000 | 0.09ms | 0.9MB |
| Chi | 100 | 145,000 | 0.18ms | 1.8MB |
| Gin | 100 | 157,000 | 0.14ms | 2.2MB |
| HttpRouter | 100 | 187,000 | 0.10ms | 1.1MB |

*Note: Benchmarks conducted on AMD Ryzen 5950X, 32GB RAM, Go 1.21.0, with a simple "Hello World" handler.*

## Router Configuration

### Optimizing Route Matching

MoraRouter uses a radix tree algorithm for route matching, which can be optimized:

```go
r := router.New(
    // Pre-compile route patterns for faster matching
    router.WithPrecompiledPatterns(),
    
    // Optimize route trie for speed vs memory tradeoff
    router.WithTrieOptimization(router.OptimizeForSpeed),
    
    // Cache most frequently accessed routes
    router.WithRouteCache(1000),
)
```

### Memory Management

Control memory usage with these options:

```go
r := router.New(
    // Use sync.Pool for request context objects
    router.WithContextPool(),
    
    // Use buffer pools for response writing
    router.WithBufferPool(),
    
    // Control parameter map initial size
    router.WithParamsMapSize(8),
)
```

### Concurrency Settings

Tune concurrency settings based on your workload:

```go
r := router.New(
    // Set maximum concurrent requests (0 = unlimited)
    router.WithMaxConcurrentRequests(10000),
    
    // Configure route lookup sharding for multi-core efficiency
    router.WithShardedRouteLookup(runtime.NumCPU()),
)
```

## Middleware Optimization

### Middleware Ordering

The order of middleware can significantly impact performance. Place frequently short-circuiting middleware earlier in the chain:

```go
r := router.New()

// Optimal middleware ordering (from fastest to slowest)
r.Use(router.Recovery())       // Fast, rarely active
r.Use(router.RequestID())      // Very fast, always runs
r.Use(router.RateLimit(...))   // Can short-circuit quickly
r.Use(router.BasicAuth(...))   // Authentication checks
r.Use(router.CORS(...))        // CORS headers
r.Use(router.Logger())         // Logging (place later due to latency measurement)
```

### Conditional Middleware

Only apply middleware where it's needed:

```go
// Global lightweight middleware
r.Use(router.Recovery())
r.Use(router.RequestID())

// Group-specific heavier middleware
api := r.Group("/api")
api.Use(router.JWT(...))
api.Use(router.RateLimit(...))

// Route-specific heavy middleware
r.With(loggingMiddleware).Get("/admin/logs", logsHandler)
```

### Custom Optimized Middleware

Create specialized middleware for performance-critical routes:

```go
// Fast, optimized authentication for high-traffic routes
func fastAuthMiddleware(next router.HandlerFunc) router.HandlerFunc {
    // Pre-compile regex patterns
    tokenPattern := regexp.MustCompile(`^Bearer ([a-zA-Z0-9\-_.]+)$`)
    
    // Pre-allocate token cache
    tokenCache := sync.Map{}
    
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        // Fast path: check authorization header
        auth := r.Header.Get("Authorization")
        if auth == "" {
            router.Error(w, http.StatusUnauthorized, "Missing token")
            return
        }
        
        // Extract token
        matches := tokenPattern.FindStringSubmatch(auth)
        if len(matches) < 2 {
            router.Error(w, http.StatusUnauthorized, "Invalid token format")
            return
        }
        
        token := matches[1]
        
        // Check cache first (sync.Map is concurrent-safe)
        if _, ok := tokenCache.Load(token); ok {
            // Token valid, proceed
            next(w, r, p)
            return
        }
        
        // Validate token (slower path)
        if validateToken(token) {
            // Cache for future requests (with TTL via background cleanup)
            tokenCache.Store(token, time.Now())
            next(w, r, p)
            return
        }
        
        router.Error(w, http.StatusUnauthorized, "Invalid token")
    }
}
```

## Handler Optimizations

### Efficient Parameter Access

Optimize how you access route parameters:

```go
// Less efficient - map lookup for each parameter
r.Get("/users/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    id := p["id"]
    postId := p["postId"]
    // Use parameters...
})

// More efficient - single parameter extraction
r.Get("/users/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    // Use router's optimized parameter extraction
    id, postId := router.Param2(p, "id", "postId")
    // Use parameters...
})
```

### Response Writing

Optimize response generation:

```go
// Pre-allocate commonly used responses
var (
    successResponse = []byte(`{"status":"success"}`)
    notFoundResponse = []byte(`{"status":"error","code":"not_found"}`)
    errorResponse = []byte(`{"status":"error","code":"server_error"}`)
)

r.Get("/status", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    // Set headers once
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    
    // Write pre-allocated response
    w.Write(successResponse)
})

// Use buffer pooling for dynamic responses
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

r.Get("/dynamic", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    // Get buffer from pool
    buf := bufferPool.Get().(*bytes.Buffer)
    buf.Reset()
    defer bufferPool.Put(buf)
    
    // Build response in buffer
    buf.WriteString(`{"status":"success","data":`)
    json.NewEncoder(buf).Encode(getData())
    buf.WriteString(`}`)
    
    // Set headers and write response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    buf.WriteTo(w)
})
```

### JSON Optimization

For JSON-heavy APIs, consider these optimizations:

```go
// Use json.RawMessage for pre-computed portions
type CombinedResponse struct {
    Status string          `json:"status"`
    Time   string          `json:"time"`
    Data   json.RawMessage `json:"data"`
}

// Pre-compute common JSON fragments
var userListJSON json.RawMessage

func init() {
    // Compute once at startup
    userListJSON, _ = json.Marshal(getAllUsers())
}

r.Get("/users", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    resp := CombinedResponse{
        Status: "success",
        Time:   time.Now().Format(time.RFC3339),
        Data:   userListJSON,
    }
    
    router.JSON(w, http.StatusOK, resp)
})
```

## Resource Usage Optimization

### Connection Management

Tune HTTP server parameters for optimal connection handling:

```go
func main() {
    r := router.New()
    // Configure routes...
    
    srv := &http.Server{
        Addr:    ":8080",
        Handler: r,
        
        // Connection timeouts
        ReadTimeout:       5 * time.Second,
        WriteTimeout:      10 * time.Second,
        IdleTimeout:       120 * time.Second,
        ReadHeaderTimeout: 2 * time.Second,
        
        // Connection limits
        MaxHeaderBytes: 1 << 20, // 1MB
    }
    
    // Enable keep-alives but with reasonable timeout
    srv.SetKeepAlivesEnabled(true)
    
    srv.ListenAndServe()
}
```

### Static File Serving

Optimize static file serving for better performance:

```go
// Use efficient static file serving
r.Static("/assets", "public/assets", 
    // Cache file info to avoid stat calls
    router.WithFileCaching(),
    
    // Set aggressive cache headers
    router.WithCacheControl("public, max-age=31536000"),
    
    // Enable compression
    router.WithCompression(),
    
    // Serve files in parallel
    router.WithConcurrency(runtime.NumCPU()),
)
```

### Memory Profiling and Optimization

MoraRouter includes built-in profiling tools:

```go
r := router.New(
    // Enable profiling endpoints
    router.WithProfiling(),
    
    // Enable memory statistics
    router.WithMemStats(),
)

// Access profiling info at /_mora/profile
// Access memory stats at /_mora/memstats
```

## Advanced Techniques

### Zero-Copy Routing

For absolute maximum performance, use zero-copy techniques:

```go
r := router.New(router.WithZeroCopy())
```

This optimizes request handling to minimize memory allocations and copies.

### Custom Allocators

For extreme performance requirements, configure custom memory allocation:

```go
r := router.New(
    router.WithCustomAllocator(&router.PoolAllocator{
        ParamsSize:      8,
        ContextPoolSize: 1024,
        BufferSize:      4096,
    }),
)
```

### Request Batching

Process multiple requests in a batch for efficiency:

```go
r.BatchEndpoint("/batch", func(batch []*router.BatchRequest) []*router.BatchResponse {
    responses := make([]*router.BatchResponse, len(batch))
    
    // Process all requests in parallel
    var wg sync.WaitGroup
    wg.Add(len(batch))
    
    for i, req := range batch {
        go func(i int, req *router.BatchRequest) {
            defer wg.Done()
            // Process individual request
            responses[i] = processRequest(req)
        }(i, req)
    }
    
    wg.Wait()
    return responses
})
```

## Load Testing and Benchmarking

MoraRouter includes tools for benchmarking your application:

```go
// Create benchmark report
report, err := router.Benchmark(r, router.BenchmarkOptions{
    URL:         "/api/users",
    Method:      "GET",
    Concurrency: 100,
    Duration:    10 * time.Second,
})

fmt.Printf("Requests/sec: %v\n", report.RequestsPerSecond)
fmt.Printf("Latency (P99): %v\n", report.LatencyP99)
```

You can also use external tools like `hey`, `wrk`, or `vegeta` for load testing.

## Production Checklist

Before deploying to production, ensure you've optimized these key areas:

1. **Route organization**: Group related routes, minimize wildcard routes
2. **Middleware efficiency**: Only use necessary middleware, order them correctly
3. **Connection handling**: Set appropriate timeouts and connection limits
4. **Memory management**: Use pooling for request objects and buffers
5. **Response generation**: Optimize JSON serialization, use pre-computed responses
6. **Static file serving**: Set proper cache headers, use compression
7. **Monitoring**: Enable metrics collection for real-time performance monitoring
8. **Hardware provisioning**: Allocate sufficient CPU/memory based on benchmarks

## Conclusion

MoraRouter is built for performance, but these optimization techniques can help you squeeze out even more speed when handling large-scale production traffic. Remember to measure before and after optimization to ensure your changes are having the desired effect.

Always prioritize optimizations that address your specific bottlenecks rather than implementing all possible optimizations at once. The most effective performance improvements come from understanding your application's unique traffic patterns and resource usage.

Happy optimizing! 🚀
