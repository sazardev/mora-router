package router

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Params map[string]string

type HandlerFunc func(http.ResponseWriter, *http.Request, Params)

type Middleware func(HandlerFunc) HandlerFunc

type Option func(*MoraRouter)

// MoraRouter es un enrutador personalizable estilo Mora.
type MoraRouter struct {
	routes             []route
	middlewares        []Middleware
	notFound           HandlerFunc
	namedRoutes        map[string]string
	mounts             []mount
	middlewareRegistry map[string]Middleware
	i18n               map[string]map[string]string
}

// Alias para compatibilidad
type Router = MoraRouter

// segment representa un segmento de ruta, estático o dinámico con regex opcional.
type segment struct {
	literal  string         // valor a comparar para segmentos estáticos
	name     string         // nombre de parámetro para segmentos dinámicos
	regex    *regexp.Regexp // patrón para validar el valor dinámico
	wildcard bool           // si es segmento comodín que captura el resto de la ruta
}

type route struct {
	method   string
	pattern  string
	segments []segment
	handler  HandlerFunc
}

// mount representa una ruta montada de http.Handler con prefijo.
type mount struct {
	prefix  string
	handler http.Handler
}

// New crea un nuevo enrutador MoraRouter con opciones.
func NewMoraRouter(opts ...Option) *MoraRouter {
	r := &MoraRouter{
		notFound:           defaultNotFound,
		namedRoutes:        make(map[string]string),
		middlewareRegistry: make(map[string]Middleware),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// New crea un nuevo enrutador MoraRouter con alias para compatibilidad.
func New(opts ...Option) *MoraRouter {
	return NewMoraRouter(opts...)
}

// WithLogging agrega middleware de registro de peticiones.
func WithLogging() Option {
	return func(r *MoraRouter) {
		r.middlewareRegistry["logging"] = loggingMiddleware
		r.middlewares = append(r.middlewares, loggingMiddleware)
	}
}

// WithRecovery agrega middleware para recuperación de panics.
func WithRecovery() Option {
	return func(r *MoraRouter) {
		r.middlewareRegistry["recovery"] = recoveryMiddleware
		r.middlewares = append(r.middlewares, recoveryMiddleware)
	}
}

// WithCORS permite configurar CORS con orígenes permitidos.
func WithCORS(allow string) Option {
	return func(r *MoraRouter) {
		cors := corsMiddleware(allow)
		r.middlewareRegistry["cors"] = cors
		r.middlewares = append(r.middlewares, cors)
	}
}

// UseMiddleware configura global middlewares por nombre en orden específico.
func UseMiddleware(names ...string) Option {
	return func(r *MoraRouter) {
		r.middlewares = nil
		for _, name := range names {
			if mw, ok := r.middlewareRegistry[name]; ok {
				r.middlewares = append(r.middlewares, mw)
			}
		}
	}
}

// WithAPIVersioning aplica versionado automático según cabecera o URL.
func WithAPIVersioning(headerName, defaultVersion string) Option {
	return func(r *MoraRouter) {
		r.middlewares = append([]Middleware{versioningMiddleware(headerName, defaultVersion)}, r.middlewares...)
	}
}

// versioningMiddleware reescribe la URL para añadir el prefijo /v{version} según cabecera.
func versioningMiddleware(headerName, defaultVersion string) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, p Params) {
			ver := req.Header.Get(headerName)
			if ver == "" {
				ver = defaultVersion
			}
			prefix := "/v" + ver
			if !strings.HasPrefix(req.URL.Path, "/v") {
				req.URL.Path = prefix + req.URL.Path
			}
			next(w, req, p)
		}
	}
}

// Use permite agregar middlewares directamente.
func (r *MoraRouter) Use(mw ...Middleware) {
	r.middlewares = append(r.middlewares, mw...)
}

// Group crea un subgrupo de rutas con prefijo.
type RouteGroup struct {
	prefix string
	router *MoraRouter
}

// Group inicia un nuevo grupo enrutado.
func (r *MoraRouter) Group(prefix string) *RouteGroup {
	return &RouteGroup{prefix: prefix, router: r}
}

// Métodos de grupo
func (g *RouteGroup) Get(pattern string, handler HandlerFunc) {
	g.router.Handle("GET", g.prefix+pattern, handler)
}
func (g *RouteGroup) Post(pattern string, handler HandlerFunc) {
	g.router.Handle("POST", g.prefix+pattern, handler)
}
func (g *RouteGroup) Put(pattern string, handler HandlerFunc) {
	g.router.Handle("PUT", g.prefix+pattern, handler)
}
func (g *RouteGroup) Delete(pattern string, handler HandlerFunc) {
	g.router.Handle("DELETE", g.prefix+pattern, handler)
}

// Handle registra una ruta con método HTTP, patrón y manejador.
func (r *MoraRouter) Handle(method, pattern string, handler HandlerFunc) {
	// aplicar middlewares
	final := applyMiddlewares(handler, r.middlewares)
	// parsear segmentos con posibles validadores
	rawSegs := splitPath(pattern)
	segs := make([]segment, len(rawSegs))
	for i, raw := range rawSegs {
		segs[i] = parseSegment(raw)
	}
	r.routes = append(r.routes, route{method, pattern, segs, final})
}

// parseSegment analiza un raw segment y construye un segment con regex si aplica.
func parseSegment(raw string) segment {
	// wildcard *name captura el resto
	if strings.HasPrefix(raw, "*") {
		return segment{name: raw[1:], wildcard: true}
	}
	// sintaxis :name(regex)
	if strings.HasPrefix(raw, ":") {
		// extraer nombre y patrón opcional
		body := raw[1:]
		if idx := strings.Index(body, "("); idx >= 0 && strings.HasSuffix(body, ")") {
			name := body[:idx]
			pattern := body[idx+1 : len(body)-1]
			expr := regexp.MustCompile("^" + pattern + "$")
			return segment{name: name, regex: expr}
		}
		return segment{name: body}
	}
	// sintaxis {name:regex}
	if strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}") {
		inner := raw[1 : len(raw)-1]
		parts := strings.SplitN(inner, ":", 2)
		if len(parts) == 2 {
			expr := regexp.MustCompile("^" + parts[1] + "$")
			return segment{name: parts[0], regex: expr}
		}
	}
	// segmento estático
	return segment{literal: raw}
}

// Get, Post, Put y Delete son atajos para Handle con métodos específicos.
func (r *MoraRouter) Get(pattern string, handler HandlerFunc)  { r.Handle("GET", pattern, handler) }
func (r *MoraRouter) Post(pattern string, handler HandlerFunc) { r.Handle("POST", pattern, handler) }
func (r *MoraRouter) Put(pattern string, handler HandlerFunc)  { r.Handle("PUT", pattern, handler) }
func (r *MoraRouter) Delete(pattern string, handler HandlerFunc) {
	r.Handle("DELETE", pattern, handler)
}

// NotFound permite personalizar el manejador 404.
func (r *MoraRouter) NotFound(handler HandlerFunc) {
	r.notFound = handler
}

// Mount permite montar un http.Handler externo bajo un prefijo.
func (r *MoraRouter) Mount(prefix string, h http.Handler) {
	// normalizar prefijo
	p := "/" + strings.Trim(prefix, "/")
	// delegar con StripPrefix para ajustar la ruta interna
	r.mounts = append(r.mounts, mount{prefix: p, handler: http.StripPrefix(p, h)})
}

// ServeHTTP despacha la petición incluyendo mounts, OPTIONS automáticos y manejo 405.
func (r *MoraRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	// primero, manejar montajes externos
	for _, m := range r.mounts {
		if strings.HasPrefix(path, m.prefix) {
			m.handler.ServeHTTP(w, req)
			return
		}
	}
	// particionar path
	pathSegs := splitPath(path)
	// recolectar métodos permitidos para esta ruta
	var allowed []string
	for _, rt := range r.routes {
		// verificar coincidencia de segmentos ignorando método
		if matchSegments(rt.segments, pathSegs, nil) {
			allowed = append(allowed, rt.method)
		}
	}
	// manejo automático de OPTIONS
	if req.Method == http.MethodOptions {
		if len(allowed) > 0 {
			w.Header().Set("Allow", strings.Join(allowed, ","))
			w.WriteHeader(http.StatusNoContent)
		} else {
			r.notFound(w, req, nil)
		}
		return
	}
	// manejar petición normal buscando método exacto
	for _, rt := range r.routes {
		if req.Method != rt.method {
			continue
		}
		params := make(Params)
		if matchSegments(rt.segments, pathSegs, params) {
			// embed en Context
			req2 := req.WithContext(context.WithValue(req.Context(), paramsKey, params))
			rt.handler(w, req2, params)
			return
		}
	}
	// si coincidió path pero no método, responder 405
	if len(allowed) > 0 {
		w.Header().Set("Allow", strings.Join(allowed, ","))
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	// no encontrado
	r.notFound(w, req, nil)
}

// matchSegments verifica si los segments de ruta concuerdan con los pathSegs.
// Si params no es nil, lo llena con valores dinámicos capturados.
func matchSegments(segs []segment, pathSegs []string, params Params) bool {
	n := len(segs)
	// ajustar wildcard
	if n > 0 && segs[n-1].wildcard {
		if len(pathSegs) < n-1 {
			return false
		}
	} else if len(pathSegs) != n {
		return false
	}
	for i, seg := range segs {
		if seg.wildcard {
			if params != nil {
				params[seg.name] = strings.Join(pathSegs[i:], "/")
			}
			return true
		}
		val := pathSegs[i]
		if seg.name != "" {
			if seg.regex != nil && !seg.regex.MatchString(val) {
				return false
			}
			if params != nil {
				params[seg.name] = val
			}
		} else {
			if seg.literal != val {
				return false
			}
		}
	}
	return true
}

// defaultNotFound maneja rutas no encontradas.
func defaultNotFound(w http.ResponseWriter, r *http.Request, p Params) {
	http.NotFound(w, r)
}

// applyMiddlewares aplica los middlewares en orden.
func applyMiddlewares(h HandlerFunc, mws []Middleware) HandlerFunc {
	wrapped := h
	for i := len(mws) - 1; i >= 0; i-- {
		wrapped = mws[i](wrapped)
	}
	return wrapped
}

// loggingMiddleware registra método y ruta.
func loggingMiddleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		log.Printf("[Mora] %s %s", r.Method, r.URL.Path)
		next(w, r, p)
	}
}

// recoveryMiddleware captura panic y responde 500.
func recoveryMiddleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[Mora][Recovery] panic: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next(w, r, p)
	}
}

// corsMiddleware configura cabeceras CORS.
func corsMiddleware(allow string) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			w.Header().Set("Access-Control-Allow-Origin", allow)
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next(w, r, p)
		}
	}
}

// JSON codifica automáticamente la respuesta en JSON.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// BindJSON decodifica JSON en struct T antes de llamar al handler y valida tags `validate`.
func BindJSON[T any](h func(http.ResponseWriter, *http.Request, Params, T)) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		var obj T
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&obj); err != nil {
			http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
			return
		}
		if err := validate(obj); err != nil {
			http.Error(w, fmt.Sprintf("validation error: %v", err), http.StatusBadRequest)
			return
		}
		h(w, r, p, obj)
	}
}

// BindXML decodifica XML en struct T antes de llamar al handler y valida tags `validate`.
func BindXML[T any](h func(http.ResponseWriter, *http.Request, Params, T)) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		var obj T
		dec := xml.NewDecoder(r.Body)
		if err := dec.Decode(&obj); err != nil {
			http.Error(w, fmt.Sprintf("invalid XML: %v", err), http.StatusBadRequest)
			return
		}
		if err := validate(obj); err != nil {
			http.Error(w, fmt.Sprintf("validation error: %v", err), http.StatusBadRequest)
			return
		}
		h(w, r, p, obj)
	}
}

// validate inspecciona tags `validate` en campos de structs y aplica reglas básicas.
func validate(obj any) error {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}
		parts := strings.Split(tag, ",")
		fval := v.Field(i)
		for _, rule := range parts {
			switch {
			case rule == "required":
				zero := reflect.Zero(fval.Type()).Interface()
				if reflect.DeepEqual(fval.Interface(), zero) {
					return fmt.Errorf("%s is required", field.Name)
				}
			case strings.HasPrefix(rule, "min="):
				min, _ := strconv.Atoi(strings.TrimPrefix(rule, "min="))
				switch fval.Kind() {
				case reflect.String:
					if len(fval.String()) < min {
						return fmt.Errorf("%s length must be >= %d", field.Name, min)
					}
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if fval.Int() < int64(min) {
						return fmt.Errorf("%s must be >= %d", field.Name, min)
					}
				}
			}
		}
	}
	return nil
}

// splitPath divide la ruta en segmentos, eliminando barras inicial y final.
func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return []string{}
	}
	return strings.Split(p, "/")
}

// Name asigna un nombre a una ruta para su inversión de URL.
func (r *MoraRouter) Name(name, pattern string) {
	r.namedRoutes[name] = pattern
}

// URL genera la URL de la ruta nombrada con los parámetros dados.
func (r *MoraRouter) URL(name string, params ...string) (string, error) {
	pattern, ok := r.namedRoutes[name]
	if !ok {
		return "", fmt.Errorf("ruta no encontrada: %s", name)
	}
	segs := splitPath(pattern)
	var result []string
	idx := 0
	for _, seg := range segs {
		if strings.HasPrefix(seg, ":") {
			if idx >= len(params) {
				return "", fmt.Errorf("faltan parámetros para la ruta %s", name)
			}
			result = append(result, params[idx])
			idx++
		} else {
			result = append(result, seg)
		}
	}
	if idx < len(params) {
		return "", fmt.Errorf("demasiados parámetros para la ruta %s", name)
	}
	return "/" + strings.Join(result, "/"), nil
}

// context key for params embedding
type contextKey string

const paramsKey contextKey = "routerParams"

// Param obtiene un parámetro de ruta desde el context.Context de la petición
func Param(r *http.Request, name string) string {
	if p, ok := r.Context().Value(paramsKey).(Params); ok {
		return p[name]
	}
	return ""
}

// WithMetrics registra un endpoint /metrics y un middleware para latencias
func WithMetrics() Option {
	return func(r *MoraRouter) {
		// middleware
		m := metricsMiddleware
		r.middlewareRegistry["metrics"] = m
		r.middlewares = append(r.middlewares, m)
		// endpoint
		r.Get("/metrics", func(w http.ResponseWriter, req *http.Request, p Params) {
			metricsHandler(w, req)
		})
	}
}

var (
	metricsMu sync.Mutex
	latencies []time.Duration
)

func metricsMiddleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		start := time.Now()
		next(w, r, p)
		dur := time.Since(start)
		metricsMu.Lock()
		latencies = append(latencies, dur)
		metricsMu.Unlock()
	}
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	total := time.Duration(0)
	for _, d := range latencies {
		total += d
	}
	avg := time.Duration(0)
	if len(latencies) > 0 {
		avg = total / time.Duration(len(latencies))
	}
	fmt.Fprintf(w, "# HELP http_handler_latency_seconds_average average latency in seconds\n")
	fmt.Fprintf(w, "http_handler_latency_seconds_average %f\n", avg.Seconds())
	fmt.Fprintf(w, "# HELP http_handler_requests_total total handled requests\n")
	fmt.Fprintf(w, "http_handler_requests_total %d\n", len(latencies))
}

// WithCache activa un middleware de caching en memoria por ruta
func WithCache(ttl time.Duration) Option {
	return func(r *MoraRouter) {
		r.Use(cacheMiddleware(ttl))
	}
}

type cacheEntry struct {
	header http.Header
	status int
	body   []byte
	expire time.Time
}

var (
	cacheMu    sync.Mutex
	cacheStore = map[string]cacheEntry{}
)

func cacheMiddleware(ttl time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			key := r.Method + ":" + r.URL.RequestURI()
			cacheMu.Lock()
			e, ok := cacheStore[key]
			cacheMu.Unlock()
			if ok && time.Now().Before(e.expire) {
				for k, vs := range e.header {
					for _, v := range vs {
						w.Header().Add(k, v)
					}
				}
				w.WriteHeader(e.status)
				w.Write(e.body)
				return
			}
			// capture response
			buf := &bytes.Buffer{}
			rw := &responseBuffer{ResponseWriter: w, buf: buf, header: http.Header{}, status: http.StatusOK}
			next(rw, r, p)
			cacheMu.Lock()
			cacheStore[key] = cacheEntry{rw.header, rw.status, buf.Bytes(), time.Now().Add(ttl)}
			cacheMu.Unlock()
		}
	}
}

type responseBuffer struct {
	http.ResponseWriter
	buf    *bytes.Buffer
	header http.Header
	status int
}

func (r *responseBuffer) Header() http.Header { return r.header }
func (r *responseBuffer) Write(b []byte) (int, error) {
	r.buf.Write(b)
	return r.ResponseWriter.Write(b)
}
func (r *responseBuffer) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// WithRateLimit activa un middleware para limitar peticiones por IP
func WithRateLimit(max int, window time.Duration) Option {
	return func(r *MoraRouter) {
		r.Use(rateLimitMiddleware(max, window))
	}
}

type rateInfo struct {
	count     int
	windowEnd time.Time
}

var (
	rateMu  sync.Mutex
	rateMap = map[string]rateInfo{}
)

func rateLimitMiddleware(max int, window time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			ip := strings.Split(r.RemoteAddr, ":")[0]
			rateMu.Lock()
			info := rateMap[ip]
			now := time.Now()
			if now.After(info.windowEnd) {
				info = rateInfo{count: 0, windowEnd: now.Add(window)}
			}
			if info.count >= max {
				rateMu.Unlock()
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			info.count++
			rateMap[ip] = info
			rateMu.Unlock()
			next(w, r, p)
		}
	}
}

// Handy responders

// Error responde con un código y mensaje simple
func Error(w http.ResponseWriter, status int, msg string) {
	http.Error(w, msg, status)
}

// Redirect envía redirección HTTP
func Redirect(w http.ResponseWriter, r *http.Request, urlStr string, code int) {
	http.Redirect(w, r, urlStr, code)
}

// FileDownload fuerza descarga de un archivo
func FileDownload(w http.ResponseWriter, r *http.Request, filePath string) {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(filePath)))
	http.ServeFile(w, r, filePath)
}

// WithHotReload habilita recarga automática de rutas al detectar cambios en el archivo dado.
func WithHotReload(filePath string, interval time.Duration) Option {
	return func(r *MoraRouter) {
		go func() {
			var lastMod time.Time
			for {
				if fi, err := os.Stat(filePath); err == nil {
					if fi.ModTime().After(lastMod) {
						lastMod = fi.ModTime()
						// TODO: invocar lógica de recarga (p.ej. r.reloadRoutes())
					}
				}
				time.Sleep(interval)
			}
		}()
	}
}

// WithI18n configura mapas de traducción de rutas por idioma.
func WithI18n(translations map[string]map[string]string) Option {
	return func(r *MoraRouter) {
		// translations[rutaNombre][lang] = patrón traducido
		r.i18n = translations
	}
}

// WithSwagger registra un endpoint /openapi.json que expone la especificación OpenAPI generada automáticamente.
func WithSwagger() Option {
	return func(r *MoraRouter) {
		r.Get("/openapi.json", func(w http.ResponseWriter, req *http.Request, p Params) {
			JSON(w, http.StatusOK, r.BuildOpenAPISpec())
		})
	}
}

// BuildOpenAPISpec genera un mapa con la especificación OpenAPI 3.0 a partir de las rutas registradas.
func (r *MoraRouter) BuildOpenAPISpec() map[string]interface{} {
	paths := make(map[string]map[string]interface{})
	for _, rt := range r.routes {
		if paths[rt.pattern] == nil {
			paths[rt.pattern] = make(map[string]interface{})
		}
		// parámetros de path
		var params []map[string]interface{}
		for _, seg := range rt.segments {
			if seg.name != "" {
				params = append(params, map[string]interface{}{
					"name":     seg.name,
					"in":       "path",
					"required": true,
					"schema":   map[string]string{"type": "string"},
				})
			}
		}
		paths[rt.pattern][strings.ToLower(rt.method)] = map[string]interface{}{
			"parameters": params,
			"responses":  map[string]interface{}{"200": map[string]string{"description": "OK"}},
		}
	}
	return map[string]interface{}{
		"openapi": "3.0.0",
		"info":    map[string]string{"title": "Mora API", "version": "1.0.0"},
		"paths":   paths,
	}
}

// WithJWT agrega un middleware de autenticación JWT HMAC-SHA256 usando una clave secreta.
func WithJWT(secret string) Option {
	return func(r *MoraRouter) {
		r.Use(jwtMiddleware([]byte(secret)))
	}
}

func jwtMiddleware(secret []byte) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, p Params) {
			auth := req.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(auth, "Bearer ")
			parts := strings.Split(token, ".")
			if len(parts) != 3 {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			header, payload, sig := parts[0], parts[1], parts[2]
			data := header + "." + payload
			mac := hmac.New(sha256.New, secret)
			mac.Write([]byte(data))
			expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
			if !hmac.Equal([]byte(expected), []byte(sig)) {
				http.Error(w, "Invalid signature", http.StatusUnauthorized)
				return
			}
			decoded, err := base64.RawURLEncoding.DecodeString(payload)
			if err != nil {
				http.Error(w, "Invalid payload", http.StatusUnauthorized)
				return
			}
			var claims map[string]any
			if err := json.Unmarshal(decoded, &claims); err != nil {
				http.Error(w, "Invalid claims", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(req.Context(), contextKey("claims"), claims)
			req2 := req.WithContext(ctx)
			next(w, req2, p)
		}
	}
}

// GetClaims extrae los claims JWT del contexto de la petición.
func GetClaims(req *http.Request) map[string]any {
	if v, ok := req.Context().Value(contextKey("claims")).(map[string]any); ok {
		return v
	}
	return nil
}

// RequireRole crea un middleware que verifica que 'roles' en los claims JWT incluya el rol dado.
func RequireRole(role string) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, p Params) {
			claims := GetClaims(req)
			if claims == nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if roles, ok := claims["roles"].([]interface{}); ok {
				for _, r := range roles {
					if r == role {
						next(w, req, p)
						return
					}
				}
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	}
}
