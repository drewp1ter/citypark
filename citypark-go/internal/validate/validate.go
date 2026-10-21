// Package validate checks request structs and returns errors in the project's
// API error format.
package validate

import (
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/drewp1ter/citypark-go/internal/httpx"
)

const topMessage = "Ошибка валидации, пожалуйста проверьте введенные вами данные"

// v is safe for concurrent use and caches struct metadata, so one instance is shared.
var v = newValidator()

func newValidator() *validator.Validate {
	val := validator.New()
	val.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			return f.Name
		}
		return name
	})
	err := val.RegisterValidation("ruphone", func(fl validator.FieldLevel) bool {
		_, ok := NormalizePhone(fl.Field().String())
		return ok
	})
	if err != nil {
		panic("validate: register ruphone: " + err.Error())
	}
	return val
}

// Struct validates s using its `validate` struct tags.
// It returns nil, or *httpx.Error with status 400 and one entry per invalid field.
func Struct(s any) error {
	err := v.Struct(s)
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return err
	}
	fields := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		fields[fieldKey(fe)] = message(fe)
	}
	e := httpx.NewError(http.StatusBadRequest, topMessage)
	e.Fields = fields
	return e
}

// fieldKey drops the root struct name from the namespace: "req.items[0].amount" -> "items[0].amount".
func fieldKey(fe validator.FieldError) string {
	_, key, _ := strings.Cut(fe.Namespace(), ".")
	return key
}

func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "поле не должно быть пустым"
	case "email":
		return "email должен быть электронной почтой"
	case "ruphone":
		return "телефон должен быть действительным номером телефона"
	case "min", "gte":
		return "значение должно быть не меньше " + fe.Param()
	case "max":
		return "значение должно быть не больше " + fe.Param()
	case "gt":
		return "значение должно быть больше " + fe.Param()
	case "oneof":
		return "недопустимое значение"
	default:
		return "некорректное значение"
	}
}
