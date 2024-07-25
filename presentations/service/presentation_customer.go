package service

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

func (p *PresentationService) getDemoCustomerFromID(ctx *gin.Context, customerID string) (*common.Customer, error) {
	customer, err := p.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	if customer.PresentationMode == nil && !customer.PresentationMode.IsPredefined {
		return nil, errors.New("customer is not a demo customer")
	}

	return customer, nil
}
