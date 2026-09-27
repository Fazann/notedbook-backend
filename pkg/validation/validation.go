// Package validation turns request binding errors into per-field translation keys.
package validation

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// FieldError is one invalid field; Field is the JSON path, e.g. "device.os".
type FieldError struct {
	Field  string
	Key    i18n.Key
	Params i18n.Params
}

// Register makes gin's validator report JSON (or form/uri) field names instead of Go struct field names.
// Call once at startup.
func Register() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		registerTagName(v)
	}
}

func registerTagName(v *validator.Validate) {
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		for _, tag := range []string{"json", "form", "uri"} {
			name, _, _ := strings.Cut(f.Tag.Get(tag), ",")
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
		}
		return f.Name
	})
}

// FromError returns the invalid fields in a binding error. ok is false when err is not a field error
// (e.g. malformed JSON or an empty body).
func FromError(err error) (fields []FieldError, ok bool) {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		for _, fe := range verrs {
			fields = append(fields, fromFieldError(fe))
		}
		return fields, true
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) && typeErr.Field != "" {
		return []FieldError{{
			Field:  typeErr.Field,
			Key:    i18n.FieldInvalidType,
			Params: i18n.Params{"field": typeErr.Field, "type": jsonType(typeErr.Type)},
		}}, true
	}

	return nil, false
}

func fromFieldError(fe validator.FieldError) FieldError {
	field := fieldPath(fe.Namespace())
	params := i18n.Params{"field": field}
	key := i18n.FieldInvalid

	switch fe.Tag() {
	case "required", "required_if", "required_unless", "required_with", "required_without":
		key = i18n.FieldRequired
	case "oneof":
		key = i18n.FieldOneOf
		params["values"] = strings.Join(strings.Fields(fe.Param()), ", ")
	case "email":
		key = i18n.FieldInvalidEmail
	case "min", "gte":
		if isLengthKind(fe.Kind()) {
			key = i18n.FieldMinLength
		} else {
			key = i18n.FieldMinValue
		}
		params["min"] = fe.Param()
	case "max", "lte":
		if isLengthKind(fe.Kind()) {
			key = i18n.FieldMaxLength
		} else {
			key = i18n.FieldMaxValue
		}
		params["max"] = fe.Param()
	case "len":
		if isLengthKind(fe.Kind()) {
			key = i18n.FieldExactLength
			params["len"] = fe.Param()
		}
	}

	return FieldError{Field: field, Key: key, Params: params}
}

// fieldPath drops the root struct name: "LoginReq.device.os" → "device.os".
func fieldPath(namespace string) string {
	if _, rest, found := strings.Cut(namespace, "."); found {
		return rest
	}
	return namespace
}

func isLengthKind(k reflect.Kind) bool {
	return k == reflect.String || k == reflect.Slice || k == reflect.Array || k == reflect.Map
}

func jsonType(t reflect.Type) string {
	if t == nil {
		return "value"
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	}
	return "value"
}
