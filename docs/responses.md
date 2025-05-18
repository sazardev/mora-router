# Response Handling in MoraRouter

MoraRouter provides a variety of tools for sending well-formatted responses in different formats. This guide covers JSON, XML, HTML templates, file downloads, and other response types.

## JSON Responses

JSON is the most common format for API responses. MoraRouter makes it easy to send JSON:

```go
// Basic JSON response
r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    user := User{
        ID:    p["id"],
        Name:  "John Doe",
        Email: "john@example.com",
    }
    
    router.JSON(w, http.StatusOK, user)
})
```

This automatically:
- Sets Content-Type to application/json
- Marshals your data to JSON
- Sets the appropriate status code
- Handles errors during marshaling

### Pretty Printed JSON

For debugging or development environments, you can send pretty-printed JSON:

```go
router.PrettyJSON(w, http.StatusOK, complexObject)
```

### JSONP Responses

For cross-domain requests that need JSONP:

```go
r.Get("/api/data", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    data := map[string]interface{}{
        "message": "Hello World",
        "count": 42,
    }
    
    // Get callback from query parameter
    callback := r.URL.Query().Get("callback")
    
    router.JSONP(w, http.StatusOK, callback, data)
})
```

## XML Responses

XML responses are handled similarly to JSON:

```go
// XML response
r.Get("/products/:id", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    product := Product{
        ID:    p["id"],
        Name:  "Widget",
        Price: 29.99,
    }
    
    router.XML(w, http.StatusOK, product)
})
```

This automatically:
- Sets Content-Type to application/xml
- Marshals your data to XML
- Sets the appropriate status code
- Handles errors during marshaling

## HTML Responses

MoraRouter provides template rendering for HTML responses:

```go
// Setup templates
r := router.New(router.WithTemplates("templates"))

// Render a template
r.Get("/", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    data := map[string]interface{}{
        "Title":   "Welcome",
        "Message": "Hello from MoraRouter",
    }
    
    router.RenderTemplate(w, r, "home.html", data)
})
```

### Advanced Template Configuration

```go
// Create a template manager
tm := router.NewTemplateManager("templates")

// Configure layout and partials
tm.WithLayout("layouts/main.html")
   .WithPartials("partials/header.html", "partials/footer.html")
   .WithFuncs(template.FuncMap{
        "formatDate": func(t time.Time) string {
            return t.Format("Jan 02, 2006")
        },
   })

// Use it with the router
r := router.New(router.WithTemplateManager(tm))
```

## CSV Responses

For tabular data, CSV responses are useful:

```go
// CSV response
r.Get("/exports/users", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    users := []User{
        {ID: "1", Name: "Alice", Email: "alice@example.com"},
        {ID: "2", Name: "Bob", Email: "bob@example.com"},
    }
    
    router.CSV(w, http.StatusOK, users)
})
```

This automatically:
- Sets Content-Type to text/csv
- Converts your slice of structs to CSV rows
- Uses struct field names or csv tags for headers

### Custom CSV Headers

```go
type User struct {
    ID    string `csv:"User ID"`    // Custom CSV header
    Name  string `csv:"Full Name"`  // Custom CSV header
    Email string `csv:"Email Address"` // Custom CSV header
}

// CSV with custom headers
router.CSVWithHeaders(w, http.StatusOK, users, []string{"User ID", "Full Name", "Email Address"})
```

## File Downloads

For file downloads:

```go
// File download
r.Get("/downloads/:filename", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    filePath := "./files/" + p["filename"]
    
    router.FileDownload(w, r, filePath)
})
```

To force the browser to download rather than display:

```go
// Force download with custom filename
r.Get("/reports/:id", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    report := generateReport(p["id"])
    
    // Save report to temp file
    tempFile := fmt.Sprintf("./temp/report-%s.pdf", p["id"])
    saveReport(report, tempFile)
    
    // Force download with custom filename
    router.FileDownload(w, r, tempFile)
    router.ForceDownload(w, "report.pdf")
})
```

## Static File Serving

For serving static files:

```go
// Serve static files from a directory
r.Static("/assets", "./public")

// Serve a single file
r.StaticFile("/favicon.ico", "./public/favicon.ico")

// Serve a SPA (Single Page Application)
r.SPA("/app", "./dist", "index.html")
```

## Error Responses

For error handling:

```go
// Simple error response
r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    user, err := fetchUser(p["id"])
    
    if err == ErrNotFound {
        router.Error(w, http.StatusNotFound, "User not found")
        return
    }
    
    if err != nil {
        router.Error(w, http.StatusInternalServerError, "Failed to fetch user")
        return
    }
    
    router.JSON(w, http.StatusOK, user)
})
```

For more detailed errors:

```go
// Detailed error response
type ErrorResponse struct {
    Status  int    `json:"status"`
    Message string `json:"message"`
    Code    string `json:"code,omitempty"`
    Details any    `json:"details,omitempty"`
}

r.Get("/resources/:id", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    resource, err := fetchResource(p["id"])
    
    if err == ErrNotFound {
        router.JSON(w, http.StatusNotFound, ErrorResponse{
            Status:  404,
            Message: "Resource not found",
            Code:    "RESOURCE_NOT_FOUND",
        })
        return
    }
    
    if err != nil {
        router.JSON(w, http.StatusInternalServerError, ErrorResponse{
            Status:  500,
            Message: "Internal server error",
            Code:    "INTERNAL_ERROR",
            Details: map[string]string{
                "error": err.Error(),
            },
        })
        return
    }
    
    router.JSON(w, http.StatusOK, resource)
})
```

## Redirects

For redirects:

```go
// Simple redirect
r.Get("/old-path", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    router.Redirect(w, r, "/new-path", http.StatusSeeOther)
})

// Redirect with named route
r.Get("/users/profile", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    userID := getUserFromContext(r.Context())
    url, _ := r.URL("user.profile", userID)
    router.Redirect(w, r, url, http.StatusFound)
})
```

## Content Negotiation

For services that need to serve multiple formats based on Accept header:

```go
// Content negotiation
r.Get("/api/data", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    data := struct {
        Message string `json:"message" xml:"message"`
        Count   int    `json:"count" xml:"count"`
    }{
        Message: "Hello World",
        Count:   42,
    }
    
    render := router.NewRender()
    render.Negotiate(w, r, http.StatusOK, data)
})
```

This will:
- Check the Accept header
- Render as JSON, XML, or plain text based on the header
- Default to JSON if no match is found

## Custom Content Types

For custom content types:

```go
// Custom content type
r.Get("/special-data", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    data := "CUSTOM:DATA:FORMAT:123:456:789"
    
    w.Header().Set("Content-Type", "application/x-custom-format")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(data))
})
```

## Streaming Responses

For long-running processes or large data sets:

```go
// Streaming response
r.Get("/events", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    // Set headers for streaming
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    // Get a flusher if available
    flusher, ok := w.(http.Flusher)
    if !ok {
        router.Error(w, http.StatusInternalServerError, "Streaming not supported")
        return
    }
    
    // Send events periodically
    for i := 0; i < 10; i++ {
        fmt.Fprintf(w, "data: Event %d\n\n", i)
        flusher.Flush()
        time.Sleep(1 * time.Second)
    }
})
```

## WebSocket Responses

For real-time bidirectional communication:

```go
r := router.New(router.WithGorillaWebSocket())

r.Get("/ws", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    // Handle WebSocket connection
    conn, err := router.UpgradeWebSocket(w, r)
    if err != nil {
        return
    }
    defer conn.Close()
    
    // Echo all messages
    for {
        messageType, p, err := conn.ReadMessage()
        if err != nil {
            break
        }
        if err := conn.WriteMessage(messageType, p); err != nil {
            break
        }
    }
})
```

## Custom Response Writer

For advanced cases where you need to wrap the response writer:

```go
// Custom response writer
r.Get("/logged-response", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    // Create a custom response writer that logs all writes
    loggedWriter := &loggingResponseWriter{
        ResponseWriter: w,
        body:           new(bytes.Buffer),
    }
    
    // Use the custom writer
    router.JSON(loggedWriter, http.StatusOK, map[string]string{
        "message": "This response will be logged",
    })
    
    // Log the response
    log.Printf("Response body: %s", loggedWriter.body.String())
})

type loggingResponseWriter struct {
    http.ResponseWriter
    body *bytes.Buffer
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
    // Write to the buffer for logging
    w.body.Write(b)
    
    // Write to the original writer
    return w.ResponseWriter.Write(b)
}
```

## Response Helpers

MoraRouter provides several helper functions for common responses:

```go
// Success response with default message
router.Success(w, "Operation completed successfully")

// Created response with location header
router.Created(w, "/resources/123", map[string]string{"id": "123"})

// NoContent response
router.NoContent(w)

// BadRequest with message
router.BadRequest(w, "Invalid parameters")

// Unauthorized with realm
router.Unauthorized(w, "api")

// Forbidden with message
router.Forbidden(w, "Insufficient permissions")

// NotFound with message
router.NotFound(w, "Resource not found")

// ServerError with message
router.ServerError(w, "An unexpected error occurred")
```

## Response Builder Pattern

For complex response building:

```go
// Response builder
response := router.NewResponse(w)

// Build a complex response
response.
    Status(http.StatusOK).
    Header("X-Custom-Header", "Value").
    Cookie(&http.Cookie{Name: "session", Value: "abc123", MaxAge: 3600}).
    JSON(map[string]interface{}{
        "user": user,
        "token": token,
    })
```

## Conclusion

MoraRouter provides a comprehensive set of response tools to handle various output formats and patterns. By using these built-in methods, you can create consistent, well-formatted responses for your API clients.

Next, check out [Templates](templates.md) for more details on HTML rendering options.
