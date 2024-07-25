//go:build integration
// +build integration

package dataapi

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	logObj := LogItem{
		Operation: "cmp.test",
		ProjectID: "project",
		Context:   "context",
		Category:  "category",
		UserEmail: "m@t.com",
	}

	err := SendLogToCloudLogging(&logObj)
	fmt.Println(err)
	assert.Equal(t, err, nil)
}
