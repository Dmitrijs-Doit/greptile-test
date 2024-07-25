package iface

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"
)

//go:generate mockery --name AWSAccess
type AWSAccess interface {
	GetSavingsPlansPurchaseRecommendation(input costexplorer.GetSavingsPlansPurchaseRecommendationInput, accountID string) (*costexplorer.GetSavingsPlansPurchaseRecommendationOutput, error)
	GetAWSSession(accountID, functionName string) (*session.Session, error)
}
