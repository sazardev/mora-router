package router

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// WithTemplates sets up template rendering for all handlers
func WithTemplates(templatesDir string) Option {
	return func(r *MoraRouter) {
		tm := NewTemplateManager(templatesDir)

		// Automatically load CSS files matching HTML templates
		cssFiles, _ := filepath.Glob(filepath.Join(templatesDir, "*.css"))
		for _, cssFile := range cssFiles {
			baseName := filepath.Base(cssFile)
			varName := strings.TrimSuffix(baseName, filepath.Ext(baseName)) + "CSS"
			tm.WithCSS(varName, baseName)
		}

		// Try to load all templates
		if err := tm.LoadAll(); err != nil {
			log.Printf("Error preloading templates: %v", err)
		}
		// Make the template manager available to all handlers
		r.Use(func(next HandlerFunc) HandlerFunc {
			return func(w http.ResponseWriter, req *http.Request, p Params) {
				ctx := req.Context()
				ctx = context.WithValue(ctx, templateManagerContextKey, tm)
				next(w, req.WithContext(ctx), p)
			}
		})
	}
}

// RenderTemplate is a helper function to render templates from handlers
func RenderTemplate(w http.ResponseWriter, r *http.Request, name string, data interface{}, status ...int) {
	// Default status is 200 OK
	responseStatus := http.StatusOK
	if len(status) > 0 {
		responseStatus = status[0]
	}

	// Try to get template manager from context
	tm := GetTemplateManager(r)
	if tm != nil {
		tm.Render(w, responseStatus, name, data)
		return
	}

	// Fallback to direct render
	render := NewRender()
	render.TemplateDir = "templates" // Default template directory
	render.HTML(w, responseStatus, name, data)
}

// Helper to get any CSS file and include it in response data
func GetCSS(cssPath string) (string, error) {
	content, err := os.ReadFile(cssPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
