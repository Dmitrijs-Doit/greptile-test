package service

import (
	"context"

	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"

	tiersPkg "github.com/doitintl/firestore/pkg"
	bqLensDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/domain"
	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func (s *OptimizerService) SingleCustomerOptimizer(ctx context.Context, customerID string, payload domain.Payload) error {
	l := s.loggerProvider(ctx)
	l.SetLabels(DefaultLogFields)

	customerRef := s.customerDAL.GetRef(ctx, customerID)
	customerTier, err := s.tiers.GetCustomerTier(ctx, customerRef, tiersPkg.NavigatorPackageTierType)
	if err != nil {
		return s.handleOptimizerError(ctx, l, "GetCustomerTier", customerID, err)
	}

	connect, options, err := s.cloudConnect.NewGCPClients(ctx, customerID)
	if err != nil {
		return s.handleOptimizerError(ctx, l, "NewGCPClients", customerID, err)
	}

	crm := connect.CRM

	bq := connect.BQ.BigqueryService
	defer bq.Close()

	reservationClient, err := s.reservations.NewClient(ctx, options)
	if err != nil {
		return s.handleOptimizerError(ctx, l, "NewClient", customerID, err)
	}

	hasTableDiscovery := true

	_, err = s.serviceBQ.GetTableDiscoveryMetadata(ctx, bq)
	if err != nil {
		l.Error(wrapOperationError("GetTableDiscoveryMetadata", customerID, err).Error())

		hasTableDiscovery = false
	}

	location, projectID, err := s.serviceBQ.GetDatasetLocationAndProjectID(ctx, bq, bqLensDomain.DoitCmpDatasetID)
	if err != nil {
		return s.handleOptimizerError(ctx, l, "GetDatasetLocationAndProjectID", customerID, err)
	}

	minmaxDates, err := s.serviceBQ.GetMinAndMaxDates(ctx, bq, projectID, location)
	if err != nil {
		return s.handleOptimizerError(ctx, l, "GetMinAndMaxDates", customerID, err)
	}

	if !minmaxDates.Min.Valid && !minmaxDates.Max.Valid {
		l.Infof("min and max days are not defined for customer '%s'", customerID)

		return s.handleOptimizerError(ctx, l, "", customerID, nil)
	}

	projectReservations, reservationAssignments := s.reservations.GetProjectsWithReservations(ctx, customerID, reservationClient, crm, payload.BillingProjectWithReservation)

	var projectsByEdition map[reservationpb.Edition][]string

	if isEntitledToBQLensEditions(customerTier.Name) {
		err = s.GenerateSwitchToEditionsRecommendation(ctx, customerID, reservationAssignments, bq, projectID, location)
		if err != nil {
			return s.handleOptimizerError(ctx, l, "GenerateSwitchToEditionsRecommendation", customerID, err)
		}

		projectsByEdition = s.getProjectsByEdition(reservationAssignments)
	}

	replacements := domain.Replacements{
		ProjectID:                projectID,
		DatasetID:                bqLensDomain.DoitCmpDatasetID,
		TablesDiscoveryTable:     bqLensDomain.DoitCmpTablesTable,
		ProjectsWithReservations: projectReservations,
		ProjectsByEdition:        projectsByEdition,
		MinDate:                  minmaxDates.Min.Timestamp,
		MaxDate:                  minmaxDates.Max.Timestamp,
		Location:                 location,
	}

	periodTotal, storageRecommendation, err := s.serviceBQ.GenerateStorageRecommendation(ctx, customerID, bq, payload.Discount, replacements, s.timeNowFunc(), hasTableDiscovery)
	if err != nil {
		return s.handleOptimizerError(ctx, l, "GenerateStorageRecommendation", customerID, err)
	}

	transformerCtx := domain.TransformerContext{
		Discount:                payload.Discount,
		TotalScanPricePerPeriod: periodTotal,
	}

	executorData, executorErrors := s.executor.Execute(ctx, bq, replacements, transformerCtx, bqmodels.QueriesPerMode, hasTableDiscovery)

	for _, e := range executorErrors {
		wrappedError := wrapOperationError("Execute", customerID, e)
		l.Error(wrappedError)
	}

	err = s.dalFS.SetRecommendationDataIncrementally(ctx, customerID, mergeMaps(executorData, storageRecommendation))
	if err != nil {
		return s.handleOptimizerError(ctx, l, "SetRecommendationDataIncrementally", customerID, err)
	}

	return nil
}

func (s *OptimizerService) handleOptimizerError(ctx context.Context, log logger.ILogger, operation, customerID string, operationErr error) error {
	updateErr := s.dalFS.UpdateSimulationDetails(ctx, customerID, map[string]interface{}{"progress": 100, "status": "END"})
	if updateErr != nil {
		log.Errorf("failed to update simulation details for customer '%s': %w", customerID, updateErr)
	}

	if operationErr != nil {
		return wrapOperationError(operation, customerID, operationErr)
	}

	return nil
}

func mergeMaps(executorData, storageRecommendations dal.RecommendationSummary) dal.RecommendationSummary {
	combinedData := make(dal.RecommendationSummary)

	for queryName, timeRangeMap := range executorData {
		combinedData[queryName] = timeRangeMap
	}

	for queryName, timeRangeMap := range storageRecommendations {
		combinedData[queryName] = timeRangeMap
	}

	return combinedData
}

func isEntitledToBQLensEditions(tierName string) bool {
	return tierName == string(tiersPkg.Enhanced) || tierName == string(tiersPkg.Premium)
}
