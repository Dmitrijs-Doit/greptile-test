package service

import (
	"github.com/google/uuid"
)

const (
	msRequestID       string = "MS-RequestId"
	msCorrelationID   string = "MS-CorrelationId"
	applicationJSON          = "application/json"
	contentType       string = "Content-Type"
	StatusSuspended   string = "suspended"
	subscriptionsPath string = "/v1/customers/%s/subscriptions/%s"
)

func getBaseRequestHeaders() map[string]string {
	requestID, _ := uuid.NewRandom()
	correlationID, _ := uuid.NewRandom()

	return map[string]string{
		"Accept":        applicationJSON,
		contentType:     applicationJSON,
		msRequestID:     requestID.String(),
		msCorrelationID: correlationID.String(),
	}
}
