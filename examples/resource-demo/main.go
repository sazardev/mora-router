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
	store.AddUser(User{Name: "Omar", Email: "omar@example.com"})
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
		router.WithStaticFiles("/static", "templates"),
	)

	// Register the user resource
	r.Resource("/api/users", userController)

	// Set up templates with our new helper
	router.WithTemplates("templates")(r)

	// Add a simple home page that explains how to use the API
	r.Get("/", func(w http.ResponseWriter, req *http.Request, p router.Params) {
		// Get current users for display
		users := userStore.GetUsers()

		// Setup template data (CSS is automatically loaded from templates/style.css)
		data := map[string]interface{}{
			"BasePath": "/api/users",
			"Users":    users,
		}

		// Use our new helper function to render templates
		router.RenderTemplate(w, req, "index.html", data)
	})

	// Start the server
	port := ":8080"
	log.Printf("Server starting on http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, r))
}
