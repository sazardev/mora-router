# Controllers in MoraRouter

Controllers in MoraRouter provide a structured way to organize your request handlers, especially for RESTful resources. They help you follow consistent patterns and reduce boilerplate code.

## What are Controllers?

A controller is a collection of related handler functions that typically operate on a single resource or concept in your application. For example, a `UserController` might handle all user-related operations like listing, creating, updating, and deleting users.

## The ResourceController Interface

MoraRouter defines a `ResourceController` interface that represents a standard RESTful controller:

```go
type ResourceController interface {
    Index(http.ResponseWriter, *http.Request, Params)
    Show(http.ResponseWriter, *http.Request, Params)
    Create(http.ResponseWriter, *http.Request, Params)
    Update(http.ResponseWriter, *http.Request, Params)
    Delete(http.ResponseWriter, *http.Request, Params)
}
```

These methods correspond to the standard CRUD operations:

- `Index` - Lists all resources (GET /resources)
- `Show` - Shows a single resource (GET /resources/:id)
- `Create` - Creates a new resource (POST /resources)
- `Update` - Updates a resource (PUT /resources/:id)
- `Delete` - Deletes a resource (DELETE /resources/:id)

## DefaultController

MoraRouter provides a `DefaultController` that implements the `ResourceController` interface with empty methods. You can embed this in your own controllers and override only the methods you need:

```go
// UserController with default implementations
type UserController struct {
    router.DefaultController
}

// Only override the methods you need
func (c UserController) Index(w http.ResponseWriter, r *http.Request, p router.Params) {
    users := []User{
        {ID: "1", Name: "Alice"},
        {ID: "2", Name: "Bob"},
    }
    router.JSON(w, http.StatusOK, users)
}

func (c UserController) Show(w http.ResponseWriter, r *http.Request, p router.Params) {
    id := p["id"]
    router.JSON(w, http.StatusOK, User{ID: id, Name: "User " + id})
}
```

## Registering a Resource Controller

Register your controller with the `Resource` method:

```go
r := router.New()
r.Resource("/users", UserController{})
```

This automatically creates the following routes:

| Method | Path       | Handler                | Description       |
|--------|------------|------------------------|-------------------|
| GET    | /users     | UserController.Index   | List all users    |
| GET    | /users/:id | UserController.Show    | Get a single user |
| POST   | /users     | UserController.Create  | Create a new user |
| PUT    | /users/:id | UserController.Update  | Update a user     |
| DELETE | /users/:id | UserController.Delete  | Delete a user     |

You don't need to implement all methods - any method not implemented from the `DefaultController` will return a 405 Method Not Allowed response.

## Controllers with Dependencies

Controllers can have dependencies injected through their constructors:

```go
type UserService interface {
    List() ([]User, error)
    Get(id string) (User, error)
    Create(user User) error
    Update(id string, user User) error
    Delete(id string) error
}

type UserController struct {
    router.DefaultController
    service UserService
}

// Constructor function
func NewUserController(service UserService) UserController {
    return UserController{service: service}
}

// Index method using the service
func (c UserController) Index(w http.ResponseWriter, r *http.Request, p router.Params) {
    users, err := c.service.List()
    if err != nil {
        router.Error(w, http.StatusInternalServerError, "Failed to fetch users")
        return
    }
    router.JSON(w, http.StatusOK, users)
}

// In main.go
userService := services.NewUserService(db)
userController := controllers.NewUserController(userService)
r.Resource("/users", userController)
```

## Custom Resource Methods

You can add custom methods to your resources beyond the standard CRUD operations:

```go
type UserController struct {
    router.DefaultController
}

// Standard method
func (c UserController) Show(w http.ResponseWriter, r *http.Request, p router.Params) {
    id := p["id"]
    router.JSON(w, http.StatusOK, User{ID: id, Name: "User " + id})
}

// Custom method
func (c UserController) ResetPassword(w http.ResponseWriter, r *http.Request, p router.Params) {
    id := p["id"]
    // Reset password logic...
    router.JSON(w, http.StatusOK, map[string]string{
        "message": "Password reset email sent",
    })
}

// In main.go
userController := UserController{}
r.Resource("/users", userController)

// Add custom route for the resource
r.Post("/users/:id/reset-password", userController.ResetPassword)
```

## Resource with Nested Resources

You can nest resources to represent hierarchical relationships:

```go
// Controllers
userController := UserController{}
postController := PostController{}

// Register parent resource
r.Resource("/users", userController)

// For each user, register their posts as a nested resource
r.Get("/users/:userId/posts", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    userId := p["userId"]
    // List posts for this user...
})

r.Get("/users/:userId/posts/:id", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    userId := p["userId"]
    postId := p["id"]
    // Get a specific post for this user...
})
```

## Resource with Automatic Parameter Binding

Combine resource controllers with parameter binding for cleaner code:

```go
type UserController struct {
    router.DefaultController
    service UserService
}

type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=3"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=18"`
}

func (c UserController) Create(w http.ResponseWriter, r *http.Request, p router.Params) {
    var req CreateUserRequest
    
    // Parse and validate JSON input
    if err := router.ParseJSON(r, &req); err != nil {
        router.Error(w, http.StatusBadRequest, err.Error())
        return
    }
    
    // Now use validated input
    user := User{
        ID:    generateID(),
        Name:  req.Name,
        Email: req.Email,
        Age:   req.Age,
    }
    
    if err := c.service.Create(user); err != nil {
        router.Error(w, http.StatusInternalServerError, "Failed to create user")
        return
    }
    
    router.JSON(w, http.StatusCreated, user)
}

// Alternative using the router.BindJSON helper
func CreateUser(w http.ResponseWriter, r *http.Request, p router.Params, req CreateUserRequest) {
    // req is already validated
    // ...
}

// Register with binding
r.Post("/users", router.BindJSON(CreateUser))
```

## Structuring Controllers in a Project

A common way to organize controllers in a larger project:

```
project/
  ├── controllers/
  │   ├── user_controller.go
  │   ├── post_controller.go
  │   └── comment_controller.go
  ├── models/
  │   ├── user.go
  │   ├── post.go
  │   └── comment.go
  ├── services/
  │   ├── user_service.go
  │   └── post_service.go
  ├── routes/
  │   └── routes.go
  └── main.go
```

In `routes.go`:

```go
package routes

import (
    "github.com/yourusername/myapp/controllers"
    "github.com/yourusername/myapp/services"
    "github.com/yourusername/mora-router/router"
)

func Setup(r *router.MoraRouter, db *sql.DB) {
    // Initialize services
    userService := services.NewUserService(db)
    postService := services.NewPostService(db)
    
    // Initialize controllers
    userController := controllers.NewUserController(userService)
    postController := controllers.NewPostController(postService)
    
    // Register resources
    r.Resource("/users", userController)
    r.Resource("/posts", postController)
    
    // Add custom routes
    r.Post("/users/:id/reset-password", userController.ResetPassword)
}
```

In `main.go`:

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/yourusername/myapp/routes"
    "github.com/yourusername/mora-router/router"
)

func main() {
    // Initialize router
    r := router.New(
        router.WithLogging(),
        router.WithRecovery(),
    )
    
    // Initialize database
    db, err := initDB()
    if err != nil {
        log.Fatal(err)
    }
    
    // Setup routes
    routes.Setup(r, db)
    
    // Start server
    log.Println("Server started on :8080")
    http.ListenAndServe(":8080", r)
}
```

## Controller Testing

Testing controllers is straightforward with MoraRouter's test utilities:

```go
func TestUserController_Show(t *testing.T) {
    // Create a test router
    r := router.New()
    
    // Create a mock service
    mockService := &MockUserService{
        GetFunc: func(id string) (User, error) {
            return User{ID: id, Name: "Test User"}, nil
        },
    }
    
    // Create controller with mock
    controller := NewUserController(mockService)
    
    // Register it
    r.Resource("/users", controller)
    
    // Create test client
    client := router.NewTestClient(r)
    
    // Make a test request
    resp := client.Get("/users/123")
    
    // Assert response
    if !resp.IsOK() {
        t.Fatalf("Expected OK response, got %d", resp.StatusCode)
    }
    
    var user User
    resp.DecodeJSON(&user)
    
    if user.ID != "123" || user.Name != "Test User" {
        t.Fatalf("Unexpected user: %+v", user)
    }
}
```

## Conclusion

Controllers in MoraRouter provide a structured way to organize your handlers, especially for RESTful resources. By using the `ResourceController` interface and the `DefaultController` base, you can create consistent APIs with minimal boilerplate code.

Next, check out [Data Binding](data-binding.md) to see how to handle request data more effectively.
