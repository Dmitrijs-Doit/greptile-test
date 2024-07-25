package cloudanalytics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// test equality of aws and gcp fields as needed for UNION in query
func Test_getReportFields(t *testing.T) {
	type args struct {
		isCSP bool
	}

	tests := []struct {
		name string
		args args
		err  error
	}{
		{
			name: "non csp query",
			args: args{
				isCSP: false,
			},
		},
		{
			name: "csp query",
			args: args{
				isCSP: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gcpFields := convertStringToSlice(getGcpReportFields(tt.args.isCSP))
			awsFields := convertStringToSlice(getAwsReportFields(tt.args.isCSP))
			assert.Equal(t, len(gcpFields), len(awsFields))
			awsFieldNames := extractFieldNames(awsFields)
			gcpFieldNames := extractFieldNames(gcpFields)
			assert.Equal(t, gcpFieldNames, awsFieldNames)
		})
	}
}

// converts the string of fields e.g. `a, b, NULL as c, STRUCT(d, e) AS f` to slice string[]{"a", "b", "NULL as c", "STRUCT(d, e) AS f"}
func convertStringToSlice(fieldsString string) []string {
	var (
		field          string
		isComplexField bool
		fieldsList     []string
		parentheses    int
	)

	for i := 0; i < len(fieldsString); i++ {
		char := string(fieldsString[i])
		switch char {
		case "(":
			isComplexField = true
			field += char
			parentheses++

			break
		case ")":
			isComplexField = false
			field += char
			parentheses--

			break
		case ",":
			if isComplexField || parentheses != 0 {
				field += char
				break
			} else {
				fieldsList = append(fieldsList, field)
				field = ""
				parentheses = 0

				break
			}
		case " ":
			if field == "" {
				break
			}

			field += char
		default:
			field += char
		}
	}

	return fieldsList
}

// accepts a slice of fields and removes the string before the alias e.g. STRUCT(d, e) AS f will become f.
func extractFieldNames(fieldsList []string) []string {
	const alias = "AS"
	for i, field := range fieldsList {
		aliasIndex := strings.LastIndex(field, alias)
		if aliasIndex > 0 {
			fieldsList[i] = field[aliasIndex+3:]
		}
	}

	return fieldsList
}
