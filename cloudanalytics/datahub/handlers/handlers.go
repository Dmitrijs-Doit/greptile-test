package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/client"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/service/iface"
	metadataDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal"
	datahubMetricDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/datahubmetric"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type DataHub struct {
	loggerProvider logger.Provider
	datahubService iface.DataHubService
}

func NewDataHub(loggerProvider logger.Provider, conn *connection.Connection) *DataHub {
	datahubMetadataDAL := metadataDal.NewDataHubMetadataFirestoreWithClient(conn.Firestore)
	datahubMetricDAL := datahubMetricDal.NewDataHubMetricFirestoreWithClient(conn.Firestore)
	bqDAL := dal.NewDataHubBigQuery(loggerProvider, conn)

	datahubCachedDatasetDAL := dal.NewDataHubCachedDatasetFirestoreWithClient(conn.Firestore)
	datahubDatasetDAL := dal.NewDataHubDatasetsFirestoreWithClient(conn.Firestore)
	datahubBatchesDAL := dal.NewDataHubBatchesFirestoreWithClient(conn.Firestore)
	customerDAL := customerDal.NewCustomersFirestoreWithClient(conn.Firestore)

	datahubInternalAPIClient, err := client.NewDatahubInternalAPIClient()
	if err != nil {
		panic(err)
	}

	datahubInternalAPIDAL, err := dal.NewDatahubInternalAPIDAL(
		datahubInternalAPIClient,
	)
	if err != nil {
		panic(err)
	}

	svc := service.NewService(
		loggerProvider,
		datahubMetadataDAL,
		datahubMetricDAL,
		bqDAL,
		datahubCachedDatasetDAL,
		datahubDatasetDAL,
		datahubBatchesDAL,
		datahubInternalAPIDAL,
		customerDAL,
		conn.CloudTaskClient,
	)

	return &DataHub{
		loggerProvider: loggerProvider,
		datahubService: svc,
	}
}

func (h *DataHub) DeleteCustomerDatasets(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customer id"), http.StatusBadRequest)
	}

	var deleteDatasetsReq domain.DeleteDatasetsReq

	if err := ctx.ShouldBindJSON(&deleteDatasetsReq); err != nil {
		l.Errorf(domain.ParsingRequestErrorTpl, err)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	deletedBy := ctx.GetString("email")

	if err := h.datahubService.DeleteCustomerDataByClouds(ctx, customerID, deleteDatasetsReq, deletedBy); err != nil {
		if errors.Is(err, domain.ErrCloudsCanNotBeEmpty) {
			return web.NewRequestError(err, http.StatusBadRequest)
		} else if errors.Is(err, dal.ErrDeleteTookTooLongRunningAsync) {
			return web.Respond(ctx, err, http.StatusAccepted)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *DataHub) GetCustomerDatasets(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(nil, http.StatusBadRequest)
	}

	forceRefresh := ctx.Query("forceRefresh") == "true"

	res, err := h.datahubService.GetCustomerDatasets(
		ctx,
		customerID,
		forceRefresh,
	)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *DataHub) GetCustomerDatasetBatches(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(ErrMissingCustomerID, http.StatusBadRequest)
	}

	datasetName := ctx.Param("datasetName")
	if datasetName == "" {
		return web.NewRequestError(ErrMissingDatasetName, http.StatusBadRequest)
	}

	forceRefresh := ctx.Query("forceRefresh") == "true"

	res, err := h.datahubService.GetCustomerDatasetBatches(ctx, customerID, datasetName, forceRefresh)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *DataHub) AddRawEvents(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(nil, http.StatusBadRequest)
	}

	email := ctx.GetString("email")
	if email == "" {
		return web.NewRequestError(nil, http.StatusBadRequest)
	}

	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
	})

	var rawEventsReq domain.RawEventsReq

	if err := ctx.ShouldBindJSON(&rawEventsReq); err != nil {
		l.Errorf(domain.ParsingRequestErrorTpl, err)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if validationErrs := rawEventsReq.Validate(); validationErrs != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": validationErrs})
		return nil
	}

	events, validationErrs, err := h.datahubService.AddRawEvents(
		ctx,
		customerID,
		email,
		rawEventsReq,
	)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if validationErrs != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": validationErrs})
		return nil
	}

	resp := domain.AddRawEventsRes{
		EventsCount: len(events),
		Execute:     rawEventsReq.Execute,
	}
	return web.Respond(ctx, resp, http.StatusOK)
}

// DeleteCustomerData deletes all the customer's DataHub API data.
func (h *DataHub) DeleteCustomerData(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginDataHub)

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(nil, http.StatusBadRequest)
	}

	if err := h.datahubService.DeleteCustomerData(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *DataHub) DeleteCustomerSpecificEvents(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginDataHub)

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(nil, http.StatusBadRequest)
	}

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
	})

	var deleteEventsReq domain.DeleteEventsReq

	if err := ctx.ShouldBindJSON(&deleteEventsReq); err != nil {
		l.Errorf(domain.ParsingRequestErrorTpl, err)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	eventIDs := deleteEventsReq.EventIDs
	clouds := deleteEventsReq.Clouds
	deletedBy := ctx.GetString("email")

	if len(eventIDs) != 0 && len(clouds) != 0 {
		return web.NewRequestError(domain.ErrGeneralFilterWithSpecificFilter, http.StatusBadRequest)
	}

	if len(eventIDs) > 0 {
		if err := h.datahubService.DeleteCustomerDataByEventIDs(ctx, customerID, deleteEventsReq, deletedBy); err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	} else if len(clouds) > 0 {
		deleteDatasetReq := domain.DeleteDatasetsReq{
			Datasets: clouds,
		}
		if err := h.datahubService.DeleteCustomerDataByClouds(ctx, customerID, deleteDatasetReq, deletedBy); err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	} else {
		return web.NewRequestError(domain.ErrAtLeastOneFilter, http.StatusBadRequest)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *DataHub) DeleteDatasetBatches(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(ErrMissingCustomerID, http.StatusBadRequest)
	}

	datasetName := ctx.Param("datasetName")
	if datasetName == "" {
		return web.NewRequestError(ErrMissingDatasetName, http.StatusBadRequest)
	}

	var deleteBatchReq domain.DeleteBatchesReq
	if err := ctx.ShouldBindJSON(&deleteBatchReq); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if len(deleteBatchReq.Batches) == 0 {
		return web.NewRequestError(ErrMissingBatchesIDs, http.StatusBadRequest)
	}

	deletedBy := ctx.GetString("email")

	if err := h.datahubService.DeleteDatasetBatches(ctx, customerID, datasetName, deleteBatchReq, deletedBy); err != nil {
		if errors.Is(err, domain.ErrBatchIsProcessing) {
			return web.NewRequestError(err, http.StatusConflict)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *DataHub) DeleteAllCustomersDataHard(ctx *gin.Context) error {
	if err := h.datahubService.DeleteAllCustomersDataHard(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *DataHub) DeleteCustomerDataHard(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginDataHub)

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(nil, http.StatusBadRequest)
	}

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
	})

	if err := h.datahubService.DeleteCustomerDataHard(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *DataHub) CreateDataset(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(nil, http.StatusBadRequest)
	}

	email := ctx.GetString("email")

	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
	})

	var datasetReq domain.CreateDatasetRequest

	if err := ctx.ShouldBindJSON(&datasetReq); err != nil {
		l.Errorf(domain.ParsingRequestErrorTpl, err)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.datahubService.CreateDataset(ctx, customerID, email, datasetReq); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusCreated)
}
