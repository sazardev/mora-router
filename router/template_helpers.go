package router

import (
	"log"
	"net/http"
)

// This file contains deprecated legacy helpers.
// All template functionality has been moved to template.go

// LegacyTemplateSetup provides backwards compatibility with old code
func LegacyTemplateSetup(dir string) Option {
	log.Println("Warning: Using deprecated template setup function. Use ConfigureTemplates instead.")
	return ConfigureTemplates(dir)
}

// LegacyRenderTemplate provides backwards compatibility with old code
func LegacyRenderTemplate(w http.ResponseWriter, r *http.Request, name string, data interface{}) error {
	log.Println("Warning: Using deprecated template rendering function. Use RenderTemplateView instead.")
	return RenderTemplateView(w, r, name, data)
}
