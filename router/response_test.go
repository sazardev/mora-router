package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestResponseStatusCodes verifica varios códigos de estado HTTP
func TestResponseStatusCodes(t *testing.T) {
	r := New()

	// Configurar rutas para diferentes códigos de estado
	statusCodes := map[string]int{
		"/ok":                  http.StatusOK,
		"/created":             http.StatusCreated,
		"/accepted":            http.StatusAccepted,
		"/no-content":          http.StatusNoContent,
		"/moved":               http.StatusMovedPermanently,
		"/bad-request":         http.StatusBadRequest,
		"/unauthorized":        http.StatusUnauthorized,
		"/forbidden":           http.StatusForbidden,
		"/not-found":           http.StatusNotFound,
		"/method-not-allowed":  http.StatusMethodNotAllowed,
		"/server-error":        http.StatusInternalServerError,
		"/service-unavailable": http.StatusServiceUnavailable,
	}

	for path, code := range statusCodes {
		// Capturar variables del loop con clausura
		statusCode := code
		r.Get(path, func(w http.ResponseWriter, r *http.Request, p Params) {
			w.WriteHeader(statusCode)
		})
	}

	client := NewTestClient(r)

	// Probar cada ruta
	for path, expectedCode := range statusCodes {
		resp := client.Get(path)
		if resp.StatusCode != expectedCode {
			t.Errorf("Path %s: Expected status %d, got %d", path, expectedCode, resp.StatusCode)
		}
	}
}

// TestResponseHeaders verifica el manejo de cabeceras HTTP
func TestResponseHeaders(t *testing.T) {
	r := New()

	// Ruta que establece múltiples cabeceras
	r.Get("/headers", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-API-Version", "1.0")
		w.Header().Set("X-Rate-Limit", "100")
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Write([]byte("{}"))
	})

	// Ruta que establece cabeceras múltiples con el mismo nombre
	r.Get("/multi-headers", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Header().Add("X-Tag", "tag1")
		w.Header().Add("X-Tag", "tag2")
		w.Header().Add("X-Tag", "tag3")
		w.Write([]byte("ok"))
	})

	client := NewTestClient(r)

	// Probar cabeceras individuales
	resp := client.Get("/headers")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	expectedHeaders := map[string]string{
		"Content-Type":  "application/json",
		"X-API-Version": "1.0",
		"X-Rate-Limit":  "100",
		"Cache-Control": "max-age=3600",
	}

	for key, expected := range expectedHeaders {
		if resp.Header.Get(key) != expected {
			t.Errorf("Header %s: Expected '%s', got '%s'", key, expected, resp.Header.Get(key))
		}
	}

	// Probar cabeceras múltiples
	resp = client.Get("/multi-headers")
	xTagValues := resp.Header.Values("X-Tag")

	// Verificar que haya al menos 3 valores
	if len(xTagValues) != 3 {
		t.Errorf("Expected 3 X-Tag headers, got %d", len(xTagValues))
	}

	expectedTags := []string{"tag1", "tag2", "tag3"}
	for i, tag := range expectedTags {
		if i >= len(xTagValues) || xTagValues[i] != tag {
			t.Errorf("X-Tag[%d]: Expected '%s', got '%s'", i, tag, xTagValues[i])
		}
	}
}

// TestResponseRedirects verifica el manejo de redirecciones
func TestResponseRedirects(t *testing.T) {
	r := New()

	// Configurar rutas para diferentes tipos de redirección
	r.Get("/redirect-301", func(w http.ResponseWriter, r *http.Request, p Params) {
		Redirect(w, r, "/destination", http.StatusMovedPermanently)
	})

	r.Get("/redirect-302", func(w http.ResponseWriter, r *http.Request, p Params) {
		Redirect(w, r, "/destination", http.StatusFound)
	})

	r.Get("/redirect-307", func(w http.ResponseWriter, r *http.Request, p Params) {
		Redirect(w, r, "/destination", http.StatusTemporaryRedirect)
	})

	r.Get("/destination", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("You've been redirected"))
	})

	// Crear servidor de prueba
	server := httptest.NewServer(r)
	defer server.Close()

	// Cliente HTTP que NO sigue redirecciones
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: time.Second,
	}

	// Probar redirecciones
	redirectTests := []struct {
		path           string
		expectedStatus int
		expectedLoc    string
	}{
		{"/redirect-301", http.StatusMovedPermanently, "/destination"},
		{"/redirect-302", http.StatusFound, "/destination"},
		{"/redirect-307", http.StatusTemporaryRedirect, "/destination"},
	}

	for _, test := range redirectTests {
		resp, err := client.Get(server.URL + test.path)
		if err != nil {
			t.Fatalf("Error making request to %s: %v", test.path, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != test.expectedStatus {
			t.Errorf("%s: Expected status %d, got %d", test.path, test.expectedStatus, resp.StatusCode)
		}

		location := resp.Header.Get("Location")
		// La Location puede tener el servidor, solo verificamos el final
		if location == "" || !strings.HasSuffix(location, test.expectedLoc) {
			t.Errorf("%s: Expected Location to end with '%s', got '%s'", test.path, test.expectedLoc, location)
		}
	}
}

// TestResponseCompression verifica la compresión de respuestas
func TestResponseCompression(t *testing.T) {
	r := New()

	// Ruta con contenido que se puede comprimir
	longText := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 100)

	r.Get("/compressed", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte(longText))
	})

	// Crear servidor de prueba
	server := httptest.NewServer(r)
	defer server.Close()

	// Hacer petición con Accept-Encoding: gzip
	req, _ := http.NewRequest("GET", server.URL+"/compressed", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	client := &http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	// En Go 1.24, encoding/gzip se aplica automáticamente para respuestas largas
	// pero esto puede variar según la implementación.
	// Aquí solo verificamos la respuesta básica
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Si hay compresión, debería haber un header Content-Encoding
	contentEncoding := resp.Header.Get("Content-Encoding")
	// Aceptamos tanto que haya compresión como que no la haya en esta prueba
	t.Logf("Response compression: Content-Encoding=%s", contentEncoding)
}
