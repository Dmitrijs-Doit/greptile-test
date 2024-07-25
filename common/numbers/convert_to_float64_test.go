package numbers

import (
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"
)

func TestConvertToFloat64(t *testing.T) {
	_, err := ConvertToFloat64(int(100))
	assert.NoError(t, err)

	_, err = ConvertToFloat64(int64(100))
	assert.NoError(t, err)

	_, err = ConvertToFloat64(100.11)
	assert.NoError(t, err)

	_, err = ConvertToFloat64(float32(100.11))
	assert.NoError(t, err)

	_, err = ConvertToFloat64(bigquery.Value(float64(100.11)))
	assert.NoError(t, err)

	_, err = ConvertToFloat64("str")
	assert.Error(t, err)
}
