package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TemplateManager handles loading and rendering of templates
type TemplateManager struct {
	mutex        sync.RWMutex
	templates    map[string]*template.Template
	directory    string
	layout       string
	partials     []string
	funcMap      template.FuncMap
	cssMap       map[string]string
	jsMap        map[string]string
	errorHandler func(error)
	disableCache bool
	development  bool
}

// NewTemplateManager creates a new template manager for the given directory
func NewTemplateManager(directory string) *TemplateManager {
	return &TemplateManager{
		templates: make(map[string]*template.Template),
		directory: directory,
		cssMap:    make(map[string]string),
		jsMap:     make(map[string]string),
		funcMap:   make(template.FuncMap),
		errorHandler: func(err error) {
			log.Printf("[TemplateManager] Error: %v", err)
		},
	}
}

// WithLayout sets a common layout template for all views
func (tm *TemplateManager) WithLayout(layout string) *TemplateManager {
	tm.layout = layout
	tm.Reload()
	return tm
}

// WithPartials adds partial templates to be included in all templates
func (tm *TemplateManager) WithPartials(partials ...string) *TemplateManager {
	tm.partials = partials
	tm.Reload()
	return tm
}

// WithFuncs adds custom functions to the templates
func (tm *TemplateManager) WithFuncs(funcMap template.FuncMap) *TemplateManager {
	// Append to existing function map
	if tm.funcMap == nil {
		tm.funcMap = make(template.FuncMap)
	}
	for name, fn := range funcMap {
		tm.funcMap[name] = fn
	}
	tm.Reload()
	return tm
}

// WithCSS adds a CSS file to be available as a function in templates
func (tm *TemplateManager) WithCSS(name, path string) *TemplateManager {
	tm.cssMap[name] = path
	return tm
}

// WithJS adds a JavaScript file to be available as a function in templates
func (tm *TemplateManager) WithJS(name, path string) *TemplateManager {
	tm.jsMap[name] = path
	return tm
}

// WithErrorHandler sets a custom error handler
func (tm *TemplateManager) WithErrorHandler(handler func(error)) *TemplateManager {
	tm.errorHandler = handler
	return tm
}

// DisableCache prevents templates from being cached
func (tm *TemplateManager) DisableCache() *TemplateManager {
	tm.disableCache = true
	return tm
}

// Development enables development mode with better error messages
func (tm *TemplateManager) Development() *TemplateManager {
	tm.development = true
	return tm
}

// Reload forces a reload of all templates
func (tm *TemplateManager) Reload() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Clear existing templates
	tm.templates = make(map[string]*template.Template)

	// Create base function map with asset helpers
	funcMap := tm.createFuncMap()

	// Find all template files
	err := filepath.Walk(tm.directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-HTML files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".html") {
			return nil
		}

		// Skip layout and partials
		if tm.layout != "" && strings.HasSuffix(path, tm.layout) {
			return nil
		}
		for _, partial := range tm.partials {
			if strings.HasSuffix(path, partial) {
				return nil
			}
		}

		// Get relative path as the template name
		relPath, err := filepath.Rel(tm.directory, path)
		if err != nil {
			return err
		}

		// Create template with functions
		var tmpl *template.Template

		// Start with base template
		tmpl = template.New(filepath.Base(path)).Funcs(funcMap)

		// Add layout if specified
		if tm.layout != "" {
			layoutPath := filepath.Join(tm.directory, tm.layout)
			layoutContent, err := os.ReadFile(layoutPath)
			if err != nil {
				return fmt.Errorf("error reading layout %s: %w", tm.layout, err)
			}
			tmpl, err = tmpl.Parse(string(layoutContent))
			if err != nil {
				return fmt.Errorf("error parsing layout %s: %w", tm.layout, err)
			}
		}

		// Add partials
		for _, partial := range tm.partials {
			partialPath := filepath.Join(tm.directory, partial)
			partialContent, err := os.ReadFile(partialPath)
			if err != nil {
				return fmt.Errorf("error reading partial %s: %w", partial, err)
			}
			tmpl, err = tmpl.Parse(string(partialContent))
			if err != nil {
				return fmt.Errorf("error parsing partial %s: %w", partial, err)
			}
		}

		// Parse the template file itself
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading template %s: %w", relPath, err)
		}

		tmpl, err = tmpl.Parse(string(content))
		if err != nil {
			return fmt.Errorf("error parsing template %s: %w", relPath, err)
		}

		// Store the template
		tm.templates[relPath] = tmpl
		return nil
	})

	if err != nil {
		tm.errorHandler(fmt.Errorf("error loading templates: %w", err))
	}
}

// createFuncMap builds the function map for templates
func (tm *TemplateManager) createFuncMap() template.FuncMap {
	funcMap := template.FuncMap{
		// Basic helpers
		"now": time.Now,
		"formatDate": func(t time.Time, layout string) string {
			return t.Format(layout)
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
		"json": func(v interface{}) string {
			b, err := json.Marshal(v)
			if err != nil {
				return err.Error()
			}
			return string(b)
		},
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"title":     strings.ToTitle,
	}

	// Add user-defined functions
	for name, fn := range tm.funcMap {
		funcMap[name] = fn
	}

	// Add CSS helpers
	for name, path := range tm.cssMap {
		cssPath := path
		funcMap[name] = func() template.HTML {
			content, err := os.ReadFile(filepath.Join(tm.directory, cssPath))
			if err != nil {
				tm.errorHandler(fmt.Errorf("error reading CSS %s: %w", cssPath, err))
				return template.HTML(fmt.Sprintf("<!-- Error loading CSS: %s -->", cssPath))
			}
			return template.HTML(fmt.Sprintf("<style>\n%s\n</style>", content))
		}
	}

	// Add JS helpers
	for name, path := range tm.jsMap {
		jsPath := path
		funcMap[name] = func() template.HTML {
			content, err := os.ReadFile(filepath.Join(tm.directory, jsPath))
			if err != nil {
				tm.errorHandler(fmt.Errorf("error reading JS %s: %w", jsPath, err))
				return template.HTML(fmt.Sprintf("<!-- Error loading JS: %s -->", jsPath))
			}
			return template.HTML(fmt.Sprintf("<script>\n%s\n</script>", content))
		}
	}

	return funcMap
}

// Render renders a template with the given data
func (tm *TemplateManager) Render(w io.Writer, name string, data interface{}) error {
	// Reload templates in development mode or if cache is disabled
	if tm.disableCache || tm.development {
		tm.Reload()
	}

	// Get the template
	tm.mutex.RLock()
	tmpl, ok := tm.templates[name]
	tm.mutex.RUnlock()

	if !ok {
		return fmt.Errorf("template %s not found", name)
	}

	// Execute the template in a buffer first for error handling
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		if tm.development {
			// In development, show the error in the response
			fmt.Fprintf(w, "<h1>Template Error</h1><pre>%s</pre>", err)
		}
		return err
	}

	// Write to the actual writer
	_, err := buf.WriteTo(w)
	return err
}

// Template returns a template by name
func (tm *TemplateManager) Template(name string) (*template.Template, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	tmpl, ok := tm.templates[name]
	if !ok {
		return nil, fmt.Errorf("template %s not found", name)
	}
	return tmpl, nil
}

// GetTemplateManager retrieves the template manager from the router
func GetTemplateManager(r *MoraRouter) *TemplateManager {
	return r.templateManager
}

// Helper functions for the router integration

// ConfigureTemplates configures the template system for the router
func ConfigureTemplates(directory string) Option {
	return func(r *MoraRouter) {
		r.templateManager = NewTemplateManager(directory)
		r.templateManager.Reload()
	}
}

// RenderTemplateView renders a template through the router's template manager
func RenderTemplateView(w http.ResponseWriter, r *http.Request, name string, data interface{}) error {
	ctx := r.Context()
	tm, ok := ctx.Value(contextKey("templateManager")).(*TemplateManager)

	if !ok {
		// Try to get from the global router
		globalRouter, ok := ctx.Value(contextKey("router")).(*MoraRouter)
		if !ok || globalRouter.templateManager == nil {
			return fmt.Errorf("no template manager found")
		}
		tm = globalRouter.templateManager
	}

	// Set content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Add request-specific template functions
	funcMap := template.FuncMap{
		"param": func(name string) string {
			return Param(r, name)
		},
		"query": func(name string) string {
			return r.URL.Query().Get(name)
		},
		"route": func(name string, params ...string) (string, error) {
			router, ok := ctx.Value(contextKey("router")).(*MoraRouter)
			if !ok {
				return "", fmt.Errorf("router not available in context")
			}
			return router.URL(name, params...)
		},
	}

	// Clone the template with request-specific functions
	// Create a new instance instead of copying to avoid mutex issues
	newTM := NewTemplateManager(tm.directory)
	newTM.templates = tm.templates
	newTM.layout = tm.layout
	newTM.partials = tm.partials
	newTM.cssMap = tm.cssMap
	newTM.jsMap = tm.jsMap
	newTM.errorHandler = tm.errorHandler
	newTM.disableCache = tm.disableCache
	newTM.development = tm.development

	// Add the request-specific functions
	for name, fn := range tm.funcMap {
		newTM.funcMap[name] = fn
	}
	newTM.WithFuncs(funcMap)

	return newTM.Render(w, name, data)
}

// ConfigureStaticFiles configures static file serving for the router
func ConfigureStaticFiles(prefix, dir string) Option {
	return func(r *MoraRouter) {
		r.Static(prefix, dir)
	}
}

// Middlewares for template rendering

// WithView returns a handler that renders a template
func WithView(name string, dataFn func(*http.Request) (interface{}, error)) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		// Get data for the view
		var data interface{}
		var err error

		if dataFn != nil {
			data, err = dataFn(r)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error preparing view data: %v", err), http.StatusInternalServerError)
				return
			}
		}

		// Render the template
		if err := RenderTemplateView(w, r, name, data); err != nil {
			http.Error(w, fmt.Sprintf("Error rendering template: %v", err), http.StatusInternalServerError)
		}
	}
}

// TemplateMiddleware adds the template manager to the request context
func TemplateMiddleware(tm *TemplateManager) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			ctx := context.WithValue(r.Context(), contextKey("templateManager"), tm)
			next(w, r.WithContext(ctx), p)
		}
	}
}

// WithTemplates is a convenience function for ConfigureTemplates
func WithTemplates(directory string) Option {
	return ConfigureTemplates(directory)
}

// RenderTemplate is a convenience function for RenderTemplateView
func RenderTemplate(w http.ResponseWriter, r *http.Request, name string, data interface{}) error {
	return RenderTemplateView(w, r, name, data)
}
