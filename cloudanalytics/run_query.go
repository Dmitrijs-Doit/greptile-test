package cloudanalytics

import (
	"context"

	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
)

type RunQueryInput struct {
	CustomerID              string
	PresentationModeEnabled bool
	ReportID                string
	Email                   string
}

// reservationCustomers is a map of customer IDs that should use reservation to run reports
// due to their high spend when running on demand.
var reservationCustomers = map[string]struct{}{
	"Z6G1etWHY6U3AG0WcDOs":            {}, // cybereason.com
	"KzDE3pXU4KeQ3b53ciDN":            {}, // exabeam.com
	"9CH0gRRoJ6sVdtGXS5lA":            {}, // optimove.com
	"EE8CtpzYiKp0dVAESVrB":            {}, // doit.com
	"jRRyh8x04k1c29Nq7ywZ":            {}, // rubrik.com
	"LcgELbXV21Imef3utMoh":            {}, // connatix.com
	"2Gi0e4pPA3wsfJNOOohW":            {}, // moonactive.com
	"YpNs3QEkXXhqFcxjKQ8D":            {}, // melio.com
	"ssnM3o1cMrd4aregLP67":            {}, // lightcast.io
	"Rt5zFJQEiihiOmjvnVux":            {}, // sift.com
	"E6PACEaUBOSpSFcxWUFA":            {}, // yugabyte.com
	"TKlz8pLET3Z7F0LWFyHB":            {}, // toptal.com
	"IHSePotIFuhEWuaQiuhU":            {}, // walkme.com
	"presentationcustomerAWSAzureGCP": {}, // presentation dataset - contains combined data from connatix, moonactive and others
}

// isReservationCustomer returns true if the customer ID exists in the reservationCustomers map.
func isReservationCustomer(customerID string) bool {
	_, ok := reservationCustomers[customerID]
	return ok
}

func (s *CloudAnalyticsService) RunQuery(ctx context.Context, qr *QueryRequest, params RunQueryInput) (*QueryResult, error) {
	// Query requests coming from client are "on demand"
	// - The query will run using the "online reports" project in production
	// - The query will have a deadline of 3 minutes to complete
	qr.IsCSP = params.CustomerID == domainQuery.CSPCustomerID

	// For CSP client origin query requests we use a different project.
	if qr.IsCSP || isReservationCustomer(params.CustomerID) {
		qr.Origin = domainOrigin.QueryOriginClientReservation
	}

	if qr.Type == "report" && params.ReportID != "" {
		report, err := s.GetReport(ctx, params.CustomerID, params.ReportID, params.PresentationModeEnabled)
		if err != nil {
			return nil, err
		}

		qr.Organization = report.Organization
		qr.IsPreset = report.Type == "preset"

		err = s.reportDAL.UpdateTimeLastRun(ctx, params.ReportID, domainOrigin.QueryOriginClient)
		if err != nil {
			s.loggerProvider(ctx).Errorf("failed to update last time run for report %s; %s", params.ReportID, err)
		}
	}

	result, err := s.GetQueryResult(ctx, qr, params.CustomerID, params.Email)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
