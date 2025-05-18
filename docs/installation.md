# Installation Guide

Welcome to the MoraRouter installation guide! Getting started with MoraRouter is quick and easy, allowing you to build powerful HTTP routing applications in Go with minimal setup.

## System Requirements

- Go 1.18 or higher (recommended: Go 1.24+)
- No external dependencies required for core functionality

## Basic Installation

To install MoraRouter in your Go project, use the standard `go get` command:

```bash
go get -u github.com/yourusername/mora-router
```

This will download and install the latest version of MoraRouter in your Go modules cache and add it to your project's dependencies.

## Verifying Installation

After installing, you can verify your installation by creating a simple application:

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    
    "github.com/yourusername/mora-router/router"
)

func main() {
    r := router.New()
    
    r.Get("/hello", func(w http.ResponseWriter, req *http.Request, p router.Params) {
        fmt.Fprintf(w, "Hello from MoraRouter!")
    })
    
    log.Println("Server started on http://localhost:8080")
    http.ListenAndServe(":8080", r)
}
```

Run the application with `go run main.go` and navigate to `http://localhost:8080/hello` in your browser. You should see the message "Hello from MoraRouter!".

## Installing with Specific Version

If you need a specific version of MoraRouter, you can specify it in your `go get` command:

```bash
go get -u github.com/yourusername/mora-router@v1.0.0
```

## Installing for Development

If you're planning to contribute to MoraRouter or need to modify the source code:

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/mora-router.git
   ```

2. Navigate to the project directory:
   ```bash
   cd mora-router
   ```

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Build and test:
   ```bash
   go build ./...
   go test ./...
   ```

## Optional Dependencies

While MoraRouter works great on its own, you can enhance it with these optional dependencies:

- **gorilla/websocket**: Required for WebSocket support
  ```bash
  go get -u github.com/gorilla/websocket
  ```

- **go-playground/validator**: Enhanced validation for data binding
  ```bash
  go get -u github.com/go-playground/validator/v10
  ```

- **swaggo/swag**: For OpenAPI/Swagger documentation generation
  ```bash
  go get -u github.com/swaggo/swag
  ```

## Troubleshooting

### Common Installation Issues

#### "Cannot find package" Error

If you see errors like `cannot find package "github.com/yourusername/mora-router/router"`, try:

1. Ensure your Go modules are enabled: `export GO111MODULE=on`
2. Clear your Go modules cache: `go clean -modcache`
3. Retry installation: `go get -u github.com/yourusername/mora-router`

#### Version Conflicts

If you encounter version conflicts with other dependencies, you may need to update your `go.mod` file manually:

```bash
go mod tidy
```

#### Compiler Errors

If you see compiler errors after installation, make sure you're using a compatible Go version (1.18+).

## Next Steps

Now that you've installed MoraRouter, head over to the [Quickstart Guide](quickstart.md) to create your first application!

Remember, MoraRouter is designed to be intuitive yet powerful - start simple and gradually explore its advanced features as you build more complex applications.

Happy routing! 🚀
