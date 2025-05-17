package router

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// FormFile representa un archivo subido por un formulario.
type FormFile struct {
	Filename string
	Size     int64
	Header   map[string][]string
	Content  []byte
}

// Form encapsula datos de un formulario para su procesamiento.
type Form struct {
	req           *http.Request
	Values        map[string][]string
	Files         map[string][]*FormFile
	MaxFileSize   int64
	AllowedTypes  []string
	UploadDir     string
	Errors        ValidationErrors
	parsedForm    bool
	parsedMulti   bool
	ValidateEmpty bool
}

// NewForm crea un form handler para una petición HTTP.
func NewForm(r *http.Request) *Form {
	return &Form{
		req:           r,
		Values:        make(map[string][]string),
		Files:         make(map[string][]*FormFile),
		MaxFileSize:   10 * 1024 * 1024, // 10MB por defecto
		UploadDir:     os.TempDir(),
		ValidateEmpty: false,
	}
}

// Parse procesa todos los datos del formulario.
func (f *Form) Parse() error {
	if err := f.ParseForm(); err != nil {
		return err
	}
	return f.ParseMultipart()
}

// ParseForm procesa los datos de formulario básicos.
func (f *Form) ParseForm() error {
	if f.parsedForm {
		return nil
	}

	if err := f.req.ParseForm(); err != nil {
		return err
	}

	f.Values = f.req.Form
	f.parsedForm = true
	return nil
}

// ParseMultipart procesa archivos y datos de formulario multipart.
func (f *Form) ParseMultipart() error {
	if f.parsedMulti {
		return nil
	}

	// Si no es multipart, no hay nada que hacer
	contentType := f.req.Header.Get("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") {
		f.parsedMulti = true
		return nil
	}

	// Parsear con maxMemory como límite
	if err := f.req.ParseMultipartForm(f.MaxFileSize); err != nil {
		return err
	}

	// Procesar archivos
	if f.req.MultipartForm != nil && f.req.MultipartForm.File != nil {
		for field, files := range f.req.MultipartForm.File {
			formFiles := make([]*FormFile, 0, len(files))

			for _, fileHeader := range files {
				// Comprobar extensión si hay filtros
				if len(f.AllowedTypes) > 0 {
					ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
					allowed := false
					for _, allowedType := range f.AllowedTypes {
						if ext == allowedType || "."+strings.ToLower(allowedType) == ext {
							allowed = true
							break
						}
					}
					if !allowed {
						return fmt.Errorf("tipo de archivo no permitido: %s", ext)
					}
				}

				// Abrir el archivo
				file, err := fileHeader.Open()
				if err != nil {
					return err
				}
				defer file.Close()

				// Leer el contenido
				content := make([]byte, fileHeader.Size)
				if _, err := file.Read(content); err != nil {
					return err
				}

				// Guardar la información
				formFile := &FormFile{
					Filename: fileHeader.Filename,
					Size:     fileHeader.Size,
					Header:   fileHeader.Header,
					Content:  content,
				}
				formFiles = append(formFiles, formFile)
			}

			f.Files[field] = formFiles
		}
	}

	f.parsedMulti = true
	return nil
}

// SaveFile guarda un archivo subido en el directorio de uploads.
func (f *Form) SaveFile(field, filename string) (string, error) {
	files, ok := f.Files[field]
	if !ok || len(files) == 0 {
		return "", fmt.Errorf("no file uploaded with field: %s", field)
	}

	file := files[0] // Tomar el primer archivo

	// Generar ruta
	if filename == "" {
		filename = file.Filename
	}
	fullPath := filepath.Join(f.UploadDir, filename)

	// Guardar el archivo
	if err := os.WriteFile(fullPath, file.Content, 0644); err != nil {
		return "", err
	}

	return fullPath, nil
}

// SaveFiles guarda todos los archivos subidos en el directorio de uploads.
func (f *Form) SaveFiles(field string) ([]string, error) {
	files, ok := f.Files[field]
	if !ok || len(files) == 0 {
		return nil, fmt.Errorf("no files uploaded with field: %s", field)
	}

	paths := make([]string, 0, len(files))

	for _, file := range files {
		fullPath := filepath.Join(f.UploadDir, file.Filename)

		if err := os.WriteFile(fullPath, file.Content, 0644); err != nil {
			// Limpiar archivos ya guardados en caso de error
			for _, path := range paths {
				os.Remove(path)
			}
			return nil, err
		}

		paths = append(paths, fullPath)
	}

	return paths, nil
}

// Bind mapea los valores del formulario a un struct y lo valida.
func (f *Form) Bind(dst interface{}) error {
	if err := f.Parse(); err != nil {
		return err
	}

	// Mapear valores a struct
	if err := f.mapValues(dst); err != nil {
		return err
	}

	// Validar el struct resultante
	if errs := ValidateStruct(dst); len(errs) > 0 {
		f.Errors = errs
		return errs
	}

	return nil
}

// mapValues mapea los valores del formulario a campos del struct.
func (f *Form) mapValues(dst interface{}) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer to struct")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a pointer to struct")
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)

		// Obtener nombre del campo desde tag form o usar nombre del campo
		fieldName := field.Tag.Get("form")
		if fieldName == "" {
			fieldName = field.Name
		}
		if fieldName == "-" {
			continue // Campo ignorado
		}

		// Si no existe el valor y no validamos vacíos, continuar
		if _, exists := f.Values[fieldName]; !exists && !f.ValidateEmpty {
			continue
		}

		if err := f.setFieldValue(v.Field(i), field, fieldName); err != nil {
			return err
		}
	}

	return nil
}

// setFieldValue establece el valor del campo según su tipo.
func (f *Form) setFieldValue(field reflect.Value, structField reflect.StructField, name string) error {
	if !field.CanSet() {
		return nil
	}

	// Comprobar si es un campo de archivo
	if structField.Tag.Get("form") == "file" {
		return f.setFileField(field, name)
	}

	// Comprobar si tiene valores
	values, exists := f.Values[name]
	if !exists || len(values) == 0 {
		return nil
	}

	// Setter según tipo de campo
	switch field.Kind() {
	case reflect.String:
		field.SetString(values[0])

	case reflect.Bool:
		val := values[0]
		field.SetBool(val == "true" || val == "1" || val == "on" || val == "yes")

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int value for field %s: %v", name, err)
		}
		field.SetInt(val)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(values[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid uint value for field %s: %v", name, err)
		}
		field.SetUint(val)

	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(values[0], 64)
		if err != nil {
			return fmt.Errorf("invalid float value for field %s: %v", name, err)
		}
		field.SetFloat(val)

	case reflect.Slice:
		// Para slices, usar todos los valores
		if field.Type().Elem().Kind() == reflect.String {
			slice := reflect.MakeSlice(field.Type(), len(values), len(values))
			for i, val := range values {
				slice.Index(i).SetString(val)
			}
			field.Set(slice)
		}

	case reflect.Struct:
		// Tipos especiales como time.Time
		if field.Type() == reflect.TypeOf(time.Time{}) {
			// Intentar varios formatos de fecha/hora
			formats := []string{
				time.RFC3339,
				"2006-01-02T15:04:05",
				"2006-01-02 15:04:05",
				"2006-01-02",
			}

			for _, format := range formats {
				if t, err := time.Parse(format, values[0]); err == nil {
					field.Set(reflect.ValueOf(t))
					break
				}
			}
		}
	}

	return nil
}

// setFileField maneja los campos de tipo archivo.
func (f *Form) setFileField(field reflect.Value, name string) error {
	files, exists := f.Files[name]
	if !exists || len(files) == 0 {
		return nil
	}

	// Según el tipo de campo
	switch {
	case field.Type() == reflect.TypeOf(FormFile{}):
		// Para un único FormFile
		if len(files) > 0 {
			field.Set(reflect.ValueOf(*files[0]))
		}
	case field.Type() == reflect.TypeOf(&FormFile{}):
		// Para un puntero a FormFile
		if len(files) > 0 {
			field.Set(reflect.ValueOf(files[0]))
		}
	case field.Type() == reflect.TypeOf([]*FormFile{}):
		// Para un slice de punteros a FormFile
		field.Set(reflect.ValueOf(files))
	}

	return nil
}

// HasErrors indica si hubo errores de validación.
func (f *Form) HasErrors() bool {
	return len(f.Errors) > 0
}

// GetErrors devuelve todos los errores de validación.
func (f *Form) GetErrors() ValidationErrors {
	return f.Errors
}

// GetError devuelve el primer error para un campo específico.
func (f *Form) GetError(field string) string {
	for _, err := range f.Errors {
		if err.Field == field {
			return err.Message
		}
	}
	return ""
}

// BindForm procesa un formulario en el handler.
func BindForm[T any](h func(http.ResponseWriter, *http.Request, Params, *Form, T)) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		var obj T
		form := NewForm(r)
		if err := form.Bind(&obj); err != nil {
			// Si son errores de validación, los pasamos al handler
			if verr, ok := err.(ValidationErrors); ok {
				form.Errors = verr
				h(w, r, p, form, obj)
				return
			}
			// Otro tipo de error (parseo, etc.)
			Error(w, http.StatusBadRequest, fmt.Sprintf("error processing form: %v", err))
			return
		}

		h(w, r, p, form, obj)
	}
}
