package errormsg

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ErrorMsg struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ErrorMsg) Error() string {
	return e.Field + ": " + e.Message
}

func mapTagErrorMsg(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "lte":
		return "Should be less or equal than " + fe.Param()
	case "gte":
		return "Should be greater or equal than " + fe.Param()
	case "iso4217":
		return "Currency code is not correct"
	case "min":
		return "Min array length should be " + fe.Param()
	case "max":
		return "Max array length should be " + fe.Param()
	}

	return fe.Tag()
}

func MapTagValidationErrors(err error, ignoreRequiredTag bool) []ErrorMsg {
	var ve validator.ValidationErrors

	if errors.As(err, &ve) {
		out := []ErrorMsg{}

		for _, fe := range ve {
			fullFieldName := fe.StructNamespace()
			shortFieldName := fullFieldName[strings.Index(fullFieldName, ".")+1:]

			if ignoreRequiredTag && fe.Tag() == "required" {
				continue
			}

			out = append(out, ErrorMsg{strings.ToLower(shortFieldName), mapTagErrorMsg(fe)})
		}

		return out
	}

	if err != nil {
		pattern := regexp.MustCompile("json: cannot unmarshal ([-A-z]+) into Go struct field ([-A-z]+).([-A-z0-9.]+) of type ([-A-z0-9.]+)")
		values := pattern.FindStringSubmatch(fmt.Sprint(err))

		if len(values) == 5 {
			return []ErrorMsg{{Field: values[3], Message: "wrong type"}}
		}

		return []ErrorMsg{{Message: "JSON parsing error: " + fmt.Sprint(err)}}
	}

	return nil
}
