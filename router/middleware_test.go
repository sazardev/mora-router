package router

import (
	"net/http"
	"testing"
	"time"
)

// TestRecoveryMiddleware verifica que el middleware de recovery capture panics
func TestRecoveryMiddleware(t *testing.T) {
	r := New(WithRecovery())

	// Ruta que causa un panic
	r.Get("/panic", func(w http.ResponseWriter, r *http.Request, p Params) {
		panic("test panic")
	})

	client := NewTestClient(r)

	// El middleware de recovery debería evitar que el test se rompa
	resp := client.Get("/panic")

	// Debería devolver un error 500
	if !resp.IsServerError() {
		t.Errorf("Expected server error after panic, got status %d", resp.StatusCode)
	}
}

// TestLoggingMiddleware verifica que el middleware de logging funcione correctamente
func TestLoggingMiddleware(t *testing.T) {
	// El logging es difícil de probar directamente, así que solo verificamos
	// que no interfiera con la respuesta normal
	r := New(WithLogging())

	r.Get("/log-test", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("Logged request"))
	})

	client := NewTestClient(r)

	resp := client.Get("/log-test")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Text() != "Logged request" {
		t.Errorf("Expected 'Logged request', got '%s'", resp.Text())
	}
}

// TestCORSMiddleware verifica que el middleware CORS agregue los encabezados correctos
func TestCORSMiddleware(t *testing.T) {
	r := New(WithCORS("*"))

	r.Get("/cors-test", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("CORS enabled"))
	})

	client := NewTestClient(r)

	// Simular una solicitud CORS con Origin
	resp := client.
		WithHeader("Origin", "http://example.com").
		Get("/cors-test")

	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	// Verificar que se haya agregado el encabezado Access-Control-Allow-Origin
	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin header to be '*', got '%s'", allowOrigin)
	}

	// Nota: La prueba del preflight OPTIONS se omite hasta que implementemos
	// completamente el soporte para preflight en el middleware CORS
}

// timeoutHandler es un middleware que simula un tiempo de espera
func timeoutHandler(timeout time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			// Simular procesamiento con temporizador
			done := make(chan bool)

			go func() {
				next(w, r, p)
				done <- true
			}()

			select {
			case <-done:
				// La solicitud se completó dentro del límite de tiempo
				return
			case <-time.After(timeout):
				// Se agotó el tiempo
				w.WriteHeader(http.StatusRequestTimeout)
				w.Write([]byte("Request timed out"))
				return
			}
		}
	}
}

// TestTimeoutMiddleware verifica que un middleware de tiempo de espera funcione correctamente
func TestTimeoutMiddleware(t *testing.T) {
	r := New()

	// Aplicar middleware de timeout para todas las rutas
	r.Use(timeoutHandler(50 * time.Millisecond))

	// Ruta rápida (debería completarse)
	r.Get("/fast", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("Fast response"))
	})

	// Ruta lenta (debería agotar el tiempo de espera)
	r.Get("/slow", func(w http.ResponseWriter, r *http.Request, p Params) {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte("Slow response"))
	})

	client := NewTestClient(r)

	// La ruta rápida debería completarse con éxito
	resp := client.Get("/fast")
	if !resp.IsOK() {
		t.Errorf("Expected status 200 for fast route, got %d", resp.StatusCode)
	}

	// La ruta lenta debería agotar el tiempo de espera
	resp = client.Get("/slow")
	if resp.StatusCode != http.StatusRequestTimeout {
		t.Errorf("Expected status 408 for slow route, got %d", resp.StatusCode)
	}
}

// authMiddleware es un middleware que comprueba un token de autorización básico
func authMiddleware(token string) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			// Verificar el encabezado Authorization
			auth := r.Header.Get("Authorization")
			expected := "Bearer " + token

			if auth != expected {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Unauthorized"))
				return
			}

			// Token válido, continuar
			next(w, r, p)
		}
	}
}

// TestAuthMiddleware verifica que un middleware de autenticación funcione correctamente
func TestAuthMiddleware(t *testing.T) {
	r := New()

	// Establecer un token predefinido para pruebas
	testToken := "test-token-123"

	// Aplicar middleware de autenticación
	r.Use(authMiddleware(testToken))

	// Ruta protegida
	r.Get("/protected", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("Protected content"))
	})

	client := NewTestClient(r)

	// Solicitud sin token debería fallar
	resp := client.Get("/protected")
	if !resp.IsUnauthorized() {
		t.Errorf("Expected status 401 without token, got %d", resp.StatusCode)
	}

	// Solicitud con token incorrecto debería fallar
	resp = client.WithAuth("wrong-token").Get("/protected")
	if !resp.IsUnauthorized() {
		t.Errorf("Expected status 401 with wrong token, got %d", resp.StatusCode)
	}

	// Solicitud con token correcto debería tener éxito
	resp = client.WithAuth(testToken).Get("/protected")
	if !resp.IsOK() {
		t.Errorf("Expected status 200 with correct token, got %d", resp.StatusCode)
	}

	if resp.Text() != "Protected content" {
		t.Errorf("Expected 'Protected content', got '%s'", resp.Text())
	}
}
