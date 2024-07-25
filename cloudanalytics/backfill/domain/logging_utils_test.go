package domain

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDisplayMessageFromKnownError(t *testing.T) {
	for k, e := range DisplayMessageMapping {
		msg := GetDisplayMessageFromError(fmt.Errorf(k))
		assert.Equal(t, e, msg)
	}
}

func TestGetDisplayMessageFromUnknownError(t *testing.T) {
	msg1 := GetDisplayMessageFromError(fmt.Errorf("I am an unknown error"))
	assert.Equal(t, M0007, msg1)
}

func TestGetDisplayMessageFromNil(t *testing.T) {
	msg := GetDisplayMessageFromError(nil)
	assert.Equal(t, "", msg)
}

func TestGetActionFromError(t *testing.T) {
	flowInfo := FlowInfo{
		CustomerID:       "lego",
		BillingAccountID: "AAAAAA-BBBBBB-CCCCC",
		ProjectID:        "hello",
		DatasetID:        "world",
		TableID:          "bar",
	}

	assert.Equal(t, GetActionFromMessage(M0001, &flowInfo), "url:/customers/lego/settings/gcp")
	assert.Equal(t, GetActionFromMessage(M0002, &flowInfo), "url:/customers/lego/settings/gcp")
	assert.Equal(t, GetActionFromMessage(M0003, &flowInfo), "modal")
	assert.Equal(t, GetActionFromMessage(M0004, &flowInfo), "url:https://help.doit.com/google-cloud/import-historical-billing-data#step-2-grant-the-required-permissions")
	assert.Equal(t, GetActionFromMessage(M0005, &flowInfo), "url:https://help.doit.com/google-cloud/import-historical-billing-data#step-2-grant-the-required-permissions")
	assert.Equal(t, GetActionFromMessage(M0006, &flowInfo), "modal")
	assert.Equal(t, GetActionFromMessage(M0007, &flowInfo), "url:/customers/lego/support/new/gcp-backfill-issue?billing-account-id=AAAAAA-BBBBBB-CCCCC&dataset-id=world&project-id=hello&table-id=bar")
	assert.Equal(t, GetActionFromMessage("waaaaat an unknown error", &flowInfo), "modal")
}
