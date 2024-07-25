package domain

import (
	"time"
)

type AwsAccountCommandType string

const (
	AwsAccountCommandTypeCreateAwsAccount AwsAccountCommandType = "CREATE_AWS_ACCOUNT"
)

type AwsAccountCommandStatus string

const (
	AwsAccountCommandStatusScheduled  AwsAccountCommandStatus = "SCHEDULED"
	AwsAccountCommandStatusAwaiting   AwsAccountCommandStatus = "AWAITING"
	AwsAccountCommandStatusRunning    AwsAccountCommandStatus = "RUNNING"
	AwsAccountCommandStatusFailed     AwsAccountCommandStatus = "FAILED"
	AwsAccountCommandStatusTerminated AwsAccountCommandStatus = "TERMINATED"
)

type AwsAccountCommand struct {
	AccountID           string                  `firestore:"accountId"`
	ErrorMessage        *string                 `firestore:"errorMessage"`
	TimeCreated         time.Time               `firestore:"timeCreated,serverTimestamp"`
	ProcessingStartedAt *time.Time              `firestore:"processingStartedAt"`
	RetryCount          int                     `firestore:"retryCount"`
	Status              AwsAccountCommandStatus `firestore:"status"`
	Type                AwsAccountCommandType   `firestore:"type"`
}
