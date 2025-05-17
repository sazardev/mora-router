package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"mora-router/router"
)

// User represents a user entity in our application
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// UserStore is a simple in-memory store for users
type UserStore struct {
	mu     sync.RWMutex
	users  map[string]User
	nextID int
}

// NewUserStore creates a new user store with some initial data
func NewUserStore() *UserStore {
	store := &UserStore{
		users:  make(map[string]User),
		nextID: 1,
	}

	// Add some initial users
	store.AddUser(User{Name: "Alice", Email: "alice@example.com"})
	store.AddUser(User{Name: "Bob", Email: "bob@example.com"})
	store.AddUser(User{Name: "Charlie", Email: "charlie@example.com"})

	return store
}

// AddUser adds a new user and assigns an ID
func (s *UserStore) AddUser(user User) User {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strconv.Itoa(s.nextID)
	user.ID = id
	user.CreatedAt = time.Now()
	s.users[id] = user
	s.nextID++

	return user
}

// GetUsers returns all users
func (s *UserStore) GetUsers() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users
}

// GetUser returns a user by ID
func (s *UserStore) GetUser(id string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, found := s.users[id]
	return user, found
}

// UpdateUser updates an existing user
func (s *UserStore) UpdateUser(id string, updates User) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, found := s.users[id]
	if !found {
		return User{}, false
	}

	// Update fields but preserve ID and CreatedAt
	if updates.Name != "" {
		user.Name = updates.Name
	}
	if updates.Email != "" {
		user.Email = updates.Email
	}

	s.users[id] = user
	return user, true
}

// DeleteUser removes a user by ID
func (s *UserStore) DeleteUser(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, found := s.users[id]
	if !found {
		return false
	}

	delete(s.users, id)
	return true
}

// UserController implements ResourceController for User resources
type UserController struct {
	router.DefaultController
	store *UserStore
}

// Index lists all users
func (c *UserController) Index(w http.ResponseWriter, r *http.Request, p router.Params) {
	users := c.store.GetUsers()
	router.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "All users",
		"count":   len(users),
		"users":   users,
	})
}

// Show displays a single user
func (c *UserController) Show(w http.ResponseWriter, r *http.Request, p router.Params) {
	id := p["id"]
	user, found := c.store.GetUser(id)

	if !found {
		router.JSON(w, http.StatusNotFound, map[string]interface{}{
			"error": "User not found",
		})
		return
	}

	router.JSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("User details for ID: %s", id),
		"user":    user,
	})
}

// Create adds a new user
func (c *UserController) Create(w http.ResponseWriter, r *http.Request, p router.Params) {
	var newUser User
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		router.JSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid user data",
		})
		return
	}

	createdUser := c.store.AddUser(newUser)
	router.JSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User created successfully",
		"user":    createdUser,
	})
}

// Update modifies an existing user
func (c *UserController) Update(w http.ResponseWriter, r *http.Request, p router.Params) {
	id := p["id"]
	var updates User
	err := json.NewDecoder(r.Body).Decode(&updates)
	if err != nil {
		router.JSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid update data",
		})
		return
	}

	updatedUser, found := c.store.UpdateUser(id, updates)
	if !found {
		router.JSON(w, http.StatusNotFound, map[string]interface{}{
			"error": "User not found",
		})
		return
	}

	router.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "User updated successfully",
		"user":    updatedUser,
	})
}

// Delete removes a user
func (c *UserController) Delete(w http.ResponseWriter, r *http.Request, p router.Params) {
	id := p["id"]
	success := c.store.DeleteUser(id)

	if !success {
		router.JSON(w, http.StatusNotFound, map[string]interface{}{
			"error": "User not found",
		})
		return
	}

	router.JSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("User with ID %s deleted successfully", id),
	})
}

func main() {
	// Create a user store
	userStore := NewUserStore()

	// Create a user controller with the store
	userController := &UserController{
		store: userStore,
	}

	// Create the router with some useful middleware
	r := router.New(
		router.WithLogging(),
		router.WithRecovery(),
		router.WithCORS("*"),
		router.WithDebug(),
	)

	// Register the user resource
	r.Resource("/api/users", userController)

	// Add a simple home page that explains how to use the API
	r.Get("/", func(w http.ResponseWriter, req *http.Request, p router.Params) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Mora Router Resource Demo</title>
			<style>
				body { 
					font-family: Arial, sans-serif; 
					max-width: 800px; 
					margin: 0 auto; 
					padding: 20px; 
					line-height: 1.6;
				}
				h1 { color: #333; }
				h2 { color: #555; margin-top: 30px; }
				code { 
					background: #f4f4f4; 
					padding: 2px 5px; 
					border-radius: 3px;
				}
				pre {
					background: #f4f4f4;
					padding: 10px;
					border-radius: 5px;
					overflow: auto;
				}
				table {
					border-collapse: collapse;
					width: 100%;
				}
				th, td {
					border: 1px solid #ddd;
					padding: 8px;
					text-align: left;
				}
				th {
					background-color: #f2f2f2;
				}
				.method {
					font-weight: bold;
				}
				.get { color: #2c88d9; }
				.post { color: #27ae60; }
				.put { color: #f39c12; }
				.delete { color: #e74c3c; }
			</style>
		</head>
		<body>
			<h1>Mora Router Resource Demo</h1>
			<p>This is a demonstration of the Resource functionality in Mora Router.</p>
			
			<h2>Available Endpoints</h2>
			<table>
				<tr>
					<th>Method</th>
					<th>URL</th>
					<th>Action</th>
					<th>Description</th>
				</tr>
				<tr>
					<td><span class="method get">GET</span></td>
					<td>/api/users</td>
					<td>Index</td>
					<td>List all users</td>
				</tr>
				<tr>
					<td><span class="method get">GET</span></td>
					<td>/api/users/:id</td>
					<td>Show</td>
					<td>Get a single user by ID</td>
				</tr>
				<tr>
					<td><span class="method post">POST</span></td>
					<td>/api/users</td>
					<td>Create</td>
					<td>Create a new user</td>
				</tr>
				<tr>
					<td><span class="method put">PUT</span></td>
					<td>/api/users/:id</td>
					<td>Update</td>
					<td>Update an existing user</td>
				</tr>
				<tr>
					<td><span class="method delete">DELETE</span></td>
					<td>/api/users/:id</td>
					<td>Delete</td>
					<td>Delete a user</td>
				</tr>
			</table>

			<h2>Example Usage</h2>
			
			<h3>List all users</h3>
			<pre>GET /api/users</pre>
			
			<h3>Get a specific user</h3>
			<pre>GET /api/users/1</pre>
			
			<h3>Create a new user</h3>
			<pre>
POST /api/users
Content-Type: application/json

{
  "name": "John Doe",
  "email": "john@example.com"
}</pre>
			
			<h3>Update a user</h3>
			<pre>
PUT /api/users/1
Content-Type: application/json

{
  "name": "Jane Doe",
  "email": "jane@example.com"
}</pre>
			
			<h3>Delete a user</h3>
			<pre>DELETE /api/users/1</pre>
			
			<h2>Try it out</h2>
			<p>You can explore this API in the browser or using tools like curl, Postman, etc.</p>
			<p>For a debugging interface, check out <a href="/_mora/debug">the debug panel</a>.</p>
		</body>
		</html>
		`
		fmt.Fprint(w, html)
	})

	// Start the server
	port := ":8080"
	log.Printf("Server starting on http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, r))
}
