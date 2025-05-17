package router

import (
	"fmt"
	"net/http"
	"os"
	"sort"
)

// Debug creates a debugging middleware that adds request inspection
func WithDebug() Option {
	return func(r *MoraRouter) {
		r.middlewareRegistry["debug"] = debugMiddleware
		r.middlewares = append(r.middlewares, debugMiddleware)

		// Register inspector at /_mora/debug
		r.Get("/_mora/debug", r.debugHandler)
		r.Get("/_mora/routes", r.routesHandler)
	}
}

// debugMiddleware loguea información detallada de las peticiones si se activa con la cabecera X-Mora-Debug
func debugMiddleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		if r.Header.Get("X-Mora-Debug") == "1" || r.URL.Query().Get("_debug") == "1" {
			// Add debug header to response
			w.Header().Set("X-Mora-Debug", "active")

			// Log detailed request info
			fmt.Printf("[MORA DEBUG] Request: %s %s\n", r.Method, r.URL.Path)
			fmt.Printf("[MORA DEBUG] Headers: %v\n", r.Header)
			fmt.Printf("[MORA DEBUG] Params: %v\n", p)
			fmt.Printf("[MORA DEBUG] Query: %v\n", r.URL.Query())
		}

		next(w, r, p)
	}
}

// routesHandler devuelve todas las rutas registradas en formato JSON
func (r *MoraRouter) routesHandler(w http.ResponseWriter, req *http.Request, p Params) {
	type RouteInfo struct {
		Method   string   `json:"method"`
		Pattern  string   `json:"pattern"`
		Segments []string `json:"segments"`
		Params   []string `json:"params"`
	}

	routes := make([]RouteInfo, 0, len(r.routes))
	for _, rt := range r.routes {
		params := []string{}
		segments := []string{}

		for _, seg := range rt.segments {
			if seg.name != "" {
				params = append(params, seg.name)
			}

			if seg.literal != "" {
				segments = append(segments, seg.literal)
			} else if seg.wildcard {
				segments = append(segments, "*"+seg.name)
			} else {
				var segDesc string
				if seg.regex != nil {
					segDesc = fmt.Sprintf(":%s(%s)", seg.name, seg.regex.String())
				} else {
					segDesc = ":" + seg.name
				}
				segments = append(segments, segDesc)
			}
		}

		routes = append(routes, RouteInfo{
			Method:   rt.method,
			Pattern:  rt.pattern,
			Segments: segments,
			Params:   params,
		})
	}

	// Sort routes by method and pattern for easier reading
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Method == routes[j].Method {
			return routes[i].Pattern < routes[j].Pattern
		}
		return routes[i].Method < routes[j].Method
	})

	JSON(w, http.StatusOK, routes)
}

// debugHandler muestra información detallada de la petición actual
func (r *MoraRouter) debugHandler(w http.ResponseWriter, req *http.Request, p Params) {
	debug := map[string]interface{}{
		"request": map[string]interface{}{
			"method":     req.Method,
			"path":       req.URL.Path,
			"query":      req.URL.Query(),
			"headers":    req.Header,
			"host":       req.Host,
			"remoteAddr": req.RemoteAddr,
			"params":     p,
		},
		"router": map[string]interface{}{
			"routeCount":       len(r.routes),
			"mountCount":       len(r.mounts),
			"middlewareCount":  len(r.middlewares),
			"registeredMacros": len(MacroRegistry),
		},
	}

	JSON(w, http.StatusOK, debug)
}

// DebugPrint imprime información de depuración si el modo Debug está activado
func DebugPrint(format string, args ...interface{}) {
	if os.Getenv("MORA_DEBUG") == "1" {
		fmt.Printf("[MORA DEBUG] "+format+"\n", args...)
	}
}
