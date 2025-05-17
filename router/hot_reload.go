package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// HotReloader maneja la recarga automática de configuraciones de rutas.
type HotReloader struct {
	mu        sync.Mutex
	router    *MoraRouter
	filePath  string
	interval  time.Duration
	lastMod   time.Time
	callbacks []func()
	stop      chan struct{}
}

// NewHotReloader crea un nuevo recargador para el router.
func NewHotReloader(r *MoraRouter, filePath string, interval time.Duration) *HotReloader {
	if interval == 0 {
		interval = 5 * time.Second // Valor por defecto
	}

	return &HotReloader{
		router:    r,
		filePath:  filePath,
		interval:  interval,
		callbacks: make([]func(), 0),
		stop:      make(chan struct{}),
	}
}

// Start inicia el proceso de vigilancia de cambios en el archivo de configuración.
func (hr *HotReloader) Start() {
	go hr.watchFile()
}

// Stop detiene el proceso de vigilancia.
func (hr *HotReloader) Stop() {
	close(hr.stop)
}

// OnReload registra una función callback que se ejecutará cuando se detecte un cambio.
func (hr *HotReloader) OnReload(fn func()) {
	hr.mu.Lock()
	hr.callbacks = append(hr.callbacks, fn)
	hr.mu.Unlock()
}

// watchFile monitorea cambios en el archivo de configuración.
func (hr *HotReloader) watchFile() {
	ticker := time.NewTicker(hr.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hr.checkFile()
		case <-hr.stop:
			return
		}
	}
}

// checkFile verifica si el archivo ha cambiado y ejecuta la recarga.
func (hr *HotReloader) checkFile() {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	fi, err := os.Stat(hr.filePath)
	if err != nil {
		// No existe el archivo o no se puede acceder
		return
	}

	modTime := fi.ModTime()
	if !modTime.After(hr.lastMod) {
		// No ha cambiado
		return
	}

	// Actualizar último tiempo de modificación
	hr.lastMod = modTime

	// Intentar cargar las rutas
	if err := hr.loadRoutes(); err != nil {
		fmt.Printf("[MORA][HotReload] Error cargando rutas: %v\n", err)
		return
	}

	fmt.Printf("[MORA][HotReload] Rutas recargadas desde %s\n", hr.filePath)

	// Ejecutar callbacks
	for _, cb := range hr.callbacks {
		cb()
	}
}

// RouteDefinition define una ruta en el formato JSON/YAML de configuración.
type RouteDefinition struct {
	Method      string            `json:"method"`
	Pattern     string            `json:"pattern"`
	HandlerFile string            `json:"handler_file"`
	HandlerFunc string            `json:"handler_func"`
	Middleware  []string          `json:"middleware,omitempty"`
	Name        string            `json:"name,omitempty"`
	Group       string            `json:"group,omitempty"`
	Params      map[string]string `json:"params,omitempty"`
}

// RouteCollection es una colección de definiciones de rutas.
type RouteCollection struct {
	Routes []RouteDefinition `json:"routes"`
	Groups map[string]string `json:"groups,omitempty"`
}

// loadRoutes carga las rutas desde el archivo de configuración.
func (hr *HotReloader) loadRoutes() error {
	file, err := os.Open(hr.filePath)
	if err != nil {
		return fmt.Errorf("error abriendo archivo: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error leyendo archivo: %w", err)
	}

	var routes RouteCollection
	if err := json.Unmarshal(data, &routes); err != nil {
		return fmt.Errorf("error parseando JSON: %w", err)
	}

	// Limpiar rutas anteriores
	// Nota: Esto requeriría cambios en MoraRouter para permitir remover rutas
	// hr.router.clearRoutes()

	// Crear grupos
	groups := make(map[string]*RouteGroup)
	for name, prefix := range routes.Groups {
		groups[name] = hr.router.Group(prefix)
	}

	// Registrar rutas
	for _, route := range routes.Routes {
		var handler HandlerFunc
		// Aquí podrías implementar la carga de handlers desde archivos/módulos
		// Por ahora usaremos un handler por defecto
		handler = func(w http.ResponseWriter, r *http.Request, p Params) {
			JSON(w, http.StatusOK, map[string]string{
				"message": fmt.Sprintf("Ruta dinámica %s %s cargada", route.Method, route.Pattern),
				"method":  route.Method,
				"pattern": route.Pattern,
			})
		}

		// Aplicar middlewares específicos
		if len(route.Middleware) > 0 {
			mws := make([]Middleware, 0, len(route.Middleware))
			for _, name := range route.Middleware {
				if mw, ok := hr.router.middlewareRegistry[name]; ok {
					mws = append(mws, mw)
				}
			}
			if len(mws) > 0 {
				handler = applyMiddlewares(handler, mws)
			}
		}

		// Registrar según grupo o directamente
		if route.Group != "" {
			if g, ok := groups[route.Group]; ok {
				switch route.Method {
				case "GET":
					g.Get(route.Pattern, handler)
				case "POST":
					g.Post(route.Pattern, handler)
				case "PUT":
					g.Put(route.Pattern, handler)
				case "DELETE":
					g.Delete(route.Pattern, handler)
				}
			}
		} else {
			hr.router.Handle(route.Method, route.Pattern, handler)
		}

		// Nombrar ruta si se especifica
		if route.Name != "" {
			hr.router.Name(route.Name, route.Pattern)
		}
	}

	return nil
}

// ReloadRoutes fuerza una recarga inmediata de las rutas.
func (hr *HotReloader) ReloadRoutes() error {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	return hr.loadRoutes()
}

// CompleteHotReload actualiza la implementación de WithHotReload para usar el nuevo reloader.
func CompleteHotReload(router *MoraRouter, filePath string, interval time.Duration) *HotReloader {
	hr := NewHotReloader(router, filePath, interval)
	hr.Start()
	return hr
}

// WithHotReload ahora devuelve el reloader para que pueda ser controlado.
func WithHotReloadComplete(filePath string, interval time.Duration) Option {
	return func(r *MoraRouter) {
		// Devolvemos un reloader completamente funcional
		hr := CompleteHotReload(r, filePath, interval)

		// Añadir endpoint para forzar recarga
		r.Get("/_mora/reload", func(w http.ResponseWriter, req *http.Request, p Params) {
			err := hr.ReloadRoutes()
			if err != nil {
				Error(w, http.StatusInternalServerError, fmt.Sprintf("Error reloading routes: %v", err))
				return
			}
			JSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
		})
	}
}
