package cloudanalytics

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	optimizerDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/bqlens"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func (s *CloudAnalyticsService) getBQLensQueryArgs(
	ctx context.Context,
	customerID string,
	bq *bigquery.Client,
	qr *QueryRequest,
	useProxy bool,
) (*bqlens.BQLensQueryArgs, error) {
	l := s.loggerProvider(ctx)

	if qr.DataSource == nil || *qr.DataSource != report.DataSourceBQLens {
		return nil, nil
	}

	projectsWithEditions, err := s.optmizerBQ.GetBillingProjectsWithEditionsSingleCustomer(ctx, bq, customerID)
	if err != nil {
		return nil, err
	}

	var customerBQLogsTableID string

	if useProxy {
		if err := s.proxyClient.Ping(ctx); err != nil {
			return nil, err
		}

		l.Info("ping successful")

		customerBQLogsTableID, err = s.proxyClient.GetCustomerBQLogsSinkTable(ctx, customerID)
		if err != nil {
			return nil, err
		}
	} else {
		bq, err := bqlens.GetCustomerBQClient(ctx, s.conn.Firestore(ctx), customerID)
		if err != nil {
			return nil, err
		}
		defer bq.Close()

		customerBQLogsTableID, err = bqlens.GetCustomerBQLogsSinkTable(ctx, bq)
		if err != nil {
			return nil, err
		}
	}

	if len(projectsWithEditions) == 0 {
		return &bqlens.BQLensQueryArgs{
			CustomerBQLogsTableID: customerBQLogsTableID,
		}, nil
	}

	connect, options, err := s.cloudConnect.NewGCPClients(ctx, customerID)
	if err != nil {
		return nil, err
	}

	crm := connect.CRM

	reservationClient, err := s.bqLensReservations.NewClient(ctx, options)
	if err != nil {
		return nil, err
	}

	_, reservationAssignments := s.bqLensReservations.GetProjectsWithReservations(
		ctx,
		customerID,
		reservationClient,
		crm,
		projectsWithEditions)

	var reservationMappingWithClause string

	if len(reservationAssignments) > 0 {
		startTime := (*qr.TimeSettings.From).Format(time.DateOnly)
		endTime := (*qr.TimeSettings.To).Format(time.DateOnly)
		reservationMappingWithClause = bqlens.GetReservationMappingWithClause(customerBQLogsTableID, startTime, endTime)
	}

	pricebooks, err := s.bqLensPricebook.GetPricebooks(ctx)
	if err != nil {
		return nil, err
	}

	capacityCommitments := []optimizerDomain.CapacityCommitment{}

	perProjectCapacityCommitments := s.bqLensReservations.GetCapacityCommitments(ctx, customerID, reservationClient, projectsWithEditions)
	for _, capacityCommitment := range perProjectCapacityCommitments {
		capacityCommitments = append(capacityCommitments, capacityCommitment...)
	}

	bqLensQueryArgs := &bqlens.BQLensQueryArgs{
		ReservationAssignments:       reservationAssignments,
		CapacityCommitments:          capacityCommitments,
		Pricebooks:                   pricebooks,
		CustomerBQLogsTableID:        customerBQLogsTableID,
		ReservationMappingWithClause: reservationMappingWithClause,
		StartTime:                    *qr.TimeSettings.From,
		EndTime:                      *qr.TimeSettings.To,
	}

	if bqlens.IsLegacyFlatRateUser(customerID) {
		startDate := (*qr.TimeSettings.From).Format(time.DateOnly)
		endDate := (*qr.TimeSettings.To).Format(time.DateOnly)

		flatRateUsageTypes, err := getFlatRateSKUUsage(ctx, bq, qr.Origin, customerID, startDate, endDate)
		if err != nil {
			return nil, err
		}

		bqLensQueryArgs.FlatRateUsageTypes = flatRateUsageTypes
	}

	return bqLensQueryArgs, nil
}
