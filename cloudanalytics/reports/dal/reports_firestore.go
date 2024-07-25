package dal

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	labelsDAL "github.com/doitintl/hello/scheduled-tasks/labels/dal"
	labelsDALIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
)

const (
	ReportsCollection  = "dashboards/google-cloud-reports/savedReports"
	collaboratorsField = "collaborators"
	publicField        = "public"
	timeLastRunField   = "timeLastRun"
	customReportType   = "custom"

	statsField               = "stats"
	serverDurationMsField    = "serverDurationMs"
	totalBytesProcessedField = "totalBytesProcessed"
)

// ReportsFirestore is used to interact with cloud analytics reports stored on Firestore.
type ReportsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	customerDAL        customerDAL.Customers
	documentsHandler   iface.DocumentsHandler
	timeFunc           func() time.Time
	labelsDAL          labelsDALIface.Labels
}

// NewReportsFirestore returns a new ReportsFirestore instance with given project id.
func NewReportsFirestore(ctx context.Context, projectID string) (*ReportsFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewReportsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewReportsFirestoreWithClient returns a new ReportsFirestore using given client.
func NewReportsFirestoreWithClient(fun connection.FirestoreFromContextFun) *ReportsFirestore {
	return &ReportsFirestore{
		firestoreClientFun: fun,
		customerDAL:        customerDAL.NewCustomersFirestoreWithClient(fun),
		documentsHandler:   doitFirestore.DocumentHandler{},
		timeFunc:           time.Now,
		labelsDAL:          labelsDAL.NewLabelsFirestoreWithClient(fun),
	}
}

func (d *ReportsFirestore) GetRef(ctx context.Context, reportID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(ReportsCollection).Doc(reportID)
}

// Get returns a cloud analytics report data.
func (d *ReportsFirestore) Get(ctx context.Context, reportID string) (*report.Report, error) {
	if reportID == "" {
		return nil, ErrInvalidReportID
	}

	docRef := d.GetRef(ctx, reportID)

	docSnap, err := d.documentsHandler.Get(ctx, docRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var report report.Report

	if err := docSnap.DataTo(&report); err != nil {
		return nil, err
	}

	report.ID = docSnap.ID()
	report.Ref = docRef

	return &report, nil
}

// Create creates report in firestore.
func (d *ReportsFirestore) Create(
	ctx context.Context,
	tx *firestore.Transaction,
	report *report.Report,
) (*report.Report, error) {
	docRef := d.firestoreClientFun(ctx).Collection(ReportsCollection).NewDoc()

	if tx != nil {
		if err := tx.Create(docRef, report); err != nil {
			return nil, err
		}
	} else {
		if _, err := d.documentsHandler.Create(ctx, docRef, report); err != nil {
			return nil, err
		}
	}

	report.ID = docRef.ID
	report.Ref = docRef

	return report, nil
}

func (d *ReportsFirestore) GetCustomerReports(ctx context.Context, customerID string) ([]*report.Report, error) {
	if customerID == "" {
		return nil, errors.New("invalid customer id")
	}

	customerRef := d.customerDAL.GetRef(ctx, customerID)

	iter := d.firestoreClientFun(ctx).Collection(ReportsCollection).
		Where("customer", "==", customerRef).
		Where("draft", "==", false).
		Documents(ctx)

	docSnaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	reports := make([]*report.Report, len(docSnaps))

	for i, docSnap := range docSnaps {
		var report report.Report
		if err := docSnap.DataTo(&report); err != nil {
			return nil, err
		}

		report.ID = docSnap.ID()

		reports[i] = &report
	}

	return reports, nil
}

// Delete deletes a report from firestore.
func (d *ReportsFirestore) Delete(ctx context.Context, reportID string) error {
	if reportID == "" {
		return ErrInvalidReportID
	}

	docRef := d.GetRef(ctx, reportID)

	return d.labelsDAL.DeleteObjectWithLabels(ctx, docRef)
}

func (d *ReportsFirestore) getCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(ReportsCollection)
}

func (d *ReportsFirestore) getRef(ctx context.Context, reportID string) *firestore.DocumentRef {
	return d.getCollection(ctx).Doc(reportID)
}

// Share the report with the given collaborators, and set the report public access.
func (d *ReportsFirestore) Share(ctx context.Context, reportID string, collaborators []collab.Collaborator, public *collab.PublicAccess) error {
	reportRef := d.getRef(ctx, reportID)

	if _, err := d.documentsHandler.Update(ctx, reportRef, []firestore.Update{
		{
			FieldPath: []string{collaboratorsField},
			Value:     collaborators,
		}, {
			FieldPath: []string{publicField},
			Value:     public,
		},
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return doitFirestore.ErrNotFound
		}

		return err
	}

	return nil
}

func (d *ReportsFirestore) Update(ctx context.Context, reportID string, report *report.Report) error {
	reportRef := d.getRef(ctx, reportID)

	_, err := reportRef.Get(ctx)
	if err != nil {
		return err
	}

	update := []firestore.Update{
		{
			FieldPath: []string{reportFieldCollaborators},
			Value:     report.Collaborators,
		},
		{
			FieldPath: []string{reportFieldPublic},
			Value:     report.Public,
		},
		{
			FieldPath: []string{reportFieldConfig},
			Value:     report.Config,
		},
		{
			FieldPath: []string{reportFieldCustomer},
			Value:     report.Customer,
		},
		{
			FieldPath: []string{reportFieldDescription},
			Value:     report.Description,
		},
		{
			FieldPath: []string{reportFieldDraft},
			Value:     report.Draft,
		},
		{
			FieldPath: []string{reportFieldName},
			Value:     report.Name,
		},
		{
			FieldPath: []string{reportFieldOrganization},
			Value:     report.Organization,
		},
		{
			FieldPath: []string{reportFieldSchedule},
			Value:     report.Schedule,
		},
		{
			FieldPath: []string{reportFieldTimeModified},
			Value:     firestore.ServerTimestamp,
		},
		{
			FieldPath: []string{reportFieldType},
			Value:     report.Type,
		},
		{
			FieldPath: []string{reportFieldWidgetEnabled},
			Value:     report.WidgetEnabled,
		},
		{
			FieldPath: []string{reportFieldHidden},
			Value:     report.Hidden,
		},
		{
			FieldPath: []string{reportFieldCloud},
			Value:     report.Cloud,
		},
		{
			FieldPath: []string{reportFieldLabels},
			Value:     report.Labels,
		},
	}

	if _, err := d.documentsHandler.Update(
		ctx,
		reportRef,
		update,
	); err != nil {
		return err
	}

	return nil
}

var allowedTimeLastRunOriginKeys = []domainOrigin.QueryOrigin{
	domainOrigin.QueryOriginClient,
	domainOrigin.QueryOriginClientReservation,
	domainOrigin.QueryOriginWidgets,
	domainOrigin.QueryOriginScheduledReports,
	domainOrigin.QueryOriginReportsAPI,
}

func (d *ReportsFirestore) validateAllowedOriginKey(origin domainOrigin.QueryOrigin) error {
	if !slices.Contains(allowedTimeLastRunOriginKeys, origin) {
		return ErrInvalidTimeLastRunKey
	}

	return nil
}

func (d *ReportsFirestore) UpdateTimeLastRun(ctx context.Context, reportID string, origin domainOrigin.QueryOrigin) error {
	fs := d.firestoreClientFun(ctx)

	err := d.validateAllowedOriginKey(origin)
	if err != nil {
		return err
	}

	if origin == domainOrigin.QueryOriginClientReservation {
		origin = domainOrigin.QueryOriginClient
	}

	return fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		reportRef := d.getRef(ctx, reportID)

		docSnap, err := tx.Get(reportRef)
		if err != nil {
			return err
		}

		var rep report.Report

		if err := docSnap.DataTo(&rep); err != nil {
			return err
		}

		if rep.Type != customReportType {
			return nil
		}

		return tx.Update(
			reportRef,
			[]firestore.Update{
				{
					FieldPath: []string{timeLastRunField, origin},
					Value:     d.timeFunc().UTC(),
				},
			},
		)
	})
}

func (d *ReportsFirestore) GetByMetricRef(ctx context.Context, metricRef *firestore.DocumentRef) ([]*report.Report, error) {
	if metricRef == nil {
		return nil, ErrInvalidReportID
	}

	iter := d.firestoreClientFun(ctx).Collection(ReportsCollection).
		Where("config.calculatedMetric", "==", metricRef).
		Documents(ctx)

	docSnaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	reports := make([]*report.Report, len(docSnaps))

	for i, docSnap := range docSnaps {
		var report report.Report
		if err := docSnap.DataTo(&report); err != nil {
			return nil, err
		}

		report.ID = docSnap.ID()

		reports[i] = &report
	}

	return reports, nil
}

func (d *ReportsFirestore) UpdateStats(
	ctx context.Context,
	reportID string,
	origin domainOrigin.QueryOrigin,
	serverDurationMs *int64,
	totalBytesProcessed *int64,
) error {
	fs := d.firestoreClientFun(ctx)

	err := d.validateAllowedOriginKey(origin)
	if err != nil {
		return err
	}

	if origin == domainOrigin.QueryOriginClientReservation {
		origin = domainOrigin.QueryOriginClient
	}

	return fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		reportRef := d.getRef(ctx, reportID)

		docSnap, err := tx.Get(reportRef)
		if err != nil {
			return err
		}

		var rep report.Report

		if err := docSnap.DataTo(&rep); err != nil {
			return err
		}

		if rep.Type != customReportType {
			return nil
		}

		var updates []firestore.Update

		if serverDurationMs != nil {
			updates = append(
				updates,
				firestore.Update{
					FieldPath: []string{statsField, origin, serverDurationMsField},
					Value:     serverDurationMs,
				},
			)
		}

		if totalBytesProcessed != nil {
			updates = append(
				updates,
				firestore.Update{
					FieldPath: []string{statsField, origin, totalBytesProcessedField},
					Value:     totalBytesProcessed,
				},
			)
		}

		if len(updates) > 0 {
			return tx.Update(
				reportRef,
				updates,
			)
		}

		return nil
	})
}
