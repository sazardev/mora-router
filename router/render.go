package router

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
)

// Responder es una interfaz común para diferentes formatos de respuesta.
type Responder interface {
	Respond(http.ResponseWriter, int, interface{})
}

// Render facilita el renderizado de respuestas en diferentes formatos.
type Render struct {
	// Opciones comunes
	IndentJSON      bool
	HTMLTemplates   *template.Template
	TemplateDir     string
	DefaultCharset  string
	TemplateManager *TemplateManager
}

// NewRender crea un nuevo renderizador con opciones por defecto.
func NewRender() *Render {
	return &Render{
		IndentJSON:     true,
		DefaultCharset: "utf-8",
	}
}

// JSON renderiza una respuesta en formato JSON.
func (r *Render) JSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", fmt.Sprintf("application/json; charset=%s", r.DefaultCharset))
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	if r.IndentJSON {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// XML renderiza una respuesta en formato XML.
func (r *Render) XML(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", fmt.Sprintf("application/xml; charset=%s", r.DefaultCharset))
	w.WriteHeader(status)

	w.Write([]byte(xml.Header))
	encoder := xml.NewEncoder(w)
	if err := encoder.Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HTML renderiza una plantilla HTML.
func (r *Render) HTML(w http.ResponseWriter, status int, name string, data interface{}) {
	// If we have a TemplateManager, use it
	if r.TemplateManager != nil {
		w.Header().Set("Content-Type", fmt.Sprintf("text/html; charset=%s", r.DefaultCharset))
		w.WriteHeader(status)
		if err := r.TemplateManager.Render(w, name, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// If we have access to the router through the context
	if req, ok := data.(interface{ GetRouter() *MoraRouter }); ok {
		if router := req.GetRouter(); router != nil && router.templateManager != nil {
			w.Header().Set("Content-Type", fmt.Sprintf("text/html; charset=%s", r.DefaultCharset))
			w.WriteHeader(status)
			if err := router.templateManager.Render(w, name, data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}

	// Legacy fallback using standard template loading
	w.Header().Set("Content-Type", fmt.Sprintf("text/html; charset=%s", r.DefaultCharset))
	w.WriteHeader(status)

	if r.HTMLTemplates == nil {
		// Cargar plantillas si no se han cargado
		if r.TemplateDir != "" {
			var err error
			r.HTMLTemplates, err = template.ParseGlob(filepath.Join(r.TemplateDir, "*.html"))
			if err != nil {
				http.Error(w, "Error loading templates", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "No templates configured", http.StatusInternalServerError)
			return
		}
	}

	if err := r.HTMLTemplates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Text renderiza una respuesta en texto plano.
func (r *Render) Text(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", fmt.Sprintf("text/plain; charset=%s", r.DefaultCharset))
	w.WriteHeader(status)
	w.Write([]byte(text))
}

// CSV renderiza una tabla de datos como CSV.
func (r *Render) CSV(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "text/csv")
	w.WriteHeader(status)

	csvWriter := csv.NewWriter(w)

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Handle different types - we support slices of structs or maps
	switch v.Kind() {
	case reflect.Slice:
		if v.Len() > 0 {
			firstElem := v.Index(0)
			switch firstElem.Kind() {
			case reflect.Struct:
				// Get field names for header
				t := firstElem.Type()
				header := make([]string, t.NumField())
				for i := 0; i < t.NumField(); i++ {
					header[i] = t.Field(i).Name
				}
				csvWriter.Write(header)

				// Write each row
				for i := 0; i < v.Len(); i++ {
					row := make([]string, t.NumField())
					item := v.Index(i)
					for j := 0; j < t.NumField(); j++ {
						row[j] = fmt.Sprint(item.Field(j).Interface())
					}
					csvWriter.Write(row)
				}
			case reflect.Map:
				// For slice of maps, use keys of first map as header
				firstMap := firstElem.Interface().(map[string]interface{})
				headers := make([]string, 0, len(firstMap))
				for k := range firstMap {
					headers = append(headers, k)
				}
				csvWriter.Write(headers)

				// Write each row
				for i := 0; i < v.Len(); i++ {
					row := make([]string, len(headers))
					mapValue := v.Index(i).Interface().(map[string]interface{})
					for j, header := range headers {
						if val, ok := mapValue[header]; ok {
							row[j] = fmt.Sprint(val)
						}
					}
					csvWriter.Write(row)
				}
			}
		}
	}

	csvWriter.Flush()
}

// YAML renderiza una respuesta en formato YAML.
func (r *Render) YAML(w http.ResponseWriter, status int, v interface{}) {
	// If YAML support is needed, add external dependency
	// or use JSON temporarily
	r.JSON(w, status, v)
}

// Negotiate elige automáticamente el formato de respuesta según la cabecera Accept.
func (r *Render) Negotiate(w http.ResponseWriter, req *http.Request, status int, v interface{}) {
	accept := req.Header.Get("Accept")

	// Implementación básica de negociación de contenido
	switch {
	case strings.Contains(accept, "application/json"):
		r.JSON(w, status, v)
	case strings.Contains(accept, "application/xml"):
		r.XML(w, status, v)
	case strings.Contains(accept, "text/csv"):
		r.CSV(w, status, v)
	case strings.Contains(accept, "text/html"):
		// Si es una plantilla, usar nombre proporcionado en v
		if name, ok := v.(string); ok {
			r.HTML(w, status, name, nil)
		} else {
			// Fallback a JSON
			r.JSON(w, status, v)
		}
	default:
		// Default to JSON
		r.JSON(w, status, v)
	}
}
