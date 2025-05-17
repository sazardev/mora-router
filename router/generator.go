package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"text/template"
)

// RouteGenerator genera código para controladores y rutas.
type RouteGenerator struct {
	Router        *MoraRouter
	TemplatePath  string
	OutputPath    string
	ResourcesPath string
}

// NewRouteGenerator crea un nuevo generador de rutas.
func NewRouteGenerator(r *MoraRouter) *RouteGenerator {
	return &RouteGenerator{
		Router:        r,
		TemplatePath:  "templates",
		OutputPath:    "generated",
		ResourcesPath: "resources",
	}
}

// GenerateController genera código para un controlador.
func (g *RouteGenerator) GenerateController(name string) (string, error) {
	const controllerTpl = `package controllers

import (
	"net/http"

	"mora-router/router"
)

// {{.Name}}Controller implementa un controlador RESTful para {{.Resource}}.
type {{.Name}}Controller struct {
	router.DefaultController
}

// Index lista todos los {{.Resource}}.
func (c {{.Name}}Controller) Index(w http.ResponseWriter, r *http.Request, p router.Params) {
	router.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "Lista de {{.Resource}}",
		"data":    []map[string]interface{}{},
	})
}

// Show muestra un {{.Resource}} por ID.
func (c {{.Name}}Controller) Show(w http.ResponseWriter, r *http.Request, p router.Params) {
	id := p["id"]
	router.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "Detalle de {{.Resource}}",
		"id":      id,
	})
}

// Create crea un nuevo {{.Resource}}.
func (c {{.Name}}Controller) Create(w http.ResponseWriter, r *http.Request, p router.Params) {
	router.JSON(w, http.StatusCreated, map[string]interface{}{
		"message": "{{.Resource}} creado",
	})
}

// Update actualiza un {{.Resource}} por ID.
func (c {{.Name}}Controller) Update(w http.ResponseWriter, r *http.Request, p router.Params) {
	id := p["id"]
	router.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "{{.Resource}} actualizado",
		"id":      id,
	})
}

// Delete elimina un {{.Resource}} por ID.
func (c {{.Name}}Controller) Delete(w http.ResponseWriter, r *http.Request, p router.Params) {
	id := p["id"]
	router.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "{{.Resource}} eliminado",
		"id":      id,
	})
}
`
	tpl, err := template.New("controller").Parse(controllerTpl)
	if err != nil {
		return "", err
	}

	data := struct {
		Name     string
		Resource string
	}{
		Name:     strings.Title(name),
		Resource: name,
	}

	var output strings.Builder
	if err := tpl.Execute(&output, data); err != nil {
		return "", err
	}

	return output.String(), nil
}

// GenerateModel genera código para un modelo.
func (g *RouteGenerator) GenerateModel(name string, fields map[string]string) (string, error) {
	const modelTpl = `package models

import (
	"time"
)

// {{.Name}} representa un modelo de {{.Name}}.
type {{.Name}} struct {
	ID        string    ` + "`json:\"id\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`" + `
{{range .Fields}}
	{{.Name}} {{.Type}} ` + "`json:\"{{.JSONName}}\"`" + `{{end}}
}
`

	tpl, err := template.New("model").Parse(modelTpl)
	if err != nil {
		return "", err
	}

	type Field struct {
		Name     string
		Type     string
		JSONName string
	}

	fieldsList := make([]Field, 0, len(fields))
	for fieldName, fieldType := range fields {
		fieldsList = append(fieldsList, Field{
			Name:     strings.Title(fieldName),
			Type:     fieldType,
			JSONName: strings.ToLower(fieldName),
		})
	}

	data := struct {
		Name   string
		Fields []Field
	}{
		Name:   strings.Title(name),
		Fields: fieldsList,
	}

	var output strings.Builder
	if err := tpl.Execute(&output, data); err != nil {
		return "", err
	}

	return output.String(), nil
}

// GenerateTests genera código para pruebas de API.
func (g *RouteGenerator) GenerateTests(name string, endpoints []string) (string, error) {
	const testTpl = `package tests

import (
	"net/http"
	"testing"

	"mora-router/router"
)

func Test{{.Name}}API(t *testing.T) {
	r := router.New()
	r.Resource("/{{.Resource}}", {{.Name}}Controller{})
	
	client := router.NewTestClient(r)
	
{{range .Endpoints}}
	// Test {{.Method}} {{.Path}}
	resp := client.{{.Method}}("{{.Path}}", nil)
	if !resp.Is{{.ExpectedStatus}}() {
		t.Errorf("Expected {{.ExpectedStatusCode}} status, got %d", resp.Status())
	}
{{end}}
}
`

	type Endpoint struct {
		Method             string
		Path               string
		ExpectedStatus     string
		ExpectedStatusCode int
	}

	endpointsList := make([]Endpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		parts := strings.Split(endpoint, " ")
		method := "GET"
		path := endpoint
		expectedStatus := "OK"
		expectedStatusCode := 200

		if len(parts) > 1 {
			method = parts[0]
			path = parts[1]

			switch method {
			case "POST":
				expectedStatus = "Created"
				expectedStatusCode = 201
			case "DELETE":
				expectedStatus = "NoContent"
				expectedStatusCode = 204
			}
		}

		endpointsList = append(endpointsList, Endpoint{
			Method:             method,
			Path:               path,
			ExpectedStatus:     expectedStatus,
			ExpectedStatusCode: expectedStatusCode,
		})
	}

	tpl, err := template.New("test").Parse(testTpl)
	if err != nil {
		return "", err
	}

	data := struct {
		Name      string
		Resource  string
		Endpoints []Endpoint
	}{
		Name:      strings.Title(name),
		Resource:  name,
		Endpoints: endpointsList,
	}

	var output strings.Builder
	if err := tpl.Execute(&output, data); err != nil {
		return "", err
	}

	return output.String(), nil
}

// MockResponseWriter es un ResponseWriter para pruebas.
type MockResponseWriter struct {
	headers http.Header
	body    []byte
	status  int
}

func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		headers: make(http.Header),
		status:  http.StatusOK,
	}
}

func (m *MockResponseWriter) Header() http.Header {
	return m.headers
}

func (m *MockResponseWriter) Write(body []byte) (int, error) {
	m.body = append(m.body, body...)
	return len(body), nil
}

func (m *MockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

func (m *MockResponseWriter) Result() *http.Response {
	res := &http.Response{
		StatusCode: m.status,
		Header:     m.headers,
		Body:       io.NopCloser(strings.NewReader(string(m.body))),
	}
	return res
}

// RouteDebugger imprime información de depuración sobre una ruta.
type RouteDebugger struct {
	router *MoraRouter
}

func NewRouteDebugger(r *MoraRouter) *RouteDebugger {
	return &RouteDebugger{router: r}
}

// PrintRoutes imprime información sobre todas las rutas registradas.
func (d *RouteDebugger) PrintRoutes() {
	fmt.Println("=== MoraRouter Registered Routes ===")
	fmt.Printf("Total routes: %d\n", len(d.router.routes))

	for i, rt := range d.router.routes {
		fmt.Printf("%d. %s %s\n", i+1, rt.method, rt.pattern)

		fmt.Print("   Parameters: ")
		params := []string{}
		for _, seg := range rt.segments {
			if seg.name != "" {
				param := seg.name
				if seg.regex != nil {
					param += " (regex: " + seg.regex.String() + ")"
				}
				if seg.wildcard {
					param += " (wildcard)"
				}
				params = append(params, param)
			}
		}
		fmt.Println(strings.Join(params, ", "))
	}

	fmt.Println("===================================")
}

// TraceRoute realiza un seguimiento de cómo se procesaría una ruta.
func (d *RouteDebugger) TraceRoute(method, path string) {
	fmt.Printf("=== Tracing route %s %s ===\n", method, path)

	// Separar segmentos de la ruta
	pathSegs := splitPath(path)
	fmt.Printf("Path segments: %v\n", pathSegs)

	// Buscar rutas que coincidan
	fmt.Println("\nMatching routes:")
	found := false

	for i, rt := range d.router.routes {
		params := make(Params)
		if matchSegments(rt.segments, pathSegs, params) {
			fmt.Printf("%d. %s %s\n", i+1, rt.method, rt.pattern)

			if rt.method == method {
				fmt.Println("   ✓ Method matches!")
				fmt.Println("   Parameters extracted:")
				for k, v := range params {
					fmt.Printf("     %s = %s\n", k, v)
				}
				found = true
			} else {
				fmt.Printf("   ✗ Method doesn't match (expected %s, got %s)\n", rt.method, method)
			}
		}
	}

	if !found {
		fmt.Println("No matching route found!")
	}

	fmt.Println("===================================")
}

// SimulateRequest simula una petición HTTP a la ruta dada.
func (d *RouteDebugger) SimulateRequest(method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req := httptest.NewRequest(method, path, body)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := NewMockResponseWriter()
	d.router.ServeHTTP(w, req)

	return w.Result(), nil
}

// ExportOpenAPI exporta la especificación OpenAPI del router.
func (d *RouteDebugger) ExportOpenAPI(pretty bool) (string, error) {
	spec := d.router.BuildOpenAPISpec()

	var data []byte
	var err error

	if pretty {
		data, err = json.MarshalIndent(spec, "", "  ")
	} else {
		data, err = json.Marshal(spec)
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}
