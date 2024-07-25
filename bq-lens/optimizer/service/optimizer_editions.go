package service

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	transformers "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service/executor"
	insightsSDK "github.com/doitintl/insights/sdk"
)

func (s *OptimizerService) GenerateSwitchToEditionsRecommendation(
	ctx context.Context,
	customerID string,
	reservationAssignments []domain.ReservationAssignment,
	bq *bigquery.Client,
	projectID string,
	location string,
) error {
	// Run this once a week
	if !s.shouldRunSwitchToEditionsRecommendation() {
		return nil
	}

	l := s.loggerProvider(ctx)

	aggregatedJobsStatistics, err := s.serviceBQ.GetAggregatedJobStatistics(ctx, bq, projectID, location)
	if err != nil {
		return s.handleOptimizerError(ctx, l, "GetAggregatedJobStatistics", customerID, err)
	}

	editionsPricebooks, err := s.pricebook.GetPricebooks(ctx)
	if err != nil {
		return s.handleOptimizerError(ctx, l, "GetPricebooks", customerID, err)
	}

	onDemandPricebook, err := s.pricebook.GetOnDemandPricebook(ctx)
	if err != nil {
		return s.handleOptimizerError(ctx, l, "GetOnDemandPricebook", customerID, err)
	}

	projectsAssignedToReservations := make(map[string]struct{})

	for _, assignment := range reservationAssignments {
		for _, project := range assignment.ProjectsList {
			projectsAssignedToReservations[project] = struct{}{}
		}
	}

	insightResponse, err := transformers.TransformSwitchToEditions(customerID, projectsAssignedToReservations, aggregatedJobsStatistics, editionsPricebooks, onDemandPricebook)
	if err != nil {
		return s.handleOptimizerError(ctx, l, "TransformSwitchToEditions", customerID, err)
	}

	err = s.insights.PostInsightResults(ctx, []insightsSDK.InsightResponse{insightResponse})
	if err != nil {
		return s.handleOptimizerError(ctx, l, "PostInsightResults", customerID, err)
	}

	return nil
}

func (s *OptimizerService) shouldRunSwitchToEditionsRecommendation() bool {
	return s.timeNowFunc().UTC().Weekday() == time.Monday
}

func (s *OptimizerService) getProjectsByEdition(
	reservationAssignments []domain.ReservationAssignment,
) map[reservationpb.Edition][]string {
	projectsByEdition := make(map[reservationpb.Edition][]string)

	for _, assignment := range reservationAssignments {
		// a project cannnot have multiple reservations assigned to it, so there's no risk of
		// duplication here.
		projectsByEdition[assignment.Reservation.Edition] = append(projectsByEdition[assignment.Reservation.Edition],
			assignment.ProjectsList...)
	}

	return projectsByEdition
}
