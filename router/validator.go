package router

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ValidationError representa un error de validación con información detallada.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Rule    string `json:"rule"`
	Value   string `json:"value"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors es una colección de errores de validación.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	messages := make([]string, len(e))
	for i, err := range e {
		messages[i] = err.Error()
	}
	return strings.Join(messages, "; ")
}

// Validator es un validador configurable para structs.
type Validator struct {
	// Custom validators map
	customValidators map[string]func(interface{}) bool
	// Field transformers
	transformers map[string]func(interface{}) interface{}
}

// NewValidator crea un nuevo validador.
func NewValidator() *Validator {
	return &Validator{
		customValidators: make(map[string]func(interface{}) bool),
		transformers:     make(map[string]func(interface{}) interface{}),
	}
}

// RegisterValidator registra un validador personalizado.
func (v *Validator) RegisterValidator(name string, fn func(interface{}) bool) {
	v.customValidators[name] = fn
}

// RegisterTransformer registra un transformador para un campo.
func (v *Validator) RegisterTransformer(field string, fn func(interface{}) interface{}) {
	v.transformers[field] = fn
}

// Validate valida un struct basado en tags `validate`.
func (v *Validator) Validate(obj interface{}) ValidationErrors {
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return ValidationErrors{{
			Field:   "input",
			Message: "validation requires a struct",
			Rule:    "struct",
		}}
	}

	var errors ValidationErrors

	t := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		fieldValue := value.Field(i)
		fieldName := field.Name

		// Apply transformer if exists
		if transformer, ok := v.transformers[fieldName]; ok {
			if fieldValue.CanSet() {
				transformedValue := transformer(fieldValue.Interface())
				if transformedValue != nil {
					newValue := reflect.ValueOf(transformedValue)
					if newValue.Type().AssignableTo(fieldValue.Type()) {
						fieldValue.Set(newValue)
					}
				}
			}
		}

		// Check each validation rule
		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			parts := strings.SplitN(rule, "=", 2)
			ruleName := parts[0]
			ruleValue := ""
			if len(parts) > 1 {
				ruleValue = parts[1]
			}

			var valid bool
			var errMsg string

			// Check built-in rules
			switch ruleName {
			case "required":
				valid = !v.isZero(fieldValue)
				if !valid {
					errMsg = "is required"
				}

			case "email":
				if str, ok := fieldValue.Interface().(string); ok {
					valid = v.isValidEmail(str)
					if !valid {
						errMsg = "must be a valid email address"
					}
				} else {
					valid = false
					errMsg = "must be a string for email validation"
				}

			case "min":
				minValue, err := strconv.Atoi(ruleValue)
				if err != nil {
					valid = false
					errMsg = "invalid min value"
				} else {
					valid, errMsg = v.validateMin(fieldValue, minValue)
				}

			case "max":
				maxValue, err := strconv.Atoi(ruleValue)
				if err != nil {
					valid = false
					errMsg = "invalid max value"
				} else {
					valid, errMsg = v.validateMax(fieldValue, maxValue)
				}

			case "in":
				allowedValues := strings.Split(ruleValue, "|")
				valid = v.validateIn(fieldValue, allowedValues)
				if !valid {
					errMsg = fmt.Sprintf("must be one of: %s", ruleValue)
				}

			case "regex":
				if str, ok := fieldValue.Interface().(string); ok {
					re, err := regexp.Compile(ruleValue)
					if err != nil {
						valid = false
						errMsg = "invalid regex pattern"
					} else {
						valid = re.MatchString(str)
						if !valid {
							errMsg = fmt.Sprintf("must match pattern: %s", ruleValue)
						}
					}
				} else {
					valid = false
					errMsg = "must be a string for regex validation"
				}

			default:
				// Check custom validators
				if customValidator, ok := v.customValidators[ruleName]; ok {
					valid = customValidator(fieldValue.Interface())
					if !valid {
						errMsg = fmt.Sprintf("failed custom validation: %s", ruleName)
					}
				} else {
					// Unknown rule, skip
					continue
				}
			}

			// If validation failed, add error
			if !valid {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: errMsg,
					Rule:    rule,
					Value:   fmt.Sprintf("%v", fieldValue.Interface()),
				})
				break // Stop on first error for this field
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// isZero checks if a value is the zero value for its type.
func (v *Validator) isZero(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return value.IsNil()
	}
	return false
}

// isValidEmail validates an email with a regex pattern.
func (v *Validator) isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(email)
}

// validateMin validates minimum values for different types.
func (v *Validator) validateMin(value reflect.Value, min int) (bool, string) {
	switch value.Kind() {
	case reflect.String:
		if len(value.String()) < min {
			return false, fmt.Sprintf("length must be at least %d", min)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value.Int() < int64(min) {
			return false, fmt.Sprintf("must be at least %d", min)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value.Uint() < uint64(min) {
			return false, fmt.Sprintf("must be at least %d", min)
		}
	case reflect.Float32, reflect.Float64:
		if value.Float() < float64(min) {
			return false, fmt.Sprintf("must be at least %d", min)
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if value.Len() < min {
			return false, fmt.Sprintf("must contain at least %d items", min)
		}
	default:
		return false, "min validation not supported for this type"
	}
	return true, ""
}

// validateMax validates maximum values for different types.
func (v *Validator) validateMax(value reflect.Value, max int) (bool, string) {
	switch value.Kind() {
	case reflect.String:
		if len(value.String()) > max {
			return false, fmt.Sprintf("length must be at most %d", max)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value.Int() > int64(max) {
			return false, fmt.Sprintf("must be at most %d", max)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value.Uint() > uint64(max) {
			return false, fmt.Sprintf("must be at most %d", max)
		}
	case reflect.Float32, reflect.Float64:
		if value.Float() > float64(max) {
			return false, fmt.Sprintf("must be at most %d", max)
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if value.Len() > max {
			return false, fmt.Sprintf("must contain at most %d items", max)
		}
	default:
		return false, "max validation not supported for this type"
	}
	return true, ""
}

// validateIn validates that a value is in a set of allowed values.
func (v *Validator) validateIn(value reflect.Value, allowedValues []string) bool {
	strValue := fmt.Sprintf("%v", value.Interface())
	for _, allowed := range allowedValues {
		if strValue == allowed {
			return true
		}
	}
	return false
}

// DefaultValidator es una instancia global del validador para uso conveniente.
var DefaultValidator = NewValidator()

// ValidateStruct valida un struct con el validador por defecto.
func ValidateStruct(obj interface{}) ValidationErrors {
	return DefaultValidator.Validate(obj)
}
