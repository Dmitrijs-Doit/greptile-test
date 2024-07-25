package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/customer/domain"
	"github.com/gin-gonic/gin"
)

//go:generate mockery --output=./mocks --all
type ICustomerService interface {
	ClearCustomerUsersNotifications(ctx context.Context, customerID string) error
	RestoreCustomerUsersNotifications(ctx context.Context, customerID string) error
	SetCustomerAssetTypes(ctx *gin.Context)
	Delete(
		ctx context.Context,
		customerID string,
		execute bool,
	) error
	ListAccountManagers(ctx context.Context, customerID string) (*domain.AccountManagerListAPI, error)
	UpdateAllCustomersSegment(ctx context.Context) ([]error, error)
	UpdateSegment(ctx context.Context, customerID string) error
}
