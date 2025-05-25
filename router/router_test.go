package router

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestBasicRouting verifica que las rutas básicas funcionen correctamente
func TestBasicRouting(t *testing.T) {
	r := New()

	// Definimos una ruta simple que devuelve un mensaje
	r.Get("/hello", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("Hello, World!"))
	})

	// Creamos un cliente de prueba
	client := NewTestClient(r)
	
	// Probamos la ruta existente
	resp := client.Get("/hello")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Text() != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got '%s'", resp.Text())
	}

	// Probamos una ruta inexistente
	resp = client.Get("/not-exists")
	if !resp.IsNotFound() {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

// TestRouteParams verifica el manejo de parámetros en rutas
func TestRouteParams(t *testing.T) {
	r := New()

	// Definimos una ruta con un parámetro simple
	r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("User ID: " + p["id"]))
	})

	// Ruta con parámetro y validación regex
	r.Get("/products/:code([A-Z]{3}\\d{4})", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("Product Code: " + p["code"]))
	})

	client := NewTestClient(r)
	
	// Probamos el parámetro simple
	resp := client.Get("/users/123")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Text() != "User ID: 123" {
		t.Errorf("Expected 'User ID: 123', got '%s'", resp.Text())
	}

	// Probamos el parámetro con regex válido
	resp = client.Get("/products/ABC1234")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Text() != "Product Code: ABC1234" {
		t.Errorf("Expected 'Product Code: ABC1234', got '%s'", resp.Text())
	}

	// Probamos el parámetro con regex inválido
	resp = client.Get("/products/123456")
	if !resp.IsNotFound() {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

// TestStatusCodes verifica los diferentes códigos de estado
func TestStatusCodes(t *testing.T) {
	r := New()

	// Ruta que devuelve Created (201)
	r.Post("/resources", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Resource created"))
	})

	// Ruta que devuelve No Content (204)
	r.Delete("/resources/:id", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Ruta que devuelve Bad Request (400)
	r.Get("/error/bad-request", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
	})

	// Ruta que devuelve Internal Server Error (500)
	r.Get("/error/server", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
	})

	client := NewTestClient(r)
	
	// Probamos Created
	resp := client.Post("/resources", nil)
	if !resp.IsCreated() {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Probamos No Content
	resp = client.Delete("/resources/123")
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	// Probamos Bad Request
	resp = client.Get("/error/bad-request")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	// Probamos Server Error
	resp = client.Get("/error/server")
	if !resp.IsServerError() {
		t.Errorf("Expected server error status code (5xx), got %d", resp.StatusCode)
	}
}

// TestBasicMiddleware verifica que los middlewares funcionen correctamente
func TestBasicMiddleware(t *testing.T) {
	r := New()

	// Define un middleware simple que agrega un encabezado
	headerMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			w.Header().Set("X-Test-Header", "middleware-value")
			next(w, r, p)
		}
	}

	// Configura el middleware globalmente
	r.Use(headerMiddleware)

	// Define una ruta
	r.Get("/with-middleware", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("Hello with middleware"))
	})

	client := NewTestClient(r)
	
	// Verifica que el middleware se aplique
	resp := client.Get("/with-middleware")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	headerValue := resp.Header.Get("X-Test-Header")
	if headerValue != "middleware-value" {
		t.Errorf("Expected header value 'middleware-value', got '%s'", headerValue)
	}
}

// TestJSONResponse verifica respuestas JSON
func TestJSONResponse(t *testing.T) {
	r := New()

	type User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	r.Get("/api/users/:id", func(w http.ResponseWriter, r *http.Request, p Params) {
		user := User{
			ID:   p["id"],
			Name: "Test User",
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})

	client := NewTestClient(r)
	
	resp := client.Get("/api/users/123")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var user User
	if err := resp.JSON(&user); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}
	
	if user.ID != "123" {
		t.Errorf("Expected user ID '123', got '%s'", user.ID)
	}
	
	if user.Name != "Test User" {
		t.Errorf("Expected user name 'Test User', got '%s'", user.Name)
	}
}
