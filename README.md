# MoraRouter

Un enrutador HTTP para Go ultra potente inspirado en Django, con características avanzadas para desarrollo rápido de APIs RESTful.

## ✨ Características

- **Enrutamiento avanzado**: Parámetros tipados, expresiones regulares, wildcards y comodines
- **Middleware incorporado**: Logging, recovery, CORS, cache, rate limiting, métricas, debug
- **Recursos RESTful**: Generación automática de rutas CRUD para recursos
- **Validación de datos**: Binding automático de JSON/XML/Form con validación de campos
- **OpenAPI/Swagger**: Generación automática de documentación
- **Herramientas de testing**: Cliente de pruebas para simplificar tests de API
- **Generador de código**: Scaffolding de controladores, modelos y recursos
- **Macros de rutas**: Patrones reutilizables para definición rápida de rutas
- **Internacionalización**: i18n de rutas según cabeceras Accept-Language
- **SPA y assets**: Soporte para aplicaciones de página única y archivos estáticos
- **GraphQL y WebSockets**: Integración sencilla con otros protocolos
- **Hot Reload**: Recarga automática de rutas sin reiniciar el servidor
- **Inspector de rutas**: UI web para explorar y probar rutas en runtime
- **Respuestas flexibles**: Soporte para JSON, XML, CSV, HTML y respuestas negociadas

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
    "time"
    
    "github.com/yourusername/mora-router/router"
)

func main() {
    // Crear router con middlewares y características
    r := router.New(
        router.WithLogging(),      // Log de peticiones
        router.WithRecovery(),     // Recuperación de panic
        router.WithCORS("*"),      // CORS configurado
        router.WithSwagger(),      // Endpoint OpenAPI en /openapi.json
        router.WithDebug(),        // Inspector en /_mora/inspector
        router.WithMetrics(),      // Métricas en /metrics
        router.WithCache(time.Minute), // Cacheo de respuestas
    )
    
    // Ruta simple con respuesta JSON
    r.Get("/hello/:name", func(w http.ResponseWriter, req *http.Request, p router.Params) {
        router.JSON(w, http.StatusOK, map[string]interface{}{
            "message": "¡Hola, " + p["name"] + "!",
            "timestamp": time.Now(),
        })
    })
    
    // Iniciar servidor
    log.Println("Servidor iniciado en :8080")
    log.Println("Inspector disponible en http://localhost:8080/_mora/inspector")
    http.ListenAndServe(":8080", r)
}
```

### Parámetros avanzados en rutas

MoraRouter soporta varios tipos de parámetros en rutas:

```go
// Parámetros básicos
r.Get("/users/:id", handler)                 // /users/123

// Expresiones regulares embebidas
r.Get("/posts/:year(\\d{4})/:month(\\d{2})", handler)  // /posts/2023/05

// Validación con sintaxis alternativa
r.Get("/products/{code:[A-Z]{3}\\d{4}}", handler)  // /products/ABC1234

// Parámetros comodín (capturar resto de la ruta)
r.Get("/files/*filepath", handler)  // /files/docs/manual.pdf
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

// Esto registra automáticamente:
// GET /users          -> Index()  - Listar todos
// GET /users/:id      -> Show()   - Mostrar uno
// POST /users         -> Create() - Crear nuevo
// PUT /users/:id      -> Update() - Actualizar
// DELETE /users/:id   -> Delete() - Eliminar
```

### Validación y binding automático

```go
// Con JSON
type CreateUserInput struct {
    Name  string `json:"name" validate:"required,min=3"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=18"`
}

r.Post("/users", router.BindJSON(func(w http.ResponseWriter, r *http.Request, p router.Params, input CreateUserInput) {
    // input ya está validado y procesado
    router.JSON(w, http.StatusCreated, map[string]interface{}{
        "message": "Usuario creado",
        "user": input,
    })
}))

// Con formularios y archivos
type UploadProfileInput struct {
    Name      string      `form:"name" validate:"required"`
    Email     string      `form:"email" validate:"required,email"`
    Avatar    *router.FormFile `form:"avatar"`
}

r.Post("/profile", router.BindForm(func(w http.ResponseWriter, r *http.Request, p router.Params, form *router.Form, input UploadProfileInput) {
    if form.HasErrors() {
        router.JSON(w, http.StatusBadRequest, form.GetErrors())
        return
    }
    
    // Guardar archivo si existe
    filePath := ""
    if input.Avatar != nil {
        filePath, _ = form.SaveFile("avatar", "")
    }
    
    router.JSON(w, http.StatusCreated, map[string]interface{}{
        "message": "Perfil actualizado",
        "avatar_path": filePath,
    })
}))
```

### Middleware y opciones

```go
// Middleware personalizado
func AuthMiddleware(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, p router.Params) {
        token := r.Header.Get("Authorization")
        if token == "" {
            router.Error(w, http.StatusUnauthorized, "Token requerido")
            return
        }
        // Aquí validaríamos el token...
        next(w, r, p)
    }
}

// Aplicar a rutas específicas
r.With(AuthMiddleware).Get("/admin", adminHandler)

// Aplicar a un grupo de rutas
admin := r.Group("/admin")
admin.Use(AuthMiddleware)
admin.Get("/dashboard", dashboardHandler)
admin.Get("/users", usersHandler)

// JWT integrado
r.With(router.WithJWT("secret-key"))
r.With(router.RequireRole("admin")).Get("/admin", adminHandler)
```

### Grupos de rutas y versionado API

```go
// Agrupar rutas bajo un prefijo
v1 := r.Group("/api/v1")
v1.Get("/users", listUsersV1)
v1.Get("/products", listProductsV1)

v2 := r.Group("/api/v2")
v2.Get("/users", listUsersV2)

// Versionado automático por cabecera
r := router.New(router.WithAPIVersioning("X-API-Version", "1"))
// Las rutas se reescriben automáticamente según la cabecera
```

### Uso de macros de rutas

```go
// Registrar una macro personalizada
router.RegisterMacro("paginated", "/:page(\\d+)", []string{"GET"}, rateLimitMiddleware(100, time.Minute))

// Usar la macro
r.UseMacro("/users", "paginated", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    page := p["page"] // La página viene del parámetro :page
    router.JSON(w, http.StatusOK, map[string]string{"page": page})
})

// Crear API CRUD con macros predefinidas
r.UseMacro("/products", "list", listProductsHandler)
r.UseMacro("/products", "detail", getProductHandler)
r.UseMacro("/products", "create", createProductHandler)
r.UseMacro("/products", "update", updateProductHandler)
r.UseMacro("/products", "delete", deleteProductHandler)
```

### Respuestas en múltiples formatos

```go
// Cliente solicitando diferentes formatos
// Render flexible
render := router.NewRender()

// JSON (con indentación)
render.JSON(w, http.StatusOK, data)

// XML
render.XML(w, http.StatusOK, data)

// CSV desde slice de structs
render.CSV(w, http.StatusOK, users)

// HTML con plantillas
render.HTMLTemplates = template.Must(template.ParseGlob("templates/*.html"))
render.HTML(w, http.StatusOK, "user.html", user)

// Negociación de contenido
render.Negotiate(w, r, http.StatusOK, data)
```

### Hot Reload

```go
// Recarga automática de rutas desde archivo JSON
r := router.New(router.WithHotReload("routes.json", 5 * time.Second))

// Archivo routes.json:
// {
//   "routes": [
//     { "method": "GET", "pattern": "/products", "name": "products.list" },
//     { "method": "GET", "pattern": "/products/:id", "name": "products.show" }
//   ],
//   "groups": {
//     "api": "/api"
//   }
// }
```

### Testing simplificado

```go
func TestUserAPI(t *testing.T) {
    r := router.New()
    r.Resource("/users", UserController{})
    
    client := router.NewTestClient(r)
    
    // Autenticación para todas las peticiones
    client.WithAuth("test-token")
    
    // GET /users/42
    resp := client.Get("/users/42")
    if !resp.IsOK() {
        t.Errorf("Expected 200 status, got %d", resp.Status())
    }
    
    // Deserializar respuesta JSON
    var user map[string]interface{}
    resp.JSON(&user)
    
    // POST /users con payload JSON
    createResp := client.Post("/users", map[string]interface{}{
        "name": "Nuevo Usuario",
        "email": "nuevo@example.com",
        "age": 25,
    })
    
    if !createResp.IsCreated() {
        t.Errorf("Expected 201 status, got %d", createResp.Status())
    }
}
```

### Generador de código

```go
// Generar controlador
gen := router.NewRouteGenerator(r)
controllerCode, _ := gen.GenerateController("product")

// Generar modelo
fields := map[string]string{
    "name": "string",
    "price": "float64",
    "stock": "int",
}
modelCode, _ := gen.GenerateModel("product", fields)

// Generar pruebas
endpoints := []string{
    "GET /products",
    "GET /products/:id",
    "POST /products",
}
testCode, _ := gen.GenerateTests("Product", endpoints)
```

### Inspector de rutas

El inspector web de rutas está disponible en `/_mora/inspector` cuando se activa con `router.WithDebug()`. Proporciona:

- Lista de todas las rutas registradas
- Información detallada de rutas, parámetros y métodos
- Consola para probar peticiones en tiempo real
- Información de depuración sobre el router

### Rendimiento y métricas

```go
// Añadir métricas
r := router.New(router.WithMetrics())

// Endpoint /metrics con formato compatible con Prometheus
// Ejemplo de métrica:
// http_handler_latency_seconds_average 0.002345
// http_handler_requests_total 42
```

### WebSockets

```go
// Requiere gorilla/websocket
import "github.com/gorilla/websocket"

// Activar soporte WebSocket con gorilla
r := router.New(router.WithGorillaWebSocket())

// Chat simple
r := router.New(
    router.WithGorillaWebSocket(),
    router.WithChatRoom("/chat")
)
// Accesible en http://localhost:8080/chat-ui

// WebSocket handler personalizado
r := router.New(
    router.WithGorillaWebSocket(),
    router.WithWebSocketHandler(router.WebSocketConfig{
        Path: "/ws",
        // Configuraciones opcionales
        PingInterval: 30, // segundos
        MaxMessageSize: 4096, // bytes
        AllowedOrigins: []string{"example.com"}, // orígenes permitidos
        // Manejadores
        MessageHandler: func(conn *router.WebSocketConnection, msg []byte) {
            // Procesar mensaje y enviar respuesta
            conn.SendJSON(map[string]interface{}{
                "echo": string(msg),
                "time": time.Now(),
            })
        },
        OnConnect: func(conn *router.WebSocketConnection, req *http.Request, p router.Params) {
            log.Printf("Nueva conexión: %s", conn.ID)
        },
    })
)

// Salas de chat dinámicas
r := router.New(
    router.WithGorillaWebSocket(),
    router.WithRoomProvider("/api/rooms", router.WebSocketRoomOption{
        MaxConnections: 100,
        MessageHandler: func(conn *router.WebSocketConnection, msg []byte) {
            // Retransmitir mensaje a todos en la sala
            conn.Hub.Broadcast(msg)
        },
    })
)

// Soporte para múltiples endpoints WebSocket
r := router.New(
    router.WithGorillaWebSocket(),
    router.WithWebSockets(map[string]router.WebSocketHandlerFunc{
        "/chat": chatHandler,
        "/notifications": notificationsHandler,
        "/events": eventsHandler,
    })
)
```

## 🌟 Ejemplo completo

Revisa el directorio `examples/resource-demo` para ver una API completa implementada con MoraRouter.

## 📚 Documentación completa

Para más información sobre todas las características y opciones avanzadas, consulta [la documentación completa](https://github.com/yourusername/mora-router/docs).

## 📄 Licencia

MIT
