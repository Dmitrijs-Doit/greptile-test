package bqlens

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetProjectIDFromSA(t *testing.T) {
	const (
		validServiceAccount   = "doitintl-cmp-sa@doitintl-svc-accounts.iam.gserviceaccount.com"
		invalidServiceAccount = "doitintl-svc-accounts.iam.gserviceaccount.com"
		emptyServiceAccount   = ""
	)

	result, err := extractProjectIDFromSA(validServiceAccount)
	assert.Equal(t, result, "doitintl-svc-accounts")
	assert.NoError(t, err)

	result, err = extractProjectIDFromSA(invalidServiceAccount)
	assert.Error(t, err)
	assert.Equal(t, result, "")

	result, err = extractProjectIDFromSA(emptyServiceAccount)
	assert.Error(t, err)
	assert.Equal(t, result, "")
}
