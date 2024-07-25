package service

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

func (p *PresentationService) CopyBQLensDataToCustomers(ctx *gin.Context) error {
	docSnaps, err := p.customersDAL.GetPresentationCustomersWithAssetType(ctx, common.Assets.GoogleCloud)
	if err != nil {
		return fmt.Errorf(FetchCustomerErr, err)
	}

	for _, docSnap := range docSnaps {
		var customer common.Customer

		if err := docSnap.DataTo(&customer); err != nil {
			return err
		}

		customer.Snapshot = docSnap
		customer.ID = docSnap.Ref.ID

		if err = p.copyCustomerBQLensData(ctx, &customer); err != nil {
			return err
		}
	}

	return nil
}

func (p *PresentationService) CopyBQLensDataToCustomer(ctx *gin.Context, customerID string) error {
	customer, err := p.getDemoCustomerFromID(ctx, customerID)
	if err != nil {
		return fmt.Errorf(FetchCustomerErr, err)
	}

	if err = p.copyCustomerBQLensData(ctx, customer); err != nil {
		return err
	}

	return nil
}

func (p *PresentationService) copyCustomerBQLensData(ctx context.Context, customer *common.Customer) error {
	if err := p.optimizerPresentation.CreateSuperQuerySimulationRecommender(ctx, customer); err != nil {
		return err
	}

	if err := p.optimizerPresentation.CreateSuperQuerySimulationOptimisation(ctx, customer); err != nil {
		return err
	}

	if err := p.optimizerPresentation.CreateSuperQueryBackFillData(ctx, customer); err != nil {
		return err
	}

	return nil
}
