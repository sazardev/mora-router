# Hot Reload

MoraRouter's hot reload capability allows you to make changes to your routes without restarting your application. This powerful feature is particularly useful during development, but can also be valuable in certain production scenarios.

## Basic Usage

Enabling hot reload is simple:

```go
package main

import (
    "log"
    "net/http"
    "time"
    
    "github.com/yourusername/mora-router/router"
)

func main() {
    // Create router with hot reload enabled
    r := router.New(router.WithHotReload("routes.json", 5 * time.Second))
    
    // Set up some initial routes
    r.Get("/hello", func(w http.ResponseWriter, req *http.Request, p router.Params) {
        router.JSON(w, http.StatusOK, map[string]string{"message": "Hello, World!"})
    })
    
    log.Println("Server started on :8080")
    log.Println("Hot reload enabled with config from routes.json")
    http.ListenAndServe(":8080", r)
}
```

## Configuration File Format

The hot reload feature requires a configuration file in JSON format. Here's a basic example:

```json
{
  "routes": [
    {
      "method": "GET",
      "pattern": "/users",
      "handler": "userHandlers.List",
      "name": "users.list"
    },
    {
      "method": "GET",
      "pattern": "/users/:id",
      "handler": "userHandlers.Get",
      "name": "users.get"
    },
    {
      "method": "POST",
      "pattern": "/users",
      "handler": "userHandlers.Create",
      "name": "users.create"
    }
  ],
  "groups": {
    "api": {
      "prefix": "/api",
      "middleware": ["authMiddleware", "loggerMiddleware"]
    },
    "admin": {
      "prefix": "/admin",
      "middleware": ["authMiddleware", "adminMiddleware"]
    }
  },
  "handlers": {
    "userHandlers.List": "pkg.UserHandlers.List",
    "userHandlers.Get": "pkg.UserHandlers.Get",
    "userHandlers.Create": "pkg.UserHandlers.Create",
    "authMiddleware": "pkg.Middleware.Auth",
    "loggerMiddleware": "pkg.Middleware.Logger",
    "adminMiddleware": "pkg.Middleware.AdminOnly"
  }
}
```

### Configuration Structure

- **routes**: Array of route definitions
  - **method**: HTTP method (GET, POST, PUT, DELETE, etc.)
  - **pattern**: URL pattern with parameters
  - **handler**: Handler reference (defined in the "handlers" section)
  - **name**: Optional route name for URL generation
  - **middleware**: Optional array of middleware references

- **groups**: Map of route groups
  - **prefix**: Group URL prefix
  - **middleware**: Array of middleware references applied to all routes in the group

- **handlers**: Map of handler references to actual handler implementations
  - Keys are reference names used in routes and groups
  - Values are package paths to handler functions

## Handler Resolution

MoraRouter uses reflection to find your handler functions. The handler path in the configuration should be in the format:

```
package.Type.Method
```

For example, `pkg.UserHandlers.List` would resolve to:

```go
package pkg

type UserHandlers struct{}

func (h UserHandlers) List(w http.ResponseWriter, r *http.Request, p router.Params) {
    // Handler implementation
}
```

## Handler Registration

To make your handlers available for hot reload, register them with the router:

```go
package main

import (
    "github.com/yourusername/mora-router/router"
    "myapp/handlers"
)

func main() {
    r := router.New(router.WithHotReload("routes.json", 5 * time.Second))
    
    // Register handler struct
    userHandlers := handlers.UserHandlers{}
    r.RegisterHandlers("pkg.UserHandlers", userHandlers)
    
    // Register middleware functions
    r.RegisterHandler("pkg.Middleware.Auth", handlers.AuthMiddleware)
    r.RegisterHandler("pkg.Middleware.Logger", handlers.LoggerMiddleware)
    r.RegisterHandler("pkg.Middleware.AdminOnly", handlers.AdminOnlyMiddleware)
    
    // Start server
    http.ListenAndServe(":8080", r)
}
```

## Watching for Changes

When hot reload is enabled, MoraRouter watches the configuration file for changes. When a change is detected, it:

1. Loads the new configuration
2. Resolves all handler references
3. Updates the router's routes and groups
4. Preserves the existing middleware stack

Changes take effect immediately without restarting the server.

## Advanced Configuration

### Custom Watching Interval

You can customize how often MoraRouter checks for configuration changes:

```go
// Check every 10 seconds
r := router.New(router.WithHotReload("routes.json", 10 * time.Second))

// Check every 500 milliseconds (for development)
r := router.New(router.WithHotReload("routes.json", 500 * time.Millisecond))
```

### Multiple Configuration Files

You can split your configuration across multiple files:

```go
r := router.New(router.WithMultiHotReload([]string{
    "routes/api.json",
    "routes/admin.json",
    "routes/public.json",
}, 5 * time.Second))
```

### Dynamic Configuration Source

For advanced use cases, you can provide a custom configuration source:

```go
r := router.New(router.WithHotReloadSource(
    // Custom provider function
    func() ([]byte, error) {
        // Fetch configuration from database, API, etc.
        return fetchConfigFromDatabase()
    },
    5 * time.Second,
))
```

### Change Notifications

You can set up notifications for configuration changes:

```go
r := router.New(router.WithHotReload("routes.json", 5 * time.Second))

// Register a callback for configuration changes
r.OnConfigChange(func(oldConfig, newConfig *router.Config) {
    log.Println("Router configuration changed!")
    log.Printf("Added routes: %d, Removed routes: %d", 
        len(newConfig.AddedRoutes), len(newConfig.RemovedRoutes))
})
```

## Environment-Specific Configuration

You can load different configurations based on environment:

```go
configFile := "routes.dev.json"
if os.Getenv("ENVIRONMENT") == "production" {
    configFile = "routes.prod.json"
}

r := router.New(router.WithHotReload(configFile, 5 * time.Second))
```

## Hot Reload in Development vs. Production

### Development Environment

In development, hot reload offers:
- Rapid iteration without server restarts
- Easy experimentation with routes and middleware
- Configuration changes with immediate feedback

For development, use a shorter polling interval (500ms to 1s) for quicker feedback.

### Production Environment

In production, hot reload can be used for:
- Dynamic feature flags
- A/B testing different route configurations
- Adding new API endpoints without downtime
- Emergency route changes (e.g., temporarily disabling problematic endpoints)

For production, use a longer polling interval (30s to 5min) to reduce file system load, and consider implementing proper access controls for your configuration files.

## Route Inspection

To see which routes are currently loaded, you can use the router's inspector:

```go
r := router.New(
    router.WithHotReload("routes.json", 5 * time.Second),
    router.WithDebug(),
)
```

This enables the debug inspector at `/_mora/inspector`, which shows:
- Currently loaded routes
- When routes were last reloaded
- Which configuration file each route came from
- Any errors encountered during reloading

## Best Practices

1. **Validate configuration before deploying**: Use the `router.ValidateConfig()` utility to check for errors
2. **Version your configuration files**: Keep a history of changes for rollback if needed
3. **Use descriptive route names**: This makes it easier to track changes
4. **Structure groups logically**: Organize related routes in sensible groups
5. **Monitor for errors**: Check logs for handler resolution failures
6. **Consider security implications**: Protect your configuration files with proper permissions

## Security Considerations

Hot reloading configuration from files introduces some security considerations:

1. **File permissions**: Ensure configuration files can only be modified by authorized users
2. **Input validation**: Always validate the loaded configuration before applying it
3. **Rate limiting**: Set a reasonable polling interval to prevent excessive file system access
4. **Error handling**: Have fallback mechanisms for invalid configurations
5. **Audit trail**: Log configuration changes for security auditing

## Conclusion

Hot reload is a powerful feature that can significantly improve both development workflow and production flexibility. By dynamically reconfiguring your routes without server restarts, you can achieve faster iteration cycles and more resilient applications.

Whether you're rapidly prototyping a new API or managing a complex production environment, MoraRouter's hot reload capability gives you the tools to adapt your routing configuration on the fly.

Happy routing! 🔥
