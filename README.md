# MoraRouter

Un enrutador HTTP para Go ultra potente inspirado en Django, con características avanzadas para desarrollo rápido de APIs RESTful.

## ✨ Características

- **Enrutamiento avanzado**: Parámetros tipados, expresiones regulares, wildcards y comodines
- **Middleware incorporado**: Logging, recovery, CORS, cache, rate limiting, métricas
- **Recursos RESTful**: Generación automática de rutas CRUD para recursos
- **Validación de datos**: Binding automático de JSON/XML con validación de campos
- **OpenAPI/Swagger**: Generación automática de documentación
- **Herramientas de testing**: Cliente de pruebas para simplificar tests de API
- **Generador de código**: CLI para scaffolding de controladores y recursos
- **Macros de rutas**: Patrones reutilizables para definición rápida de rutas
- **Internacionalización**: i18n de rutas según cabeceras Accept-Language
- **SPA y assets**: Soporte para aplicaciones de página única y archivos estáticos
- **GraphQL y WebSockets**: Integración sencilla con otros protocolos

## 🚀 Instalación

```bash
go get -u github.com/yourusername/mora-router
```

## 📖 Guía Rápida

### Router básico

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/yourusername/mora-router/router"
)

func main() {
    // Crear router con middlewares predeterminados
    r := router.New(
        router.WithLogging(),
        router.WithRecovery(),
        router.WithCORS("*"),
    )
    
    // Ruta con parámetros
    r.Get("/users/:id", func(w http.ResponseWriter, req *http.Request, p router.Params) {
        router.JSON(w, http.StatusOK, map[string]string{
            "id": p["id"],
            "message": "Usuario encontrado",
        })
    })
    
    // Iniciar servidor
    log.Println("Servidor iniciado en :8080")
    http.ListenAndServe(":8080", r)
}
```

### Recursos RESTful automáticos

```go
// UserController implementa un controlador RESTful
type UserController struct {
    router.DefaultController
}

// Show muestra un usuario por ID
func (c UserController) Show(w http.ResponseWriter, r *http.Request, p router.Params) {
    id := p["id"]
    router.JSON(w, http.StatusOK, map[string]interface{}{
        "id": id,
        "name": "Usuario " + id,
    })
}

// En func main():
r.Resource("/users", UserController{})
```

### Validación y binding automático

```go
type CreateUserInput struct {
    Name  string `json:"name" validate:"required,min=3"`
    Email string `json:"email" validate:"required"`
    Age   int    `json:"age" validate:"min=18"`
}

r.Post("/users", router.BindJSON(func(w http.ResponseWriter, r *http.Request, p router.Params, input CreateUserInput) {
    // input ya está validado y procesado
    router.JSON(w, http.StatusCreated, map[string]interface{}{
        "message": "Usuario creado",
        "user": input,
    })
}))
```

### Grupos de rutas y versionado API

```go
// Agrupar rutas bajo un prefijo
v1 := r.Group("/api/v1")
v1.Get("/users", listUsersV1)
v1.Get("/products", listProductsV1)

v2 := r.Group("/api/v2")
v2.Get("/users", listUsersV2)
```

### Uso de macros de rutas

```go
// Registrar una macro personalizada
router.RegisterMacro("paginated", "/:page(\\d+)", []string{"GET"})

// Usar la macro
r.UseMacro("/users", "paginated", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    page := p["page"] // La página viene del parámetro :page
    router.JSON(w, http.StatusOK, map[string]string{"page": page})
})
```

### Testing simplificado

```go
func TestUserAPI(t *testing.T) {
    r := router.New()
    r.Resource("/users", UserController{})
    
    client := router.NewTestClient(r)
    
    // GET /users/42
    resp := client.Get("/users/42")
    if !resp.IsOK() {
        t.Errorf("Expected 200 status, got %d", resp.Status())
    }
    
    // POST /users con payload JSON
    createResp := client.Post("/users", map[string]string{
        "name": "Nuevo Usuario",
        "email": "nuevo@example.com",
    })
    
    if !createResp.IsCreated() {
        t.Errorf("Expected 201 status, got %d", createResp.Status())
    }
}
```

### CLI para generación de código

```bash
go run cmd/genesis/main.go resource --name producto
```

## 📚 Documentación completa

Para más información sobre todas las características y opciones avanzadas, consulta [la documentación completa](https://github.com/yourusername/mora-router/docs).

## 📄 Licencia

MIT
