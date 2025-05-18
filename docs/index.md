# MoraRouter Documentation

<div style="text-align: center;">
  <img src="assets/mora-logo.png" alt="MoraRouter Logo" width="200" />
  <p><em>Ultra-powerful HTTP router for Go inspired by Django</em></p>
</div>

## Welcome to MoraRouter

MoraRouter is a feature-rich HTTP router for Go applications, designed to make building web applications and RESTful APIs a breeze. Inspired by Django's powerful routing capabilities but built from the ground up for Go's performance and concurrency model.

## 🚀 Getting Started

* [Installation Guide](installation.md) - Get up and running quickly
* [Quick Start](quickstart.md) - Basic usage examples
* [Core Concepts](core-concepts.md) - Understand MoraRouter's architecture

## 📚 Main Features

### Routing
* [Basic Routing](routing.md) - URLs, patterns, and HTTP methods
* [Route Parameters](routing.md#route-parameters) - Dynamic segments and pattern matching
* [Route Groups](groups.md) - Organize routes with shared prefixes
* [API Versioning](api-versioning.md) - Version your APIs elegantly
* [Route Macros](macros.md) - Reusable routing patterns

### Handling Requests
* [Middleware](middleware.md) - Request/response processing pipeline
* [Controllers](controllers.md) - Structured request handling
* [RESTful Resources](controllers.md#restful-resources) - Quickly build CRUD APIs
* [Data Binding](data-binding.md) - Parse and validate request data
* [WebSockets](websockets.md) - Real-time bidirectional communication

### Generating Responses
* [Response Helpers](responses.md) - Simplify response generation
* [Content Negotiation](responses.md#content-negotiation) - Serve different formats
* [Templates](templates.md) - HTML templating with Go's template package
* [Static Files](routing.md#static-files) - Serve assets, SPA applications

## 🛠️ Advanced Topics

* [Performance Optimization](performance.md) - Tune for maximum throughput
* [Hot Reload](hot-reload.md) - Update routes without restarting
* [Testing](testing.md) - Test your router and handlers
* [API Reference](api-reference.md) - Detailed function and type documentation

## 🤝 Community

* [Contributing Guide](contributing.md) - Help improve MoraRouter
* [FAQ](faq.md) - Common questions and answers

## Why MoraRouter?

When it comes to Go web development, there are several routing options available. So why choose MoraRouter?

- **Rich Feature Set**: MoraRouter provides everything you need out of the box - from advanced routing patterns to middleware, validation, and testing tools
- **Developer Experience**: Designed to reduce boilerplate and make common tasks simple
- **Flexibility**: Configurable through options and middleware to match your exact needs
- **Performance**: Optimized path matching algorithm that stays fast even with complex routes
- **Modern API Design**: Intuitive API that feels natural to Go developers
- **Production Ready**: Built with real-world applications in mind

## Getting Started

```bash
go get -u github.com/yourusername/mora-router
```

Basic example:

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
    
    // Add a simple route
    r.Get("/hello/:name", func(w http.ResponseWriter, req *http.Request, p router.Params) {
        router.JSON(w, http.StatusOK, map[string]interface{}{
            "message": "Hello, " + p["name"] + "!",
        })
    })
    
    // Start the server
    log.Println("Server started on :8080")
    http.ListenAndServe(":8080", r)
}
```

## Documentation Contents

- **[Quick Start](quickstart.md)**: Get up and running in minutes
- **[Core Concepts](core-concepts.md)**: Understanding MoraRouter's architecture
- **[Routing](routing.md)**: Route definition, parameters, and patterns
- **[Middleware](middleware.md)**: Built-in and custom middleware
- **[Controllers](controllers.md)**: Working with resource controllers
- **[Data Binding](data-binding.md)**: Request validation and parsing
- **[Responses](responses.md)**: Working with multiple response formats
- **[Templates](templates.md)**: Template rendering and management
- **[Groups & Nesting](groups.md)**: Organizing routes with groups
- **[Testing](testing.md)**: Utilities for testing your API
- **[API Reference](api-reference.md)**: Complete API documentation
- **[Deployment](deployment.md)**: Production deployment guides
- **[Examples](examples.md)**: Real-world usage examples
- **[FAQ](faq.md)**: Frequently asked questions
- **[Contributing](contributing.md)**: How to contribute to MoraRouter

## License

MoraRouter is available under the MIT license. See the LICENSE file for more information.
