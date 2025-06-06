package router

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// TestClient proporciona una API fluida para pruebas de integración con el router.
type TestClient struct {
	Router  http.Handler
	headers map[string]string
}

// NewTestClient crea un nuevo cliente para testing con el router dado.
func NewTestClient(router http.Handler) *TestClient {
	return &TestClient{
		Router:  router,
		headers: make(map[string]string),
	}
}

// WithHeader configura una cabecera HTTP para todas las peticiones.
func (c *TestClient) WithHeader(key, value string) *TestClient {
	c.headers[key] = value
	return c
}

// WithAuth configura la cabecera de autorización con un token.
func (c *TestClient) WithAuth(token string) *TestClient {
	c.headers["Authorization"] = "Bearer " + token
	return c
}

// WithContentType configura el tipo de contenido de la petición.
func (c *TestClient) WithContentType(contentType string) *TestClient {
	c.headers["Content-Type"] = contentType
	return c
}

// TestResponse encapsula una respuesta HTTP para pruebas.
type TestResponse struct {
	StatusCode int
	Body       []byte
	Header     http.Header
	recorder   *httptest.ResponseRecorder
}

// Status devuelve el código de estado HTTP de la respuesta.
func (r *TestResponse) Status() int {
	return r.StatusCode
}

// JSON deserializa la respuesta JSON en un objeto v.
func (r *TestResponse) JSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

// Text devuelve el cuerpo de la respuesta como string.
func (r *TestResponse) Text() string {
	return string(r.Body)
}

// IsOK verifica si el código de estado es 200 OK.
func (r *TestResponse) IsOK() bool {
	return r.StatusCode == http.StatusOK
}

// IsCreated verifica si el código de estado es 201 Created.
func (r *TestResponse) IsCreated() bool {
	return r.StatusCode == http.StatusCreated
}

// IsNotFound verifica si el código de estado es 404 Not Found.
func (r *TestResponse) IsNotFound() bool {
	return r.StatusCode == http.StatusNotFound
}

// IsServerError verifica si el código de estado es 5xx.
func (r *TestResponse) IsServerError() bool {
	return r.StatusCode >= 500 && r.StatusCode < 600
}

// IsClientError verifica si el código de estado es 4xx.
func (r *TestResponse) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsNoContent verifica si el código de estado es 204 No Content.
func (r *TestResponse) IsNoContent() bool {
	return r.StatusCode == http.StatusNoContent
}

// IsAccepted verifica si el código de estado es 202 Accepted.
func (r *TestResponse) IsAccepted() bool {
	return r.StatusCode == http.StatusAccepted
}

// IsForbidden verifica si el código de estado es 403 Forbidden.
func (r *TestResponse) IsForbidden() bool {
	return r.StatusCode == http.StatusForbidden
}

// IsUnauthorized verifica si el código de estado es 401 Unauthorized.
func (r *TestResponse) IsUnauthorized() bool {
	return r.StatusCode == http.StatusUnauthorized
}

// HasHeader verifica si existe una cabecera HTTP.
func (r *TestResponse) HasHeader(header string) bool {
	_, ok := r.Header[header]
	return ok
}

// DecodeJSON deserializa una respuesta JSON en el objeto dado.
func (r *TestResponse) DecodeJSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

// Get hace una petición GET a la ruta dada.
func (c *TestClient) Get(path string) *TestResponse {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.exec(req)
}

// Post hace una petición POST a la ruta dada con el payload proporcionado.
func (c *TestClient) Post(path string, payload interface{}) *TestResponse {
	var body io.Reader
	var contentType string

	switch p := payload.(type) {
	case string:
		body = strings.NewReader(p)
		contentType = "text/plain"
	case []byte:
		body = bytes.NewReader(p)
		contentType = "application/octet-stream"
	default:
		data, err := json.Marshal(p)
		if err != nil {
			panic("failed to marshal JSON: " + err.Error())
		}
		body = bytes.NewReader(data)
		contentType = "application/json"
	}

	req := httptest.NewRequest(http.MethodPost, path, body)
	if _, ok := c.headers["Content-Type"]; !ok {
		req.Header.Set("Content-Type", contentType)
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.exec(req)
}

// Put hace una petición PUT a la ruta dada con el payload proporcionado.
func (c *TestClient) Put(path string, payload interface{}) *TestResponse {
	var body io.Reader
	var contentType string

	switch p := payload.(type) {
	case string:
		body = strings.NewReader(p)
		contentType = "text/plain"
	case []byte:
		body = bytes.NewReader(p)
		contentType = "application/octet-stream"
	default:
		data, err := json.Marshal(p)
		if err != nil {
			panic("failed to marshal JSON: " + err.Error())
		}
		body = bytes.NewReader(data)
		contentType = "application/json"
	}

	req := httptest.NewRequest(http.MethodPut, path, body)
	if _, ok := c.headers["Content-Type"]; !ok {
		req.Header.Set("Content-Type", contentType)
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.exec(req)
}

// Delete hace una petición DELETE a la ruta dada.
func (c *TestClient) Delete(path string) *TestResponse {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.exec(req)
}

// Patch hace una petición PATCH a la ruta dada con el payload proporcionado.
func (c *TestClient) Patch(path string, payload interface{}) *TestResponse {
	var body io.Reader
	var contentType string

	switch p := payload.(type) {
	case string:
		body = strings.NewReader(p)
		contentType = "text/plain"
	case []byte:
		body = bytes.NewReader(p)
		contentType = "application/octet-stream"
	default:
		data, err := json.Marshal(p)
		if err != nil {
			panic("failed to marshal JSON: " + err.Error())
		}
		body = bytes.NewReader(data)
		contentType = "application/json"
	}

	req := httptest.NewRequest(http.MethodPatch, path, body)
	if _, ok := c.headers["Content-Type"]; !ok {
		req.Header.Set("Content-Type", contentType)
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.exec(req)
}

// exec ejecuta la petición HTTP y devuelve una TestResponse.
func (c *TestClient) exec(req *http.Request) *TestResponse {
	rr := httptest.NewRecorder()
	c.Router.ServeHTTP(rr, req)
	return &TestResponse{
		StatusCode: rr.Code,
		Body:       rr.Body.Bytes(),
		Header:     rr.Header(),
		recorder:   rr,
	}
}

// Options hace una petición OPTIONS a la ruta dada.
func (c *TestClient) Options(path string) *TestResponse {
	req := httptest.NewRequest(http.MethodOptions, path, nil)
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.exec(req)
}

// GetJSON hace una petición GET y espera una respuesta JSON.
func (c *TestClient) GetJSON(path string) *TestResponse {
	return c.WithHeader("Accept", "application/json").Get(path)
}

// PostJSON hace una petición POST con un cuerpo JSON.
func (c *TestClient) PostJSON(path string, payload interface{}) *TestResponse {
	data, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal JSON: " + err.Error())
	}

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.exec(req)
}

// PutJSON hace una petición PUT con un cuerpo JSON.
func (c *TestClient) PutJSON(path string, payload interface{}) *TestResponse {
	data, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal JSON: " + err.Error())
	}

	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.exec(req)
}

// PatchJSON hace una petición PATCH con un cuerpo JSON.
func (c *TestClient) PatchJSON(path string, payload interface{}) *TestResponse {
	data, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal JSON: " + err.Error())
	}

	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.exec(req)
}
