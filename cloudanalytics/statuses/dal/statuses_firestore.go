package dal

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"

	"github.com/doitintl/firestore/iface"
)

const (
	cloudAnalyticsCollection        = "cloudAnalytics"
	statusCollection                = "statuses"
	cloudAnalyicsStatusesCollection = "cloudAnalyticsStatuses"
	integrationCollection           = "integrations"
)

type ReportStatusesFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

func NewReportStatusFirestoreWithClient(fun connection.FirestoreFromContextFun) *ReportStatusesFirestore {
	return &ReportStatusesFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

// helper function to get a report status snapshot
func (d *ReportStatusesFirestore) getReportStatusDocRef(ctx context.Context, reportStatusID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(cloudAnalyticsCollection).Doc(statusCollection).Collection(cloudAnalyicsStatusesCollection).Doc(reportStatusID)
}

// GetReportStatus returns a report status struct for a given id:
func (d *ReportStatusesFirestore) GetReportStatus(ctx context.Context, reportStatusID string) (*common.ReportStatusData, error) {
	if reportStatusID == "" {
		return nil, errors.New("invalid reportStatus id")
	}

	docRef := d.getReportStatusDocRef(ctx, reportStatusID)

	snap, err := d.documentsHandler.Get(ctx, docRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var rep common.ReportStatusData

	err = snap.DataTo(&rep)
	if err != nil {
		return nil, err
	}

	return &rep, nil
}

func (d *ReportStatusesFirestore) UpdateReportStatus(ctx context.Context, report *common.ReportStatusData) error {
	docRef := d.getReportStatusDocRef(ctx, report.Customer.ID)

	//5. save the status struct back to Firestore (using Set)
	_, err := d.documentsHandler.Set(ctx, docRef, report)
	if err != nil {
		return err
	}

	return nil
}

type Cht struct {
	ID       int64                  `firestore:"id"`
	Customer *firestore.DocumentRef `firestore:"customer"`
	Disabled bool                   `firestore:"disabled"`
}

func (d *ReportStatusesFirestore) GetCloudHealthRef(ctx context.Context, chtID int) (*firestore.DocumentRef, error) {
	chtIDString := strconv.Itoa(chtID)

	customerSnap, err := d.firestoreClientFun(ctx).Collection(integrationCollection).
		Doc("cloudhealth").
		Collection("cloudhealthCustomers").Doc(chtIDString).Get(ctx)
	if err != nil {
		return nil, err
	}

	var chCustomer Cht
	if err := customerSnap.DataTo(&chCustomer); err != nil {
		return nil, fmt.Errorf("REPORT STATUS: error deserializing cloudhealth customer %s for reportStatus update err meesage %s", chtIDString, err.Error())
	}

	if chCustomer.Disabled {
		return nil, fmt.Errorf("REPORT STATUS: cloud health customer %s is disabled", chtIDString)
	}

	return chCustomer.Customer, nil
}
