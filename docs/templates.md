## Plantillas y Assets

MoraRouter proporciona un sistema de plantillas mejorado para facilitar el desarrollo de aplicaciones web:

```go
// Configuración básica de plantillas
router.WithTemplates("templates")(r)

// Renderizar una plantilla desde un handler
r.Get("/", func(w http.ResponseWriter, req *http.Request, p router.Params) {
    // Los datos estarán disponibles en la plantilla
    data := map[string]interface{}{
        "Title": "Mi Aplicación",
        "User": currentUser,
    }
    
    // Renderizar plantilla con datos
    router.RenderTemplate(w, req, "index.html", data)
})
```

### Configuración Avanzada de Plantillas

```go
// Crear un gestor de plantillas personalizado
tm := router.NewTemplateManager("templates")

// Configurar layout y parciales
tm.WithLayout("layout.html")
   .WithPartials("partials/header.html", "partials/footer.html")
   
// Añadir CSS automático
tm.WithCSS("mainCSS", "styles/main.css")
   .WithCSS("darkThemeCSS", "styles/dark.css")
   
// Añadir funciones personalizadas
tm.WithFuncs(template.FuncMap{
    "formatDate": func(t time.Time) string {
        return t.Format("02/01/2006")
    },
})

// Opciones de desarrollo
tm.DisableCache() // Útil durante desarrollo
```

### Estructura de Archivos de Plantillas

```
templates/
  ├── index.html    // Página principal
  ├── about.html    // Página acerca de
  ├── layout.html   // Layout común (opcional)
  ├── style.css     // CSS principal (auto-incluido)
  └── partials/     // Parciales reutilizables
      ├── header.html
      └── footer.html
```

### Uso en Plantillas HTML

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <style>
        /* CSS automáticamente disponible como variable */
        {{.mainCSS}}
        
        /* CSS condicional */
        {{if .darkMode}}
            {{.darkThemeCSS}}
        {{end}}
    </style>
</head>
<body>
    <h1>{{.Title}}</h1>
    
    <!-- Incluir parciales -->
    {{template "header" .}}
    
    <!-- Contenido específico de la página -->
    <main>{{.Content}}</main>
    
    <!-- Funciones personalizadas -->
    <p>Fecha: {{formatDate .CurrentDate}}</p>
    
    {{template "footer" .}}
</body>
</html>
```
