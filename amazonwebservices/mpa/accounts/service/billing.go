package service

import (
	"context"

	dal2 "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"
)

var riAccountIDs = []string{
	"781862544523", // DoiT RI Account #1
	"119448299328", // DoiT RI Account #2
	"989413324604", // DoiT RI Account #3
	"652313714033", // DoiT RI Account #4
	"726085735545", // DoiT RI Account #5
}

type BillingService struct {
	billing dal2.Billing
	flexAPI flexapi.FlexAPI
}

func NewBillingService(projectID string) (*BillingService, error) {
	billing, err := dal2.NewBillingDAL(context.Background(), projectID)
	if err != nil {
		return nil, err
	}

	flexAPI, err := flexapi.NewFlexAPIService()
	if err != nil {
		return nil, err
	}

	return &BillingService{
		billing: billing,
		flexAPI: flexAPI,
	}, nil
}

func (s *BillingService) GetCoveredUsage(ctx context.Context, accountID string, from iface.Payer) (iface.CoveredUsage, error) {
	spARNs, err := s.flexAPI.ListARNs(ctx)
	if err != nil {
		return iface.CoveredUsage{}, err
	}

	number, err := from.GetNumber()
	if err != nil {
		return iface.CoveredUsage{}, err
	}

	usage, err := s.billing.GetCoveredUsage(ctx, accountID, from.ID, number, spARNs, riAccountIDs)
	if err != nil {
		return iface.CoveredUsage{}, err
	}

	return iface.NewCoveredUsage(usage.SPCost, usage.RICost), nil
}
