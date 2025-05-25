package router

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"time"
)

// FormFile representa un archivo subido por un formulario.
type FormFile struct {
	Filename string
	Size     int64
	Header   map[string][]string
	Content  []byte
}

// Form encapsula los datos de un formulario y sus posibles errores.
type Form struct {
	Values    map[string][]string
	Files     map[string][]*FormFile
	Errors    map[string]string
	validated bool
}

// NewForm crea un nuevo Form desde una petición HTTP.
func NewForm(r *http.Request, maxMemory int64) (*Form, error) {
	if maxMemory <= 0 {
		maxMemory = 32 << 20 // 32MB por defecto
	}

	// Parsear formulario y archivos
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		// Si no es multipart, intentar como form normal
		if err != http.ErrNotMultipart {
			// Intentar ParseForm para formularios normales
			if err := r.ParseForm(); err != nil {
				return nil, fmt.Errorf("error parsing form: %w", err)
			}
		}
	}

	form := &Form{
		Values:    make(map[string][]string),
		Files:     make(map[string][]*FormFile),
		Errors:    make(map[string]string),
		validated: false,
	}

	// Copiar valores del formulario
	if r.PostForm != nil {
		for k, v := range r.PostForm {
			form.Values[k] = append(form.Values[k], v...)
		}
	}
	if r.Form != nil {
		for k, v := range r.Form {
			if _, exists := form.Values[k]; !exists {
				form.Values[k] = append(form.Values[k], v...)
			}
		}
	}

	// Procesar archivos si es multipart
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		for field, fileHeaders := range r.MultipartForm.File {
			for _, header := range fileHeaders {
				file, err := header.Open()
				if err != nil {
					return nil, fmt.Errorf("error opening uploaded file: %w", err)
				}

				content, err := io.ReadAll(file)
				if err != nil {
					file.Close()
					return nil, fmt.Errorf("error reading uploaded file: %w", err)
				}
				file.Close()

				formFile := &FormFile{
					Filename: header.Filename,
					Size:     header.Size,
					Header:   header.Header,
					Content:  content,
				}

				form.Files[field] = append(form.Files[field], formFile)
			}
		}
	}

	return form, nil
}

// Get devuelve el primer valor para un campo del formulario.
func (f *Form) Get(key string) string {
	if vals, ok := f.Values[key]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// GetFile devuelve el primer archivo para un campo del formulario.
func (f *Form) GetFile(key string) *FormFile {
	if files, ok := f.Files[key]; ok && len(files) > 0 {
		return files[0]
	}
	return nil
}

// GetAll devuelve todos los valores para un campo del formulario.
func (f *Form) GetAll(key string) []string {
	return f.Values[key]
}

// GetAllFiles devuelve todos los archivos para un campo del formulario.
func (f *Form) GetAllFiles(key string) []*FormFile {
	return f.Files[key]
}

// Required valida que un campo exista y no esté vacío.
func (f *Form) Required(fields ...string) *Form {
	for _, field := range fields {
		if value := f.Get(field); value == "" {
			f.Errors[field] = "This field is required"
		}
	}
	return f
}

// MaxLength valida que un campo no exceda un largo máximo.
func (f *Form) MaxLength(field string, d int) *Form {
	if value := f.Get(field); value != "" && len(value) > d {
		f.Errors[field] = fmt.Sprintf("This field cannot be longer than %d characters", d)
	}
	return f
}

// MinLength valida que un campo tenga un largo mínimo.
func (f *Form) MinLength(field string, d int) *Form {
	if value := f.Get(field); value != "" && len(value) < d {
		f.Errors[field] = fmt.Sprintf("This field must be at least %d characters long", d)
	}
	return f
}

// IsEmail valida que un campo contenga un email válido.
func (f *Form) IsEmail(field string) *Form {
	value := f.Get(field)
	if value == "" {
		return f
	}

	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !re.MatchString(value) {
		f.Errors[field] = "Invalid email address"
	}
	return f
}

// IsInt valida que un campo contenga un número entero.
func (f *Form) IsInt(field string) *Form {
	value := f.Get(field)
	if value == "" {
		return f
	}

	_, err := strconv.Atoi(value)
	if err != nil {
		f.Errors[field] = "This field must be an integer"
	}
	return f
}

// IsFloat valida que un campo contenga un número decimal.
func (f *Form) IsFloat(field string) *Form {
	value := f.Get(field)
	if value == "" {
		return f
	}

	_, err := strconv.ParseFloat(value, 64)
	if err != nil {
		f.Errors[field] = "This field must be a number"
	}
	return f
}

// CustomValidation aplica una validación personalizada.
func (f *Form) CustomValidation(field string, fn func(string) bool, message string) *Form {
	value := f.Get(field)
	if value == "" {
		return f
	}

	if !fn(value) {
		f.Errors[field] = message
	}
	return f
}

// Valid comprueba si el formulario no tiene errores.
func (f *Form) Valid() bool {
	f.validated = true
	return len(f.Errors) == 0
}

// HasErrors devuelve true si el formulario tiene errores.
func (f *Form) HasErrors() bool {
	return len(f.Errors) > 0
}

// GetErrors devuelve todos los errores.
func (f *Form) GetErrors() map[string]string {
	return f.Errors
}

// AddError agrega un error manualmente.
func (f *Form) AddError(field, message string) *Form {
	f.Errors[field] = message
	return f
}

// SaveFile guarda un archivo subido en una ubicación específica.
func (f *Form) SaveFile(fieldName, targetDir string) (string, error) {
	file := f.GetFile(fieldName)
	if file == nil {
		return "", fmt.Errorf("no file uploaded for field %s", fieldName)
	}

	if targetDir == "" {
		targetDir = os.TempDir()
	}

	// Crear directorio si no existe
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Generar nombre de archivo único si es necesario
	fileName := file.Filename
	if fileName == "" {
		fileName = fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), strconv.Itoa(int(time.Now().Unix())))
	}

	// Crear ruta completa
	filePath := filepath.Join(targetDir, fileName)

	// Escribir archivo
	if err := os.WriteFile(filePath, file.Content, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filePath, nil
}

// Bind completa un struct con datos del formulario usando reflection.
func (f *Form) Bind(obj interface{}) error {
	// Validate forms first
	if !f.validated {
		if !f.Valid() {
			return fmt.Errorf("form validation failed: %v", f.Errors)
		}
	}

	val := reflect.ValueOf(obj)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("bind requires a non-nil pointer")
	}

	// Desreferencia el puntero
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("bind requires a struct pointer")
	}

	typ := val.Type()

	// Recorre los campos del struct
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}

		typeField := typ.Field(i)
		formKey := typeField.Tag.Get("form")
		if formKey == "" {
			formKey = typeField.Name
		}

		// Si el campo es un archivo
		if typeField.Type == reflect.TypeOf(&FormFile{}) && f.GetFile(formKey) != nil {
			file := f.GetFile(formKey)
			field.Set(reflect.ValueOf(file))
			continue
		}

		// Para valores normales
		formVal := f.Get(formKey)
		if formVal == "" {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			field.SetString(formVal)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal, err := strconv.ParseInt(formVal, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid integer value for field %s: %w", formKey, err)
			}
			field.SetInt(intVal)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			uintVal, err := strconv.ParseUint(formVal, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid unsigned integer value for field %s: %w", formKey, err)
			}
			field.SetUint(uintVal)
		case reflect.Float32, reflect.Float64:
			floatVal, err := strconv.ParseFloat(formVal, 64)
			if err != nil {
				return fmt.Errorf("invalid float value for field %s: %w", formKey, err)
			}
			field.SetFloat(floatVal)
		case reflect.Bool:
			boolVal := false
			if formVal == "on" || formVal == "true" || formVal == "1" || formVal == "yes" {
				boolVal = true
			}
			field.SetBool(boolVal)
		}
	}

	return nil
}

// BindForm procesa un formulario, lo valida y enlaza a un struct.
func BindForm[T any](h func(http.ResponseWriter, *http.Request, Params, *Form, T)) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		var obj T

		// Crear y procesar formulario
		form, err := NewForm(r, 32<<20) // 32MB limit
		if err != nil {
			http.Error(w, fmt.Sprintf("error processing form: %v", err), http.StatusBadRequest)
			return
		}

		// Enlazar datos al struct
		if err := form.Bind(&obj); err != nil {
			form.AddError("_form", err.Error())
		}

		// Validar struct usando tags validate
		if errs := ValidateStruct(obj); len(errs) > 0 {
			for _, e := range errs {
				form.AddError(e.Field, e.Message)
			}
		}

		// Llamar al handler con el formulario y el objeto enlazado
		h(w, r, p, form, obj)
	}
}
