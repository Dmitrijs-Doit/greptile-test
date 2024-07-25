package iface

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type PresentationService interface {
	AggregateBillingDataAWS(ctx *gin.Context) error
	AggregateBillingDataAzure(ctx *gin.Context) error
	AggregateBillingDataGCP(ctx *gin.Context) error
	ChangePresentationMode(ctx context.Context, customerID string) error
	CopyBQLensDataToCustomers(ctx *gin.Context) error
	CopyBQLensDataToCustomer(ctx *gin.Context, customerID string) error
	CopySpotScalingDataToCustomers(ctx *gin.Context) error
	CopySpotScalingDataToCustomer(ctx *gin.Context, customerID string) error
	CopyCostAnomaliesToCustomers(ctx *gin.Context) error
	CopyCostAnomaliesToCustomer(ctx *gin.Context, customerID string) error
	CreateCustomer(
		ctx *gin.Context,
	) (*common.Customer, error)
	UpdateAWSAssets(ctx *gin.Context, customerID string) error
	UpdateAWSBillingData(ctx *gin.Context, incrementalUpdate bool) error
	UpdateCustomerAzureBillingData(ctx *gin.Context, customerID string, incrementalUpdate bool) error
	UpdateAzureBillingData(ctx *gin.Context, incrementalUpdate bool) error
	UpdateEKSLensBillingData(ctx *gin.Context, incrementalUpdate bool) error
	UpdateCustomerEKSLensBillingData(ctx *gin.Context, customerID string, incrementalUpdate bool) error
	UpdateAzureAssets(ctx *gin.Context, customerID string) error
	UpdateAssetsMetadataAWS(ctx *gin.Context) error
	UpdateAssetsMetadataGCP(ctx *gin.Context) error
	UpdateAssetsMetadataAzure(ctx *gin.Context) error
	UpdateCustomerAWSBillingData(ctx *gin.Context, customerID string, incrementalUpdate bool) error
	UpdateGCPAssets(ctx *gin.Context, customerID string) error
	UpdateGCPBillingData(ctx *gin.Context, incrementalUpdate bool) error
	UpdateCustomerGCPBillingData(ctx *gin.Context, customerID string, incrementalUpdate bool) error
	UpdateFlexsaveAWSSavings(ctx *gin.Context) error

	CreateWidgetUpdateTasks(ctx *gin.Context) error
	DeletePresentationCustomerAssets(ctx *gin.Context, customerID string) error
	DispatchCloudTask(ctx *gin.Context, path string, queue common.TaskQueue) error
	ReceiveDoneUpdateMetadataAssetsMessage(ctx *gin.Context, assetTypes []string, sessionID string) error
	SendDoneSyncMessage(ctx *gin.Context, assetType string) error
	SendErrorSyncMessage(ctx *gin.Context, assetType string, err error) error
	GetPresentationCustomersAssetTypes(ctx *gin.Context) (map[string]bool, error)
}
