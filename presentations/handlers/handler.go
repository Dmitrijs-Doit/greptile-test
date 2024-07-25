package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/presentations/domain"
	"github.com/doitintl/hello/scheduled-tasks/presentations/iface"
	presentationLogger "github.com/doitintl/hello/scheduled-tasks/presentations/log"
	"github.com/doitintl/hello/scheduled-tasks/presentations/service"
)

type Presentation struct {
	loggerProvider logger.Provider
	service        iface.PresentationService
}

const schedulerPath = "/tasks/presentation/scheduler"

func getSchedulerPathWithSessionIDFromContext(ctx *gin.Context, path string, queryParams map[string]string) string {
	return getSchedulerPathWithSessionID(path, ctx.Query(domain.SessionIDQueryParamName), queryParams)
}

func getSchedulerPathWithSessionID(path string, sessionID string, queryParams map[string]string) string {
	var schedulerURL *url.URL

	schedulerURL, err := url.Parse(schedulerPath)
	if err != nil {
		panic("should not happen")
	}

	schedulerURL.Path, err = url.JoinPath(schedulerURL.Path, path)
	parameters := url.Values{}

	for key, value := range queryParams {
		parameters.Add(key, value)
	}

	parameters.Add(domain.SessionIDQueryParamName, sessionID)
	schedulerURL.RawQuery = parameters.Encode()

	return schedulerURL.String()
}

func NewPresentation(
	log logger.Provider,
	conn *connection.Connection,
) *Presentation {
	enhancedLoggerProvider := func(ctx context.Context) logger.ILogger {
		return presentationLogger.GetPresentationLogger(log(ctx))
	}

	presentationService, err := service.NewPresentationService(
		enhancedLoggerProvider,
		conn,
	)
	if err != nil {
		panic(err)
	}

	return &Presentation{
		loggerProvider: enhancedLoggerProvider,
		service:        presentationService,
	}
}

func (h *Presentation) ChangePresentationMode(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	if err := h.service.ChangePresentationMode(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateCustomerAWSAssets(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)

	if err := h.service.UpdateAWSAssets(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateCustomerAWSBillingData(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)

	incrementalUpdate := ctx.Query("incrementalUpdate") == "true"

	if err := h.service.UpdateCustomerAWSBillingData(ctx, customerID, incrementalUpdate); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateCustomerAzureBillingData(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)

	incrementalUpdate := ctx.Query("incrementalUpdate") == "true"

	if err := h.service.UpdateCustomerAzureBillingData(ctx, customerID, incrementalUpdate); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateCustomerGCPBillingData(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	incrementalUpdate := ctx.Query("incrementalUpdate") == "true"

	if err := h.service.UpdateCustomerGCPBillingData(ctx, customerID, incrementalUpdate); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateCustomerAzureAssets(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)

	if err := h.service.UpdateAzureAssets(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) CreateCustomer(ctx *gin.Context) error {
	c, err := h.service.CreateCustomer(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)

	return web.Respond(ctx, c, http.StatusOK)
}

func (h *Presentation) UpdateCustomGcpAssets(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	if customerID == "" {
		return web.NewRequestError(errors.New("missing customer id"), http.StatusBadRequest)
	}

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)

	if err := h.service.UpdateGCPAssets(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) SyncCustomersBillingData(ctx *gin.Context) error {
	sessionID := uuid.New().String()
	incrementalUpdate := ctx.Query("incrementalUpdate")
	updateBQLensData := ctx.Query("updateBQLensData") == "true"

	assetTypes, err := h.service.GetPresentationCustomersAssetTypes(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	billingParams := map[string]string{
		"incrementalUpdate": incrementalUpdate,
	}

	if _, ok := assetTypes[common.Assets.AmazonWebServices]; ok {
		if err := h.service.DispatchCloudTask(ctx, getSchedulerPathWithSessionID("update-aws-billing-data", sessionID, billingParams), common.TaskQueuePresentationAWS); err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	if _, ok := assetTypes[common.Assets.GoogleCloud]; ok {
		if err := h.service.DispatchCloudTask(ctx, getSchedulerPathWithSessionID("update-gcp-billing-data", sessionID, billingParams), common.TaskQueuePresentationGCP); err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	if _, ok := assetTypes[common.Assets.MicrosoftAzure]; ok {
		if err := h.service.DispatchCloudTask(ctx, getSchedulerPathWithSessionID("update-azure-billing-data", sessionID, billingParams), common.TaskQueuePresentationAzure); err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	assetTypesToWaitFor := []string{}
	for assetType := range assetTypes {
		assetTypesToWaitFor = append(assetTypesToWaitFor, assetType)
	}

	if err := h.service.ReceiveDoneUpdateMetadataAssetsMessage(ctx, assetTypesToWaitFor, sessionID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.service.DispatchCloudTask(ctx, getSchedulerPathWithSessionID("update-widgets", sessionID, nil), common.TaskQueuePresentationAWS); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if updateBQLensData {
		if err := h.service.DispatchCloudTask(ctx, getSchedulerPathWithSessionID("copy-bq-lens-data", sessionID, nil), common.TaskQueuePresentationAWS); err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	if err := h.service.DispatchCloudTask(ctx, getSchedulerPathWithSessionID("copy-spot-scaling-data", sessionID, nil), common.TaskQueuePresentationAWS); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.service.DispatchCloudTask(ctx, getSchedulerPathWithSessionID("copy-cost-anomalies", sessionID, nil), common.TaskQueuePresentationAWS); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateGCPBillingData(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)
	incrementalUpdate := ctx.Query("incrementalUpdate") == "true"

	if err := h.service.UpdateGCPBillingData(ctx, incrementalUpdate); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.GoogleCloud, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	taskPath := getSchedulerPathWithSessionIDFromContext(ctx, "aggregate-gcp-billing-data", nil)
	if err := h.service.DispatchCloudTask(ctx, taskPath, common.TaskQueuePresentationGCP); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.GoogleCloud, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateAWSBillingData(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)
	incrementalUpdate := ctx.Query("incrementalUpdate")

	if err := h.service.UpdateAWSBillingData(ctx, incrementalUpdate == "true"); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.AmazonWebServices, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	taskPath := getSchedulerPathWithSessionIDFromContext(ctx, "update-eks-billing", map[string]string{
		"incrementalUpdate": incrementalUpdate,
	})
	if err := h.service.DispatchCloudTask(ctx, taskPath, common.TaskQueuePresentationAWS); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.AmazonWebServices, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateAzureBillingData(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)
	incrementalUpdate := ctx.Query("incrementalUpdate") == "true"

	if err := h.service.UpdateAzureBillingData(ctx, incrementalUpdate); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.MicrosoftAzure, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	taskPath := getSchedulerPathWithSessionIDFromContext(ctx, "aggregate-azure-billing-data", nil)
	if err := h.service.DispatchCloudTask(ctx, taskPath, common.TaskQueuePresentationAzure); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.MicrosoftAzure, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) AggregateBillingDataGCP(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)
	l := h.loggerProvider(ctx)
	l.SetLabel(presentationLogger.LabelPresentationUpdateStage.String(), "aggregation")

	if err := h.service.AggregateBillingDataGCP(ctx); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.GoogleCloud, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.Respond(ctx, err, http.StatusMultiStatus)
	}

	if err := h.service.DispatchCloudTask(ctx, getSchedulerPathWithSessionIDFromContext(ctx, "update-gcp-assets-metadata", nil), common.TaskQueuePresentationGCP); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.GoogleCloud, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	l.Infof("GCP data aggregation completed")

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) AggregateBillingDataAWS(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)

	if err := h.service.AggregateBillingDataAWS(ctx); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.AmazonWebServices, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.Respond(ctx, err, http.StatusMultiStatus)
	}

	if err := h.service.DispatchCloudTask(ctx, getSchedulerPathWithSessionIDFromContext(ctx, "update-aws-assets-metadata", nil), common.TaskQueuePresentationAWS); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.AmazonWebServices, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) AggregateBillingDataAzure(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)
	if err := h.service.AggregateBillingDataAzure(ctx); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.MicrosoftAzure, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.Respond(ctx, err, http.StatusMultiStatus)
	}

	if err := h.service.DispatchCloudTask(ctx, getSchedulerPathWithSessionIDFromContext(ctx, "update-azure-assets-metadata", nil), common.TaskQueuePresentationAzure); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.MicrosoftAzure, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateAssetsMetadataGCP(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)
	err := h.service.UpdateAssetsMetadataGCP(ctx)
	if err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.GoogleCloud, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.service.SendDoneSyncMessage(ctx, common.Assets.GoogleCloud); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateAssetsMetadataAWS(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)

	err := h.service.UpdateAssetsMetadataAWS(ctx)
	if err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.AmazonWebServices, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.service.DispatchCloudTask(ctx, getSchedulerPathWithSessionIDFromContext(ctx, "update-flexsave-aws-savings", nil), common.TaskQueuePresentationAWS); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.AmazonWebServices, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateFlexsaveAWSSavings(ctx *gin.Context) error {
	err := h.service.UpdateFlexsaveAWSSavings(ctx)
	if err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.AmazonWebServices, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.service.SendDoneSyncMessage(ctx, common.Assets.AmazonWebServices); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateAssetsMetadataAzure(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)
	err := h.service.UpdateAssetsMetadataAzure(ctx)
	if err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.MicrosoftAzure, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.service.SendDoneSyncMessage(ctx, common.Assets.MicrosoftAzure); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateWidgets(ctx *gin.Context) error {
	if err := h.service.CreateWidgetUpdateTasks(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) DeletePresentationCustomerAssets(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	if err := h.service.DeletePresentationCustomerAssets(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) CopyBQLensDataToCustomers(ctx *gin.Context) error {
	if err := h.service.CopyBQLensDataToCustomers(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) CopySpotScalingDataToCustomers(ctx *gin.Context) error {
	if err := h.service.CopySpotScalingDataToCustomers(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) CopyCostAnomaliesToCustomers(ctx *gin.Context) error {
	if err := h.service.CopyCostAnomaliesToCustomers(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateCustomerEKSLensBillingData(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	incrementalUpdate := ctx.Query("incrementalUpdate") == "true"

	if err := h.service.UpdateCustomerEKSLensBillingData(ctx, customerID, incrementalUpdate); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) UpdateEKSBillingData(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginPresentationModeSync)
	incrementalUpdate := ctx.Query("incrementalUpdate")

	if err := h.service.UpdateEKSLensBillingData(ctx, incrementalUpdate == "true"); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.AmazonWebServices, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	taskPath := getSchedulerPathWithSessionIDFromContext(ctx, "aggregate-aws-billing-data", nil)
	if err := h.service.DispatchCloudTask(ctx, taskPath, common.TaskQueuePresentationAWS); err != nil {
		sendErr := h.service.SendErrorSyncMessage(ctx, common.Assets.AmazonWebServices, err)
		if sendErr != nil {
			err = sendErr
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) CopyBQLensDataToCustomer(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	if err := h.service.CopyBQLensDataToCustomer(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) CopySpotScalingDataToCustomer(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	if err := h.service.CopySpotScalingDataToCustomer(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Presentation) CopyCostAnomaliesToCustomer(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	if err := h.service.CopyCostAnomaliesToCustomer(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}
