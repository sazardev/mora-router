# Testing with MoraRouter

MoraRouter comes with built-in utilities that make it easy to test your routes, handlers, and middleware. This guide covers how to write effective tests for your MoraRouter applications.

## Testing Utilities

MoraRouter provides a `TestClient` that simplifies API testing by removing the need to manually set up HTTP servers and clients:

```go
// Import testing package
import (
    "testing"
    "github.com/yourusername/mora-router/router"
)

func TestUserAPI(t *testing.T) {
    // Create a router for testing
    r := router.New()
    
    // Add routes
    r.Get("/users/:id", getUserHandler)
    
    // Create a test client
    client := router.NewTestClient(r)
    
    // Make a test request
    resp := client.Get("/users/123")
    
    // Assert the response
    if !resp.IsOK() {
        t.Fatalf("Expected OK response, got %d", resp.StatusCode)
    }
    
    // Read the response body as JSON
    var user User
    resp.DecodeJSON(&user)
    
    // Make assertions on the response content
    if user.ID != "123" {
        t.Errorf("Expected user ID 123, got %s", user.ID)
    }
}
```

## TestResponse Helpers

The `TestResponse` returned by the client has several helpers to simplify assertions:

```go
// Status code helpers
if !resp.IsOK() { /* Status is not 200 */ }
if !resp.IsCreated() { /* Status is not 201 */ }
if !resp.IsNoContent() { /* Status is not 204 */ }
if resp.IsClientError() { /* Status is 4xx */ }
if resp.IsServerError() { /* Status is 5xx */ }

// Header helpers
contentType := resp.Header.Get("Content-Type")
if !resp.HasHeader("X-Custom-Header") { /* Header missing */ }

// Body helpers
body := resp.Body()  // Raw body as bytes
text := resp.Text()  // Body as string

// Decode helpers
var data map[string]interface{}
resp.DecodeJSON(&data)  // Parse JSON body

var xmlData MyXMLStruct
resp.DecodeXML(&xmlData)  // Parse XML body
```

## Testing Different HTTP Methods

TestClient supports all HTTP methods:

```go
// GET request
resp := client.Get("/users")

// GET with query parameters
resp := client.GetQuery("/users", map[string]string{
    "page": "2",
    "limit": "10",
})

// POST with JSON body
resp := client.PostJSON("/users", User{
    Name: "Alice",
    Email: "alice@example.com",
})

// PUT with JSON body
resp := client.PutJSON("/users/123", User{
    Name: "Alice Updated",
    Email: "alice.updated@example.com",
})

// PATCH with JSON body
resp := client.PatchJSON("/users/123", map[string]string{
    "name": "Alice Changed",
})

// DELETE request
resp := client.Delete("/users/123")

// Custom method
resp := client.DoJSON("OPTIONS", "/users", nil)
```

## Testing Form Submissions

For testing form submissions and file uploads:

```go
// POST with form data
resp := client.PostForm("/contact", map[string]string{
    "name": "Alice",
    "email": "alice@example.com",
    "message": "Hello, world!",
})

// File upload
file := router.TestFile{
    Name:     "avatar.png",
    Content:  []byte("fake image content"),
    Filename: "profile.png",
}

resp := client.PostMultipart("/profile", map[string]string{
    "name": "Alice",
}, map[string]router.TestFile{
    "avatar": file,
})
```

## Setting Headers

You can set custom headers for your test requests:

```go
// Set headers for a single request
resp := client.GetWithHeaders("/api/data", map[string]string{
    "Authorization": "Bearer token123",
    "Accept-Language": "es-ES",
})

// Set default headers for all requests
client.SetDefaultHeaders(map[string]string{
    "Authorization": "Bearer token123",
    "X-API-Key": "abc123",
})

// Now all requests will include these headers
resp := client.Get("/protected-resource")
```

## Testing Authenticated Routes

For testing routes that require authentication:

```go
// Set auth token for all requests
client.WithAuth("Bearer token123")

// Make authenticated requests
resp := client.Get("/protected/resource")

// Set a different authentication scheme
client.WithBasicAuth("username", "password")

// Test with basic auth
resp := client.Get("/protected/resource")
```

## Testing Context Values

If your handlers use context values:

```go
// Set context values for all requests
client.WithContextValue("user_id", "123")
client.WithContextValue("is_admin", true)

// Make requests with context values
resp := client.Get("/dashboard")
```

## Testing Route Parameters

Test how your handlers extract and use route parameters:

```go
func TestRouteParameters(t *testing.T) {
    r := router.New()
    
    // Route with parameters
    r.Get("/users/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request, p router.Params) {
        userId := p["id"]
        postId := p["postId"]
        
        router.JSON(w, http.StatusOK, map[string]string{
            "userId": userId,
            "postId": postId,
        })
    })
    
    client := router.NewTestClient(r)
    
    // Test with parameters
    resp := client.Get("/users/123/posts/456")
    
    // Assert response
    var data map[string]string
    resp.DecodeJSON(&data)
    
    if data["userId"] != "123" || data["postId"] != "456" {
        t.Errorf("Parameter extraction failed: %v", data)
    }
}
```

## Testing Middleware

Test middleware by attaching it to test routes:

```go
func TestAuthMiddleware(t *testing.T) {
    r := router.New()
    
    // Add middleware to a route
    r.With(AuthMiddleware).Get("/protected", func(w http.ResponseWriter, r *http.Request, p router.Params) {
        router.JSON(w, http.StatusOK, map[string]bool{"authenticated": true})
    })
    
    client := router.NewTestClient(r)
    
    // Test without auth header (should fail)
    resp := client.Get("/protected")
    if !resp.IsForbidden() && !resp.IsUnauthorized() {
        t.Error("Expected 401 or 403 for missing auth")
    }
    
    // Test with auth header (should succeed)
    resp = client.GetWithHeaders("/protected", map[string]string{
        "Authorization": "Bearer valid-token",
    })
    
    if !resp.IsOK() {
        t.Error("Expected 200 OK with valid auth token")
    }
}
```

## Testing Resource Controllers

Test all operations of a resource controller:

```go
func TestUserController(t *testing.T) {
    r := router.New()
    
    // Register a resource controller
    r.Resource("/users", UserController{})
    
    client := router.NewTestClient(r)
    
    // Test index action
    indexResp := client.Get("/users")
    if !indexResp.IsOK() {
        t.Error("Index action failed")
    }
    
    // Test show action
    showResp := client.Get("/users/123")
    if !showResp.IsOK() {
        t.Error("Show action failed")
    }
    
    // Test create action
    createResp := client.PostJSON("/users", map[string]string{
        "name": "New User",
        "email": "new@example.com",
    })
    if !createResp.IsCreated() {
        t.Error("Create action failed")
    }
    
    // Test update action
    updateResp := client.PutJSON("/users/123", map[string]string{
        "name": "Updated User",
    })
    if !updateResp.IsOK() {
        t.Error("Update action failed")
    }
    
    // Test delete action
    deleteResp := client.Delete("/users/123")
    if !deleteResp.IsNoContent() {
        t.Error("Delete action failed")
    }
}
```

## Table-Driven Tests

Use table-driven tests for testing multiple routes and scenarios:

```go
func TestAPIEndpoints(t *testing.T) {
    r := router.New()
    setupRoutes(r)  // Your function to set up all routes
    
    client := router.NewTestClient(r)
    
    tests := []struct {
        name           string
        method         string
        path           string
        body           interface{}
        expectedStatus int
        expectedJSON   map[string]interface{}
    }{
        {
            name:           "Get all users",
            method:         "GET",
            path:           "/api/users",
            expectedStatus: 200,
        },
        {
            name:           "Get single user",
            method:         "GET",
            path:           "/api/users/1",
            expectedStatus: 200,
        },
        {
            name:           "Create user",
            method:         "POST",
            path:           "/api/users",
            body:           map[string]string{"name": "Test User", "email": "test@example.com"},
            expectedStatus: 201,
        },
        {
            name:           "Update user",
            method:         "PUT",
            path:           "/api/users/1",
            body:           map[string]string{"name": "Updated User"},
            expectedStatus: 200,
        },
        {
            name:           "Delete user",
            method:         "DELETE",
            path:           "/api/users/1",
            expectedStatus: 204,
        },
        {
            name:           "Not found",
            method:         "GET",
            path:           "/api/unknown",
            expectedStatus: 404,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var resp *router.TestResponse
            
            switch tt.method {
            case "GET":
                resp = client.Get(tt.path)
            case "POST":
                resp = client.PostJSON(tt.path, tt.body)
            case "PUT":
                resp = client.PutJSON(tt.path, tt.body)
            case "DELETE":
                resp = client.Delete(tt.path)
            default:
                t.Fatalf("Unsupported method: %s", tt.method)
            }
            
            if resp.StatusCode != tt.expectedStatus {
                t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
            }
            
            if tt.expectedJSON != nil {
                var data map[string]interface{}
                resp.DecodeJSON(&data)
                
                // Check expected JSON fields
                for k, v := range tt.expectedJSON {
                    if data[k] != v {
                        t.Errorf("Expected %s=%v, got %v", k, v, data[k])
                    }
                }
            }
        })
    }
}
```

## Testing Validation and Error Responses

Test validation logic and error responses:

```go
func TestValidation(t *testing.T) {
    r := router.New()
    
    // Route with validation
    r.Post("/users", router.BindJSON(func(w http.ResponseWriter, r *http.Request, p router.Params, input struct {
        Name  string `json:"name" validate:"required"`
        Email string `json:"email" validate:"required,email"`
        Age   int    `json:"age" validate:"min=18"`
    }) {
        router.JSON(w, http.StatusCreated, input)
    }))
    
    client := router.NewTestClient(r)
    
    // Test valid input
    validResp := client.PostJSON("/users", map[string]interface{}{
        "name": "Valid User",
        "email": "valid@example.com",
        "age": 25,
    })
    
    if !validResp.IsCreated() {
        t.Error("Valid input should return 201")
    }
    
    // Test invalid email
    invalidEmailResp := client.PostJSON("/users", map[string]interface{}{
        "name": "Invalid User",
        "email": "not-an-email",
        "age": 25,
    })
    
    if !invalidEmailResp.IsBadRequest() {
        t.Error("Invalid email should return 400")
    }
    
    // Test missing required field
    missingFieldResp := client.PostJSON("/users", map[string]interface{}{
        "email": "test@example.com",
        "age": 25,
    })
    
    if !missingFieldResp.IsBadRequest() {
        t.Error("Missing required field should return 400")
    }
    
    // Test age validation
    underageResp := client.PostJSON("/users", map[string]interface{}{
        "name": "Young User",
        "email": "young@example.com",
        "age": 16,
    })
    
    if !underageResp.IsBadRequest() {
        t.Error("Underage user should return 400")
    }
}
```

## Testing File Responses

Test handlers that serve files:

```go
func TestFileDownload(t *testing.T) {
    r := router.New()
    
    // Create a temporary test file
    content := []byte("test file content")
    tmpFile, err := os.CreateTemp("", "test-*.txt")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpFile.Name())
    
    if _, err := tmpFile.Write(content); err != nil {
        t.Fatal(err)
    }
    tmpFile.Close()
    
    // Route that serves the file
    r.Get("/download", func(w http.ResponseWriter, r *http.Request, p router.Params) {
        router.FileDownload(w, r, tmpFile.Name())
    })
    
    client := router.NewTestClient(r)
    
    // Test file download
    resp := client.Get("/download")
    
    // Check status
    if !resp.IsOK() {
        t.Error("Expected 200 OK status")
    }
    
    // Check content
    if string(resp.Body()) != string(content) {
        t.Error("File content does not match")
    }
}
```

## Testing with Database Mock

Test handlers that interact with a database:

```go
// Mock database interface
type UserRepository interface {
    FindById(id string) (*User, error)
    Create(user User) error
}

// Mock implementation
type MockUserRepository struct {
    users map[string]User
}

func NewMockUserRepository() *MockUserRepository {
    return &MockUserRepository{
        users: map[string]User{
            "123": {ID: "123", Name: "Test User", Email: "test@example.com"},
        },
    }
}

func (m *MockUserRepository) FindById(id string) (*User, error) {
    user, ok := m.users[id]
    if !ok {
        return nil, errors.New("user not found")
    }
    return &user, nil
}

func (m *MockUserRepository) Create(user User) error {
    m.users[user.ID] = user
    return nil
}

// Handler that uses the repository
func getUserHandler(repo UserRepository) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        id := p["id"]
        user, err := repo.FindById(id)
        
        if err != nil {
            router.Error(w, http.StatusNotFound, "User not found")
            return
        }
        
        router.JSON(w, http.StatusOK, user)
    }
}

// Test with mock repository
func TestUserHandler(t *testing.T) {
    r := router.New()
    
    // Create mock repository
    repo := NewMockUserRepository()
    
    // Register route with handler that uses the repository
    r.Get("/users/:id", getUserHandler(repo))
    
    client := router.NewTestClient(r)
    
    // Test existing user
    resp := client.Get("/users/123")
    
    if !resp.IsOK() {
        t.Error("Expected 200 OK for existing user")
    }
    
    var user User
    resp.DecodeJSON(&user)
    
    if user.ID != "123" || user.Name != "Test User" {
        t.Errorf("Incorrect user data: %+v", user)
    }
    
    // Test non-existing user
    notFoundResp := client.Get("/users/999")
    
    if !notFoundResp.IsNotFound() {
        t.Error("Expected 404 Not Found for non-existing user")
    }
}
```

## Integration Testing

For full integration tests that start a real server:

```go
func TestIntegration(t *testing.T) {
    // Skip if short test flag is used
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Create and configure router
    r := router.New(router.WithLogging(), router.WithRecovery())
    setupRoutes(r)  // Your function to set up all routes
    
    // Start server in a goroutine
    server := &http.Server{
        Addr:    "127.0.0.1:8081",  // Use a port unlikely to be in use
        Handler: r,
    }
    
    go func() {
        server.ListenAndServe()
    }()
    
    // Give the server a moment to start
    time.Sleep(100 * time.Millisecond)
    
    // Create HTTP client
    client := &http.Client{
        Timeout: 5 * time.Second,
    }
    
    // Make real HTTP request
    resp, err := client.Get("http://127.0.0.1:8081/api/health")
    if err != nil {
        t.Fatalf("Failed to make request: %v", err)
    }
    defer resp.Body.Close()
    
    // Assert response
    if resp.StatusCode != 200 {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
    
    // Read response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        t.Fatalf("Failed to read response body: %v", err)
    }
    
    // Assert response content
    var data map[string]interface{}
    json.Unmarshal(body, &data)
    
    status, ok := data["status"]
    if !ok || status != "ok" {
        t.Errorf("Expected status 'ok', got %v", status)
    }
    
    // Shutdown server
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    server.Shutdown(ctx)
}
```

## Conclusion

MoraRouter's testing utilities make it easy to write comprehensive tests for your API. By using the `TestClient` and `TestResponse` helpers, you can test routes, handlers, middleware, and controllers without the overhead of setting up real HTTP servers and clients.

For more examples, check out the [example projects](examples.md) in the repository.

### Best Practices

1. Use table-driven tests for testing multiple scenarios
2. Test both happy paths and error cases
3. Mock external dependencies like databases
4. Use the TestClient for unit tests and real HTTP clients for integration tests
5. Test your validation logic thoroughly
6. Test all routes in your API
7. Group related tests in test suites

Remember that good test coverage is essential for maintaining a stable API as your project grows.
