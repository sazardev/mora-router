package router

import (
	"bytes"
	"net/http"
	"regexp"
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
	templateManager    *TemplateManager
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

type cacheEntry struct {
	header http.Header
	status int
	body   []byte
	expire time.Time
}

type rateInfo struct {
	count     int
	windowEnd time.Time
}

type responseBuffer struct {
	http.ResponseWriter
	buf    *bytes.Buffer
	header http.Header
	status int
}

// Group crea un subgrupo de rutas con prefijo.
type RouteGroup struct {
	prefix string
	router *MoraRouter
}

// context key for params embedding
type contextKey string

const paramsKey contextKey = "routerParams"

// ResourceController define los métodos que un controlador de recursos puede implementar.
type ResourceController interface {
	Index(http.ResponseWriter, *http.Request, Params)
	Show(http.ResponseWriter, *http.Request, Params)
	Create(http.ResponseWriter, *http.Request, Params)
	Update(http.ResponseWriter, *http.Request, Params)
	Delete(http.ResponseWriter, *http.Request, Params)
}

// DefaultController es una implementación vacía de ResourceController para embeber y extender.
type DefaultController struct{}

// Macro representa un patrón reutilizable de rutas
type Macro struct {
	name        string
	pattern     string
	methods     []string
	middlewares []Middleware
}
