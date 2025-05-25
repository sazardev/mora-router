package router

import (
	"net/http"
	"strings"
	"testing"
)

// TestHTTPMethods verifica que todos los métodos HTTP funcionen correctamente
func TestHTTPMethods(t *testing.T) {
	r := New()

	// Definir rutas para cada método HTTP
	r.Get("/methods", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("GET"))
	})
	r.Post("/methods", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("POST"))
	})
	r.Put("/methods", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("PUT"))
	})
	r.Delete("/methods", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("DELETE"))
	})
	r.Patch("/methods", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("PATCH"))
	})
	r.Options("/methods", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("OPTIONS"))
	})

	client := NewTestClient(r)

	// Test GET
	resp := client.Get("/methods")
	if !resp.IsOK() || resp.Text() != "GET" {
		t.Errorf("GET method failed, got %d: %s", resp.StatusCode, resp.Text())
	}

	// Test POST
	resp = client.Post("/methods", nil)
	if !resp.IsOK() || resp.Text() != "POST" {
		t.Errorf("POST method failed, got %d: %s", resp.StatusCode, resp.Text())
	}

	// Nota: Para completar los tests necesitamos implementar los métodos
	// PUT, DELETE, PATCH y OPTIONS en TestClient
}

// TestWildcardRoutes verifica el manejo de rutas con comodines
func TestWildcardRoutes(t *testing.T) {
	r := New()

	// Ruta con comodín
	r.Get("/files/*filepath", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("File path: " + p["filepath"]))
	})

	client := NewTestClient(r)

	// Probar una ruta simple
	resp := client.Get("/files/document.txt")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Text() != "File path: document.txt" {
		t.Errorf("Expected 'File path: document.txt', got '%s'", resp.Text())
	}

	// Probar una ruta con subcarpetas
	resp = client.Get("/files/docs/report/annual.pdf")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Text() != "File path: docs/report/annual.pdf" {
		t.Errorf("Expected 'File path: docs/report/annual.pdf', got '%s'", resp.Text())
	}
}

// TestRouteGroups verifica el manejo de grupos de rutas
func TestRouteGroups(t *testing.T) {
	r := New()

	// Crear un grupo de rutas con prefijo /api
	api := r.Group("/api")

	// Añadir rutas al grupo
	api.Get("/users", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("API Users"))
	})

	api.Get("/products", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("API Products"))
	})
	// Duplicar lo que ya existe, no crear un subgrupo anidado
	api.Get("/v1/users", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("API v1 Users"))
	})

	client := NewTestClient(r)

	// Verificar rutas del primer grupo
	resp := client.Get("/api/users")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Text() != "API Users" {
		t.Errorf("Expected 'API Users', got '%s'", resp.Text())
	}

	resp = client.Get("/api/products")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Text() != "API Products" {
		t.Errorf("Expected 'API Products', got '%s'", resp.Text())
	}

	// Verificar ruta del subgrupo
	resp = client.Get("/api/v1/users")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Text() != "API v1 Users" {
		t.Errorf("Expected 'API v1 Users', got '%s'", resp.Text())
	}
}

// TestMiddlewareOrder verifica que el orden de los middlewares sea respetado
func TestMiddlewareOrder(t *testing.T) {
	r := New()

	// Array para registrar el orden de ejecución
	var orderTracker []string

	// Middleware 1: agregar "1" al tracker
	middleware1 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			orderTracker = append(orderTracker, "1")
			next(w, r, p)
		}
	}

	// Middleware 2: agregar "2" al tracker
	middleware2 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			orderTracker = append(orderTracker, "2")
			next(w, r, p)
		}
	}

	// Middleware 3: agregar "3" al tracker
	middleware3 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			orderTracker = append(orderTracker, "3")
			next(w, r, p)
		}
	}

	// Aplicar middlewares
	r.Use(middleware1)
	r.Use(middleware2)
	r.Use(middleware3)

	// Definir una ruta
	r.Get("/middleware-order", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("OK"))
	})

	client := NewTestClient(r)

	// Reiniciar el tracker
	orderTracker = []string{}

	// Hacer una solicitud
	resp := client.Get("/middleware-order")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verificar el orden de ejecución
	expected := "123"
	actual := strings.Join(orderTracker, "")
	if actual != expected {
		t.Errorf("Expected middleware execution order '%s', got '%s'", expected, actual)
	}
}

// TestContentNegotiation verifica la negociación de contenido básica
func TestContentNegotiation(t *testing.T) {
	r := New()

	// Ruta que responde diferente según el Accept header
	r.Get("/negotiate", func(w http.ResponseWriter, r *http.Request, p Params) {
		accept := r.Header.Get("Accept")

		switch {
		case strings.Contains(accept, "application/json"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message":"JSON response"}`))
		case strings.Contains(accept, "application/xml"):
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<response><message>XML response</message></response>`))
		case strings.Contains(accept, "text/plain"):
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("Plain text response"))
		default:
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html><body>HTML response</body></html>"))
		}
	})

	client := NewTestClient(r)

	// Probar con Accept: application/json
	resp := client.
		WithHeader("Accept", "application/json").
		Get("/negotiate")

	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Text(), "JSON response") {
		t.Errorf("Expected JSON response, got: %s", resp.Text())
	}

	// Probar con Accept: application/xml
	resp = client.
		WithHeader("Accept", "application/xml").
		Get("/negotiate")

	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Text(), "XML response") {
		t.Errorf("Expected XML response, got: %s", resp.Text())
	}

	// Probar con Accept: text/plain
	resp = client.
		WithHeader("Accept", "text/plain").
		Get("/negotiate")

	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Text() != "Plain text response" {
		t.Errorf("Expected plain text response, got: %s", resp.Text())
	}

	// Probar con Accept: text/html
	resp = client.
		WithHeader("Accept", "text/html").
		Get("/negotiate")

	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Text(), "HTML response") {
		t.Errorf("Expected HTML response, got: %s", resp.Text())
	}
}
