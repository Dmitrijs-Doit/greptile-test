package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
)

//go:generate mockery --name ContractService --output=./../mocks
type ContractService interface {
	CreateContract(ctx context.Context, req domain.ContractInputStruct) error
	CancelContract(ctx context.Context, contractID string) error
	AggregateInvoiceData(ctx context.Context, invoiceMonth, contractID string) error
	RefreshCustomerTiers(ctx context.Context, customerID string) error
	RefreshAllCustomerTiers(ctx context.Context) error
	ExportContracts(ctx context.Context) error
	UpdateContract(ctx context.Context, contractID string, req domain.ContractUpdateInputStruct, email string, userName string) error
	UpdateGoogleCloudContractsSupport(ctx context.Context) error
	DeleteContract(ctx context.Context, contractID string) error
}
