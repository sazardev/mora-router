package router

import (
	"encoding/json"
	"encoding/xml"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestJSONRendering verifica el renderizado JSON
func TestJSONRendering(t *testing.T) {
	r := New()

	// Datos para renderizar
	type User struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	testUser := User{
		ID:    "123",
		Name:  "Test User",
		Email: "test@example.com",
	}

	r.Get("/json", func(w http.ResponseWriter, r *http.Request, p Params) {
		JSON(w, http.StatusOK, testUser)
	})

	// Probar con TestClient
	client := NewTestClient(r)
	resp := client.Get("/json")

	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verificar Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
	}

	// Deserializar y verificar datos
	var user User
	if err := resp.JSON(&user); err != nil {
		t.Errorf("Error parsing JSON: %v", err)
	}

	if user.ID != testUser.ID || user.Name != testUser.Name || user.Email != testUser.Email {
		t.Errorf("JSON data mismatch: got %+v, want %+v", user, testUser)
	}
}

// TestSimpleTemplateRendering verifica el renderizado básico de plantillas HTML
func TestSimpleTemplateRendering(t *testing.T) {
	// Crear plantilla de prueba en memoria
	tmpl := template.Must(template.New("test").Parse("<html><body>Hello, {{.Name}}!</body></html>"))

	r := New()
	render := NewRender()
	render.HTMLTemplates = tmpl

	r.Get("/template", func(w http.ResponseWriter, r *http.Request, p Params) {
		render.HTML(w, http.StatusOK, "test", map[string]interface{}{
			"Name": "World",
		})
	})

	// Probar con servidor de prueba
	server := httptest.NewServer(r)
	defer server.Close()

	resp, err := http.Get(server.URL + "/template")
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verificar Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected Content-Type to contain 'text/html', got '%s'", contentType)
	}

	// Leer cuerpo
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response: %v", err)
	}

	expected := "<html><body>Hello, World!</body></html>"
	if string(body) != expected {
		t.Errorf("Body mismatch: got '%s', want '%s'", string(body), expected)
	}
}

// TestXMLRendering verifica el renderizado XML
func TestXMLRendering(t *testing.T) {
	r := New()
	render := NewRender()

	// Datos para renderizar
	type Product struct {
		ID    string  `xml:"id,attr"`
		Name  string  `xml:"name"`
		Price float64 `xml:"price"`
	}

	testProduct := Product{
		ID:    "prod-123",
		Name:  "Test Product",
		Price: 99.99,
	}

	r.Get("/xml", func(w http.ResponseWriter, r *http.Request, p Params) {
		render.XML(w, http.StatusOK, testProduct)
	})

	// Probar con TestClient
	client := NewTestClient(r)
	resp := client.Get("/xml")

	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verificar Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/xml") {
		t.Errorf("Expected Content-Type to contain 'application/xml', got '%s'", contentType)
	}

	// Deserializar y verificar datos
	var product Product
	if err := xml.Unmarshal(resp.Body, &product); err != nil {
		t.Errorf("Error parsing XML: %v", err)
	}

	if product.ID != testProduct.ID || product.Name != testProduct.Name || product.Price != testProduct.Price {
		t.Errorf("XML data mismatch: got %+v, want %+v", product, testProduct)
	}
}

// TestErrorRendering verifica el renderizado de errores
func TestErrorRendering(t *testing.T) {
	r := New()

	r.Get("/error", func(w http.ResponseWriter, r *http.Request, p Params) {
		Error(w, http.StatusBadRequest, "Invalid request")
	})

	// Probar con TestClient
	client := NewTestClient(r)
	resp := client.Get("/error")

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	expected := "Invalid request\n"
	if resp.Text() != expected {
		t.Errorf("Error message mismatch: got '%s', want '%s'", resp.Text(), expected)
	}
}

// TestJSONBindingAndResponse prueba la combinación de vinculación JSON y respuesta
func TestJSONBindingAndResponse(t *testing.T) {
	r := New()

	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	type LoginResponse struct {
		Token   string `json:"token"`
		UserID  string `json:"user_id"`
		Success bool   `json:"success"`
	}

	r.Post("/login", BindJSON(func(w http.ResponseWriter, r *http.Request, p Params, req LoginRequest) {
		// Simular autenticación
		if req.Username == "admin" && req.Password == "secret" {
			JSON(w, http.StatusOK, LoginResponse{
				Token:   "simulated-jwt-token",
				UserID:  "user-123",
				Success: true,
			})
		} else {
			JSON(w, http.StatusUnauthorized, map[string]interface{}{
				"success": false,
				"message": "Invalid credentials",
			})
		}
	}))

	// Probar con TestClient - credenciales correctas
	client := NewTestClient(r)

	loginData := LoginRequest{
		Username: "admin",
		Password: "secret",
	}

	loginDataBytes, _ := json.Marshal(loginData)
	resp := client.
		WithContentType("application/json").
		Post("/login", loginDataBytes)

	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var response LoginResponse
	if err := resp.JSON(&response); err != nil {
		t.Errorf("Error parsing JSON response: %v", err)
	}

	if !response.Success || response.Token == "" {
		t.Errorf("Expected successful login response, got %+v", response)
	}

	// Probar credenciales incorrectas
	loginData.Password = "wrong"
	loginDataBytes, _ = json.Marshal(loginData)

	resp = client.
		WithContentType("application/json").
		Post("/login", loginDataBytes)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}

	var errorResponse map[string]interface{}
	if err := resp.JSON(&errorResponse); err != nil {
		t.Errorf("Error parsing JSON error response: %v", err)
	}

	success, ok := errorResponse["success"].(bool)
	if !ok || success {
		t.Errorf("Expected success:false in error response, got %v", errorResponse)
	}
}
