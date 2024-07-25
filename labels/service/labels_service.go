package service

import (
	"context"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	doitFirestoreIface "github.com/doitintl/firestore/iface"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/labels/dal"
	"github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
	labels "github.com/doitintl/hello/scheduled-tasks/labels/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"

	alertsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal"
	alertsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal/iface"
	attributionGroupsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal"
	attributionGroupsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/iface"
	attributionsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	attributionsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/iface"
	budgetsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal"
	metricsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal"
	metricsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal/iface"
	reportsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	reportsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/iface"
)

type LabelsService struct {
	loggerProvider       logger.Provider
	conn                 *connection.Connection
	labelsDal            iface.Labels
	customerDal          customerDal.Customers
	batchProvider        doitFirestoreIface.BatchProvider
	attributionsDal      attributionsIface.Attributions
	alertsDal            alertsIface.Alerts
	attributionGroupsDal attributionGroupsIface.AttributionGroups
	budgetsDal           budgetsDal.Budgets
	metricsDal           metricsIface.Metrics
	reportsDal           reportsIface.Reports
}

func NewLabelsService(log logger.Provider, conn *connection.Connection) *LabelsService {
	return &LabelsService{
		log,
		conn,
		dal.NewLabelsFirestoreWithClient(conn.Firestore),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		doitFirestore.NewBatchProvider(conn.Firestore, 500),
		attributionsDal.NewAttributionsFirestoreWithClient(conn.Firestore),
		alertsDal.NewAlertsFirestoreWithClient(conn.Firestore),
		attributionGroupsDal.NewAttributionGroupsFirestoreWithClient(conn.Firestore),
		budgetsDal.NewBudgetsFirestoreWithClient(conn.Firestore),
		metricsDal.NewMetricsFirestoreWithClient(conn.Firestore),
		reportsDal.NewReportsFirestoreWithClient(conn.Firestore),
	}
}

func (s *LabelsService) CreateLabel(ctx context.Context, req CreateLabelRequest) (*labels.Label, error) {
	// Get customer ref
	customer, err := s.customerDal.GetCustomer(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	return s.labelsDal.Create(ctx, &labels.Label{
		Name:      req.Name,
		Color:     req.Color,
		CreatedBy: req.UserEmail,
		Customer:  customer.Snapshot.Ref,
	})
}

func (s *LabelsService) UpdateLabel(ctx context.Context, req UpdateLabelRequest) (*labels.Label, error) {
	updates := getLabelUpdates(req)

	return s.labelsDal.Update(ctx, req.LabelID, updates)
}

func (s *LabelsService) DeleteLabel(ctx context.Context, labelID string) error {
	// check label exists
	label, err := s.labelsDal.Get(ctx, labelID)
	if err != nil {
		return err
	}

	wb := s.batchProvider.ProvideWithThreshold(ctx, len(label.Objects)+1)

	if err := wb.Delete(ctx, label.Ref); err != nil {
		return err
	}

	for _, o := range label.Objects {
		var newLabels []*firestore.DocumentRef

		oLabelsRefs, err := s.labelsDal.GetObjectLabels(ctx, o)
		if err != nil {
			return err
		}

		for _, l := range oLabelsRefs {
			if l.ID != labelID {
				newLabels = append(newLabels, l)
			}
		}

		if err := wb.Update(ctx, o, []firestore.Update{{Path: "labels", Value: newLabels}}); err != nil {
			return err
		}
	}

	return wb.Commit(ctx)
}

func (s *LabelsService) AssignLabels(ctx context.Context, req AssignLabelsRequest) error {
	var addLabels []*labels.Label

	var removeLabels []*labels.Label

	if len(req.AddLabels) != 0 {
		l, err := s.labelsDal.GetLabels(ctx, req.AddLabels)
		if err != nil {
			return err
		}

		addLabels = l
	}

	if len(req.RemoveLabels) != 0 {
		l, err := s.labelsDal.GetLabels(ctx, req.RemoveLabels)
		if err != nil {
			return err
		}

		removeLabels = l
	}

	removeLabelsMap := sliceToMap(removeLabels, func(label *labels.Label) string {
		return label.Ref.ID
	})

	objectsMap := sliceToMap(req.Objects, func(o AssignLabelsObject) string { return o.ObjectID })

	wb := s.batchProvider.ProvideWithThreshold(ctx, len(req.Objects)+len(addLabels)+len(removeLabels))

	if err := s.updateObjectLabels(ctx, req, removeLabelsMap, addLabels, wb); err != nil {
		return err
	}

	if err := s.removeObjectRefFromRemovedLabels(ctx, removeLabels, objectsMap, wb); err != nil {
		return err
	}

	if err := s.addObjRefToAddedLabels(ctx, addLabels, req, wb); err != nil {
		return err
	}

	return wb.Commit(ctx)
}

func (s *LabelsService) addObjRefToAddedLabels(ctx context.Context, addLabels []*labels.Label, req AssignLabelsRequest, wb doitFirestoreIface.Batch) error {
	for _, label := range addLabels {
		labelObjectsMap := sliceToMap(label.Objects, func(o *firestore.DocumentRef) string { return o.ID })
		for _, o := range req.Objects {
			if _, exists := labelObjectsMap[o.ObjectID]; !exists {
				objRef, err := s.getObjectReference(ctx, o.ObjectID, o.ObjectType)
				if err != nil {
					return err
				}

				label.Objects = append(label.Objects, objRef)
			}
		}

		if err := wb.Update(ctx, label.Ref, []firestore.Update{{Path: "objects", Value: label.Objects}}); err != nil {
			return err
		}
	}

	return nil
}

func (*LabelsService) removeObjectRefFromRemovedLabels(ctx context.Context, removeLabels []*labels.Label, objectsMap map[string]AssignLabelsObject, wb doitFirestoreIface.Batch) error {
	for _, label := range removeLabels {
		var newObjects []*firestore.DocumentRef

		for _, o := range label.Objects {
			if _, exists := objectsMap[o.ID]; !exists {
				newObjects = append(newObjects, o)
			}
		}

		if err := wb.Update(ctx, label.Ref, []firestore.Update{{Path: "objects", Value: newObjects}}); err != nil {
			return err
		}
	}

	return nil
}

func (s *LabelsService) updateObjectLabels(ctx context.Context, req AssignLabelsRequest, removeLabelsMap map[string]*labels.Label, addLabels []*labels.Label, wb doitFirestoreIface.Batch) error {
	for _, o := range req.Objects {
		objRef, err := s.getObjectReference(ctx, o.ObjectID, o.ObjectType)
		if err != nil {
			return err
		}

		canEdit, err := s.checkObjectPermissions(ctx, o.ObjectID, o.ObjectType)
		if err != nil {
			return err
		}

		if !canEdit {
			return labels.ErrNoPermissions(o.ObjectType, o.ObjectID)
		}

		oldObjectLabels, err := s.labelsDal.GetObjectLabels(ctx, objRef)
		if err != nil {
			return err
		}

		var newLabels []*firestore.DocumentRef

		for _, label := range oldObjectLabels {
			if _, exists := removeLabelsMap[label.ID]; !exists {
				newLabels = append(newLabels, label)
			}
		}

		for _, al := range addLabels {
			shouldAddLabel := true

			for _, nl := range newLabels {
				if nl.ID == al.Ref.ID {
					shouldAddLabel = false
					break
				}
			}

			if shouldAddLabel {
				newLabels = append(newLabels, al.Ref)
			}
		}

		if err := wb.Update(ctx, objRef, []firestore.Update{{Path: "labels", Value: newLabels}}); err != nil {
			return err
		}
	}

	return nil
}
