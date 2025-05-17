package router

import (
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
)

// TemplateManager handles loading and rendering of templates
type TemplateManager struct {
	mutex        sync.RWMutex
	templates    map[string]*template.Template
	baseDir      string
	layoutFile   string
	partials     []string
	funcMap      template.FuncMap
	cssInliner   bool
	cssVariables map[string]string
	cache        bool
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(baseDir string) *TemplateManager {
	return &TemplateManager{
		templates:    make(map[string]*template.Template),
		baseDir:      baseDir,
		cssInliner:   true,
		cssVariables: make(map[string]string),
		cache:        true,
		funcMap:      make(template.FuncMap),
	}
}

// WithLayout sets a layout template to be used for all views
func (tm *TemplateManager) WithLayout(layoutFile string) *TemplateManager {
	tm.layoutFile = layoutFile
	return tm
}

// WithPartials sets partial templates to be included in all templates
func (tm *TemplateManager) WithPartials(partials ...string) *TemplateManager {
	tm.partials = partials
	return tm
}

// WithFuncs adds custom template functions
func (tm *TemplateManager) WithFuncs(funcMap template.FuncMap) *TemplateManager {
	for name, fn := range funcMap {
		tm.funcMap[name] = fn
	}
	return tm
}

// DisableCache turns off template caching for development
func (tm *TemplateManager) DisableCache() *TemplateManager {
	tm.cache = false
	return tm
}

// DisableCSSInliner turns off automatic CSS inlining
func (tm *TemplateManager) DisableCSSInliner() *TemplateManager {
	tm.cssInliner = false
	return tm
}

// WithCSS adds a CSS file to be inlined in templates
func (tm *TemplateManager) WithCSS(name, cssFile string) *TemplateManager {
	cssPath := filepath.Join(tm.baseDir, cssFile)
	content, err := os.ReadFile(cssPath)
	if err != nil {
		log.Printf("Error loading CSS file %s: %v", cssPath, err)
		return tm
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	tm.cssVariables[name] = string(content)
	return tm
}

// getOrCreateTemplate loads a template, using cache if enabled
func (tm *TemplateManager) getOrCreateTemplate(name string) (*template.Template, error) {
	if tm.cache {
		tm.mutex.RLock()
		tmpl, exists := tm.templates[name]
		tm.mutex.RUnlock()
		if exists {
			return tmpl, nil
		}
	}

	// Add standard functions
	funcMap := template.FuncMap{
		"json": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeCSS": func(s string) template.CSS {
			return template.CSS(s)
		},
		"safeJS": func(s string) template.JS {
			return template.JS(s)
		},
		"include": func(templateName string, data interface{}) (template.HTML, error) {
			var buf strings.Builder
			err := tm.ExecuteTemplate(&buf, templateName, data)
			return template.HTML(buf.String()), err
		},
	}

	// Add custom functions
	for name, fn := range tm.funcMap {
		funcMap[name] = fn
	}

	// Start with a base template with functions
	tmpl := template.New(name).Funcs(funcMap)

	// Add layout if specified
	if tm.layoutFile != "" {
		layoutPath := filepath.Join(tm.baseDir, tm.layoutFile)
		layoutBytes, err := os.ReadFile(layoutPath)
		if err != nil {
			return nil, fmt.Errorf("error loading layout %s: %v", layoutPath, err)
		}
		tmpl, err = tmpl.Parse(string(layoutBytes))
		if err != nil {
			return nil, fmt.Errorf("error parsing layout %s: %v", layoutPath, err)
		}
	}

	// Add partials
	for _, partial := range tm.partials {
		partialPath := filepath.Join(tm.baseDir, partial)
		partialBytes, err := os.ReadFile(partialPath)
		if err != nil {
			return nil, fmt.Errorf("error loading partial %s: %v", partialPath, err)
		}
		tmpl, err = tmpl.Parse(string(partialBytes))
		if err != nil {
			return nil, fmt.Errorf("error parsing partial %s: %v", partialPath, err)
		}
	}

	// Load the main template
	templatePath := filepath.Join(tm.baseDir, name)
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("error loading template %s: %v", templatePath, err)
	}

	// Parse the main template
	tmpl, err = tmpl.Parse(string(templateBytes))
	if err != nil {
		return nil, fmt.Errorf("error parsing template %s: %v", templatePath, err)
	}

	// Cache the template if caching is enabled
	if tm.cache {
		tm.mutex.Lock()
		tm.templates[name] = tmpl
		tm.mutex.Unlock()
	}

	return tmpl, nil
}

// ExecuteTemplate renders a template to the provided writer
func (tm *TemplateManager) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	tmpl, err := tm.getOrCreateTemplate(name)
	if err != nil {
		return err
	}

	// If we have CSS variables and data is a map, add them
	if tm.cssInliner && len(tm.cssVariables) > 0 {
		var dataMap map[string]interface{}

		// Convert data to map if possible
		if data == nil {
			dataMap = make(map[string]interface{})
			data = dataMap
		} else if m, ok := data.(map[string]interface{}); ok {
			dataMap = m
		} else {
			// Try to convert struct to map
			jsonBytes, err := json.Marshal(data)
			if err == nil {
				dataMap = make(map[string]interface{})
				if err = json.Unmarshal(jsonBytes, &dataMap); err == nil {
					data = dataMap
				}
			}
		}

		// Add CSS variables if we have a map
		if dataMap != nil {
			for cssName, cssContent := range tm.cssVariables {
				dataMap[cssName] = cssContent
			}
		}
	}

	// Execute the template with data
	return tmpl.Execute(w, data)
}

// Render renders a template as HTTP response
func (tm *TemplateManager) Render(w http.ResponseWriter, status int, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tm.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Template rendering error: %v", err)
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
	}
}

// LoadAll preloads all templates from the base directory
func (tm *TemplateManager) LoadAll() error {
	pattern := filepath.Join(tm.baseDir, "*.html")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, match := range matches {
		name := filepath.Base(match)
		_, err := tm.getOrCreateTemplate(name)
		if err != nil {
			return fmt.Errorf("failed to load template %s: %v", name, err)
		}
	}
	return nil
}

// WithTemplateManager adds template manager middleware to the router
func WithTemplateManager(baseDir string, options ...func(*TemplateManager)) Option {
	return func(r *MoraRouter) {
		tm := NewTemplateManager(baseDir)

		// Apply options
		for _, option := range options {
			option(tm)
		}
		// Add the template manager to the context
		r.Use(func(next HandlerFunc) HandlerFunc {
			return func(w http.ResponseWriter, req *http.Request, p Params) {
				ctx := req.Context()
				ctx = context.WithValue(ctx, templateManagerContextKey, tm)
				next(w, req.WithContext(ctx), p)
			}
		})
	}
}

// templateManagerContextKey is the context key for the template manager
var templateManagerContextKey = &struct{}{}

// GetTemplateManager retrieves the template manager from the request context
func GetTemplateManager(req *http.Request) *TemplateManager {
	if ctx := req.Context().Value(templateManagerContextKey); ctx != nil {
		return ctx.(*TemplateManager)
	}
	return nil
}
