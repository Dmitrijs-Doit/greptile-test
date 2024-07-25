package errormsg

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapTagValidationCustomErrors(t *testing.T) {
	tagErrors := []struct {
		tagError        string
		expectedMessage []ErrorMsg
	}{
		{
			tagError: "json: cannot unmarshal string into Go struct field AlertConfigAPI.config.value of type float64",
			expectedMessage: []ErrorMsg{
				{
					Field:   "config.value",
					Message: "wrong type",
				},
			},
		},
		{
			tagError: "json: cannot unmarshal number into Go struct field AlertConfigAPI.config.scope of type []string",
			expectedMessage: []ErrorMsg{{
				Field:   "config.scope",
				Message: "wrong type",
			}},
		},
	}

	for _, test := range tagErrors {
		got := MapTagValidationErrors(errors.New(test.tagError), false)
		assert.Equal(t, test.expectedMessage, got)
	}
}
