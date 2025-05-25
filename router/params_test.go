package router

import (
	"net/http"
	"testing"
)

// TestBasicParams verifica el manejo básico de parámetros
func TestBasicParams(t *testing.T) {
	r := New()

	// Ruta con parámetro simple
	r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte(p["id"]))
	})

	client := NewTestClient(r)

	// Probar con diferentes valores de parámetro
	testCases := []struct {
		path     string
		expected string
	}{
		{"/users/123", "123"},
		{"/users/abc", "abc"},
		{"/users/user_name", "user_name"},
	}

	for _, tc := range testCases {
		resp := client.Get(tc.path)
		if !resp.IsOK() {
			t.Errorf("Path %s: Expected status 200, got %d", tc.path, resp.StatusCode)
			continue
		}

		if resp.Text() != tc.expected {
			t.Errorf("Path %s: Expected '%s', got '%s'", tc.path, tc.expected, resp.Text())
		}
	}
}

// TestMultipleParams verifica múltiples parámetros en una ruta
func TestMultipleParams(t *testing.T) {
	r := New()

	// Ruta con varios parámetros
	r.Get("/users/:user_id/posts/:post_id", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("user:" + p["user_id"] + ",post:" + p["post_id"]))
	})

	client := NewTestClient(r)

	resp := client.Get("/users/123/posts/456")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	expected := "user:123,post:456"
	if resp.Text() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, resp.Text())
	}
}

// TestParamWithRegex verifica el uso de expresiones regulares en parámetros
func TestParamWithRegex(t *testing.T) {
	r := New()

	// Rutas con validaciones regex
	r.Get("/items/:id(\\d+)", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("numeric:" + p["id"]))
	})

	r.Get("/products/:code([A-Z]{3}\\d{4})", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("product:" + p["code"]))
	})

	client := NewTestClient(r)

	// Probar parámetros numéricos
	validNumericCases := []string{"/items/123", "/items/0", "/items/987654321"}
	for _, path := range validNumericCases {
		resp := client.Get(path)
		if !resp.IsOK() {
			t.Errorf("Path %s: Expected status 200, got %d", path, resp.StatusCode)
		}
	}

	// Probar parámetros no numéricos (no deberían coincidir)
	invalidNumericCases := []string{"/items/abc", "/items/123abc", "/items/"}
	for _, path := range invalidNumericCases {
		resp := client.Get(path)
		if !resp.IsNotFound() {
			t.Errorf("Path %s: Expected status 404, got %d", path, resp.StatusCode)
		}
	}

	// Probar códigos de producto
	validProductCases := []string{"/products/ABC1234", "/products/XYZ9876"}
	for _, path := range validProductCases {
		resp := client.Get(path)
		if !resp.IsOK() {
			t.Errorf("Path %s: Expected status 200, got %d", path, resp.StatusCode)
		}
	}

	// Probar códigos de producto inválidos
	invalidProductCases := []string{"/products/abc1234", "/products/ABC123", "/products/ABCD1234"}
	for _, path := range invalidProductCases {
		resp := client.Get(path)
		if !resp.IsNotFound() {
			t.Errorf("Path %s: Expected status 404, got %d", path, resp.StatusCode)
		}
	}
}

// TestWildcardParam verifica parámetros comodín
func TestWildcardParam(t *testing.T) {
	r := New()

	// Ruta con comodín
	r.Get("/files/*path", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("path:" + p["path"]))
	})

	client := NewTestClient(r)

	testCases := []struct {
		path     string
		expected string
	}{
		{"/files/document.txt", "path:document.txt"},
		{"/files/images/photo.jpg", "path:images/photo.jpg"},
		{"/files/documents/reports/2023/annual.pdf", "path:documents/reports/2023/annual.pdf"},
	}

	for _, tc := range testCases {
		resp := client.Get(tc.path)
		if !resp.IsOK() {
			t.Errorf("Path %s: Expected status 200, got %d", tc.path, resp.StatusCode)
			continue
		}

		if resp.Text() != tc.expected {
			t.Errorf("Path %s: Expected '%s', got '%s'", tc.path, tc.expected, resp.Text())
		}
	}
}

// TestParamFromContext verifica la extracción de parámetros desde el contexto
func TestParamFromContext(t *testing.T) {
	r := New()

	r.Get("/context/:id", func(w http.ResponseWriter, req *http.Request, p Params) {
		// Extraer desde el contexto usando la función helper
		id := Param(req, "id")
		w.Write([]byte("context:" + id))
	})

	client := NewTestClient(r)

	resp := client.Get("/context/42")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	expected := "context:42"
	if resp.Text() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, resp.Text())
	}
}

// TestAlternativeParamSyntax verifica la sintaxis alternativa para parámetros
func TestAlternativeParamSyntax(t *testing.T) {
	r := New()

	// Sintaxis con llaves
	r.Get("/users/{id:[0-9]+}/profile", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("profile:" + p["id"]))
	})

	client := NewTestClient(r)

	// Casos válidos
	resp := client.Get("/users/123/profile")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	expected := "profile:123"
	if resp.Text() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, resp.Text())
	}

	// Caso inválido
	resp = client.Get("/users/abc/profile")
	if !resp.IsNotFound() {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}
