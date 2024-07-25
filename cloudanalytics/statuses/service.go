package statuses

import (
	"context"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	reportStatusDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/statuses/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customersDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type ReportStatusesService struct {
	loggerProvider logger.Provider
	*connection.Connection
	reportStatusDAL *reportStatusDal.ReportStatusesFirestore
	customersDAL    *customersDal.CustomersFirestore
}

func NewReportStatusesService(loggerProvider logger.Provider, conn *connection.Connection) (*ReportStatusesService, error) {
	return &ReportStatusesService{
		loggerProvider,
		conn,
		reportStatusDal.NewReportStatusFirestoreWithClient(conn.Firestore),
		customersDal.NewCustomersFirestoreWithClient(conn.Firestore),
	}, nil
}

func (s *ReportStatusesService) getCustomerRef(ctx context.Context, requestID string) (*firestore.DocumentRef, error) {
	var customerRef *firestore.DocumentRef

	var err error

	chtID, err := strconv.Atoi(requestID)
	if err != nil {
		//this is not a cht id number, so it must be a customer ref:
		customerRef = s.customersDAL.GetRef(ctx, requestID)
	} else {
		customerRef, err = s.reportStatusDAL.GetCloudHealthRef(ctx, chtID)
		if err != nil {
			return nil, err
		}
	}

	return customerRef, nil
}

func (s *ReportStatusesService) UpdateReportStatus(ctx context.Context, customerID string, statusType common.ReportStatus) error {
	statusStruct, err := s.generateReportStatus(ctx, customerID, statusType)
	if err != nil {
		return err
	}

	return s.reportStatusDAL.UpdateReportStatus(ctx, statusStruct)
}

func (s *ReportStatusesService) generateReportStatus(ctx context.Context, customerID string, newStatus common.ReportStatus) (*common.ReportStatusData, error) {
	l := s.loggerProvider(ctx)

	var customerRef *firestore.DocumentRef
	//1. get correct customer ref - if we have a chtID then from integrations collection - else from customers collection:
	customerRef, err := s.getCustomerRef(ctx, customerID)
	if err != nil {
		return nil, err
	}

	if _, ok := newStatus.Status[string(common.AWSReportStatus)]; ok {
		l.Infof("REPORT STATUS: Found CloudHealth customer %s for %s", customerRef.ID, customerID)
	}

	//2. once we for sure have a ref, check if a reportStatus exists for this customer, if not, create one
	statusDoc, err := s.reportStatusDAL.GetReportStatus(ctx, customerRef.ID)
	if err != nil && err != doitFirestore.ErrNotFound {
		return nil, err
	}

	prevOverallLastUpdate := time.Time{}

	if statusDoc == nil {
		//initialize the status struct:
		statusDoc = &common.ReportStatusData{
			Customer: customerRef,
			Status:   newStatus.Status,
		}
	} else {
		//3. if a reportStatus exists, update the reportStatus with the new status
		for key, val := range newStatus.Status {
			statusDoc.Status[key] = val
		}

		prevOverallLastUpdate = statusDoc.OverallLastUpdate
	}

	statusDoc.OverallLastUpdate = getMostRecentTimestamp(prevOverallLastUpdate, newStatus)
	statusDoc.TimeModified = time.Time{}

	return statusDoc, nil
}

func getMostRecentTimestamp(oldLastUpdate time.Time, newStatus common.ReportStatus) time.Time {
	mostRecentTimestamp := oldLastUpdate

	for key, val := range newStatus.Status {
		//the OverallLastUpdate is only affected by gcp or AWS statuses, or GKE with Status Enabled:
		if key != string(common.GKEReportStatus) || common.GKEStatus(newStatus.Status[string(common.GKEReportStatus)].Status) == common.GKEStatusEnabled {
			if val.LastUpdate.After(mostRecentTimestamp) {
				mostRecentTimestamp = val.LastUpdate
			}
		}
	}

	return mostRecentTimestamp
}
