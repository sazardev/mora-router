package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestBasicConnection verifica que el router responda a peticiones básicas
func TestBasicConnection(t *testing.T) {
	r := New()

	// Definimos una ruta sencilla
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("pong"))
	})

	// Creamos un servidor de prueba
	server := httptest.NewServer(r)
	defer server.Close()

	// Hacemos una petición HTTP real
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(server.URL + "/ping")

	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Leemos la respuesta
	buf := make([]byte, 4)
	n, err := resp.Body.Read(buf)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("Error reading response: %v", err)
	}

	if string(buf[:n]) != "pong" {
		t.Errorf("Expected 'pong', got '%s'", string(buf[:n]))
	}
}

// TestMultipleConnections verifica que el router maneje múltiples conexiones correctamente
func TestMultipleConnections(t *testing.T) {
	r := New()

	// Contador de peticiones
	var requestCount int

	r.Get("/count", func(w http.ResponseWriter, r *http.Request, p Params) {
		requestCount++
		w.Write([]byte("ok"))
	})

	// Creamos un servidor de prueba
	server := httptest.NewServer(r)
	defer server.Close()

	// Hacemos múltiples peticiones en paralelo
	client := &http.Client{Timeout: 2 * time.Second}

	for i := 0; i < 10; i++ {
		go func() {
			resp, err := client.Get(server.URL + "/count")
			if err != nil {
				t.Errorf("Error making request: %v", err)
				return
			}
			defer resp.Body.Close()
		}()
	}

	// Esperamos un poco para que las peticiones se procesen
	time.Sleep(100 * time.Millisecond)

	// Verificamos que se hayan contabilizado todas (o casi todas)
	if requestCount < 8 {
		t.Errorf("Expected at least 8 requests processed, got %d", requestCount)
	}
}

// TestNotFoundConnection verifica la respuesta para rutas no existentes
func TestNotFoundConnection(t *testing.T) {
	r := New()

	// Definimos una ruta
	r.Get("/exists", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("ok"))
	})

	// Personalizamos el manejador 404
	r.NotFound(func(w http.ResponseWriter, r *http.Request, p Params) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("custom not found"))
	})

	// Creamos un servidor de prueba
	server := httptest.NewServer(r)
	defer server.Close()

	// Cliente HTTP
	client := &http.Client{Timeout: 2 * time.Second}

	// Probamos ruta inexistente
	resp, err := client.Get(server.URL + "/not-exists")
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	// Verificamos el mensaje personalizado
	buf := make([]byte, 16)
	n, _ := resp.Body.Read(buf)

	if string(buf[:n]) != "custom not found" {
		t.Errorf("Expected 'custom not found', got '%s'", string(buf[:n]))
	}
}

// TestConnectionTimeout verifica que el router maneje conexiones con timeout
func TestConnectionTimeout(t *testing.T) {
	r := New()

	// Ruta que tarda en responder
	r.Get("/slow", func(w http.ResponseWriter, r *http.Request, p Params) {
		time.Sleep(500 * time.Millisecond)
		w.Write([]byte("delayed response"))
	})

	// Creamos un servidor de prueba
	server := httptest.NewServer(r)
	defer server.Close()

	// Cliente HTTP con timeout corto
	clientWithShortTimeout := &http.Client{Timeout: 100 * time.Millisecond}

	// Esta petición debería fallar por timeout
	_, err := clientWithShortTimeout.Get(server.URL + "/slow")
	if err == nil {
		t.Errorf("Expected timeout error, but request succeeded")
	}

	// Cliente HTTP con timeout suficiente
	clientWithLongTimeout := &http.Client{Timeout: 1 * time.Second}

	// Esta petición debería funcionar
	resp, err := clientWithLongTimeout.Get(server.URL + "/slow")
	if err != nil {
		t.Fatalf("Error making request with sufficient timeout: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestConnectionWithHeaders verifica manejo de cabeceras en la respuesta
func TestConnectionWithHeaders(t *testing.T) {
	r := New()

	// Ruta que establece cabeceras
	r.Get("/headers", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Header().Set("X-Test", "test-value")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("with headers"))
	})

	// Creamos un servidor de prueba
	server := httptest.NewServer(r)
	defer server.Close()

	// Cliente HTTP
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get(server.URL + "/headers")
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	// Verificar cabeceras
	if resp.Header.Get("X-Test") != "test-value" {
		t.Errorf("Expected X-Test header to be 'test-value', got '%s'", resp.Header.Get("X-Test"))
	}

	if resp.Header.Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type header to be 'text/plain', got '%s'", resp.Header.Get("Content-Type"))
	}
}
