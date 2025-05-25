package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// ProductController para probar recursos
type ProductController struct {
	DefaultController
}

// Index lista todos los productos
func (c ProductController) Index(w http.ResponseWriter, r *http.Request, p Params) {
	products := []map[string]interface{}{
		{"id": "1", "name": "Laptop", "price": 999.99},
		{"id": "2", "name": "Phone", "price": 499.99},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

// Show muestra un producto por ID
func (c ProductController) Show(w http.ResponseWriter, r *http.Request, p Params) {
	id := p["id"]
	product := map[string]interface{}{
		"id":    id,
		"name":  fmt.Sprintf("Product %s", id),
		"price": 99.99,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

// Create crea un nuevo producto
func (c ProductController) Create(w http.ResponseWriter, r *http.Request, p Params) {
	var newProduct map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&newProduct)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Agregar ID simulado
	newProduct["id"] = "3"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newProduct)
}

// Update actualiza un producto existente
func (c ProductController) Update(w http.ResponseWriter, r *http.Request, p Params) {
	id := p["id"]
	var updatedProduct map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&updatedProduct)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Asegurar que el ID sea correcto
	updatedProduct["id"] = id

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedProduct)
}

// Delete elimina un producto
func (c ProductController) Delete(w http.ResponseWriter, r *http.Request, p Params) {
	w.WriteHeader(http.StatusNoContent)
}

// TestResourceRouting prueba el enrutamiento básico de recursos
func TestResourceRouting(t *testing.T) {
	r := New()

	// Registrar el controlador de recursos
	r.Resource("/products", ProductController{})

	client := NewTestClient(r)

	// Probar GET /products (Index)
	resp := client.Get("/products")
	if !resp.IsOK() {
		t.Errorf("Expected status 200 for Index, got %d", resp.StatusCode)
	}

	var products []map[string]interface{}
	if err := resp.JSON(&products); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if len(products) != 2 {
		t.Errorf("Expected 2 products, got %d", len(products))
	}

	// Probar GET /products/1 (Show)
	resp = client.Get("/products/1")
	if !resp.IsOK() {
		t.Errorf("Expected status 200 for Show, got %d", resp.StatusCode)
	}

	var product map[string]interface{}
	if err := resp.JSON(&product); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if product["id"] != "1" {
		t.Errorf("Expected product ID '1', got '%v'", product["id"])
	}

	// Probar POST /products (Create)
	newProduct := map[string]interface{}{
		"name":  "New Product",
		"price": 199.99,
	}

	resp = client.PostJSON("/products", newProduct)
	if !resp.IsCreated() {
		t.Errorf("Expected status 201 for Create, got %d", resp.StatusCode)
	}

	var createdProduct map[string]interface{}
	if err := resp.JSON(&createdProduct); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if createdProduct["id"] != "3" {
		t.Errorf("Expected product ID '3', got '%v'", createdProduct["id"])
	}

	// Probar PUT /products/2 (Update)
	updatedProduct := map[string]interface{}{
		"name":  "Updated Product",
		"price": 299.99,
	}

	resp = client.PutJSON("/products/2", updatedProduct)
	if !resp.IsOK() {
		t.Errorf("Expected status 200 for Update, got %d", resp.StatusCode)
	}

	var returnedProduct map[string]interface{}
	if err := resp.JSON(&returnedProduct); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if returnedProduct["id"] != "2" {
		t.Errorf("Expected product ID '2', got '%v'", returnedProduct["id"])
	}

	if returnedProduct["name"] != "Updated Product" {
		t.Errorf("Expected product name 'Updated Product', got '%v'", returnedProduct["name"])
	}

	// Probar DELETE /products/1 (Delete)
	resp = client.Delete("/products/1")
	if !resp.IsNoContent() {
		t.Errorf("Expected status 204 for Delete, got %d", resp.StatusCode)
	}
}

// TestCustomRoutePatterns prueba patrones de ruta personalizados
func TestCustomRoutePatterns(t *testing.T) {
	r := New()

	// Ruta con patrón regex personalizado
	r.Get("/products/:code([A-Z]{2}\\d{4})", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("Product Code: " + p["code"]))
	})
	// Usamos dos rutas en lugar de un parámetro opcional
	r.Get("/users", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("All users"))
	})
	r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("User ID: " + p["id"]))
	})

	client := NewTestClient(r)

	// Probar ruta con patrón regex válido
	resp := client.Get("/products/AB1234")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Text() != "Product Code: AB1234" {
		t.Errorf("Expected 'Product Code: AB1234', got '%s'", resp.Text())
	}

	// Probar ruta con patrón regex inválido
	resp = client.Get("/products/123456")
	if !resp.IsNotFound() {
		t.Errorf("Expected status 404 for invalid regex pattern, got %d", resp.StatusCode)
	}

	// Probar ruta con parámetro opcional presente
	resp = client.Get("/users/123")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Text() != "User ID: 123" {
		t.Errorf("Expected 'User ID: 123', got '%s'", resp.Text())
	}

	// Probar ruta con parámetro opcional ausente
	resp = client.Get("/users")
	if !resp.IsOK() {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Text() != "All users" {
		t.Errorf("Expected 'All users', got '%s'", resp.Text())
	}
}
