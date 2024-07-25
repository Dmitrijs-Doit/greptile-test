package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	doitFirestoreIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	labels "github.com/doitintl/hello/scheduled-tasks/labels/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LabelsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
	batchProvider      doitFirestoreIface.BatchProvider
}

const (
	labelsCollection = "labels"
)

func NewLabelsFirestore(ctx context.Context, projectID string) (*LabelsFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewLabelsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

func NewLabelsFirestoreWithClient(fun connection.FirestoreFromContextFun) *LabelsFirestore {
	return &LabelsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		batchProvider:      doitFirestore.NewBatchProvider(fun, 500),
	}
}

func (d *LabelsFirestore) GetRef(ctx context.Context, labelID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(labelsCollection).Doc(labelID)
}

func (d *LabelsFirestore) Get(ctx context.Context, labelID string) (*labels.Label, error) {
	if labelID == "" {
		return nil, labels.ErrInvalidLabelID
	}

	docRef := d.GetRef(ctx, labelID)

	docSnap, err := d.documentsHandler.Get(ctx, docRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, labels.ErrLabelNotFound(labelID)
		}

		return nil, err
	}

	var label labels.Label

	if err := docSnap.DataTo(&label); err != nil {
		return nil, err
	}

	label.Ref = docRef

	return &label, nil
}

func (d *LabelsFirestore) Create(ctx context.Context, label *labels.Label) (*labels.Label, error) {
	if label == nil {
		return nil, labels.ErrInvalidLabel
	}

	label.TimeCreated = time.Now()
	label.TimeModified = time.Now()

	ref, _, err := d.firestoreClientFun(ctx).Collection(labelsCollection).Add(ctx, label)
	if err != nil {
		return nil, err
	}

	docSnap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	var newLabel labels.Label
	if err := docSnap.DataTo(&newLabel); err != nil {
		return nil, err
	}

	return &newLabel, nil
}

func (d *LabelsFirestore) Update(ctx context.Context, labelID string, updates []firestore.Update) (*labels.Label, error) {
	updates = append(updates, firestore.Update{
		Path:  "timeModified",
		Value: time.Now(),
	})

	docRef := d.GetRef(ctx, labelID)

	if _, err := d.documentsHandler.Update(ctx, docRef, updates); err != nil {
		return nil, err
	}

	docSnap, err := docRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var updatedLabel labels.Label
	if err := docSnap.DataTo(&updatedLabel); err != nil {
		return nil, err
	}

	return &updatedLabel, nil
}

func (d *LabelsFirestore) GetLabels(ctx context.Context, labelsIDs []string) ([]*labels.Label, error) {
	if len(labelsIDs) == 0 {
		return nil, labels.ErrInvalidLabelID
	}

	labels := make([]*labels.Label, 0, len(labelsIDs))

	for _, l := range labelsIDs {
		label, err := d.Get(ctx, l)
		if err != nil {
			return nil, err
		}

		labels = append(labels, label)
	}

	return labels, nil
}

func (d *LabelsFirestore) GetObjectLabels(ctx context.Context, obj *firestore.DocumentRef) ([]*firestore.DocumentRef, error) {
	oSnap, err := obj.Get(ctx)
	if err != nil {
		return nil, err
	}

	var labelsRefs []*firestore.DocumentRef

	if labelsField, ok := oSnap.Data()["labels"]; ok {
		if labelsArray, isArray := labelsField.([]any); isArray {
			for _, label := range labelsArray {
				if labelRef, ok := label.(*firestore.DocumentRef); ok {
					labelsRefs = append(labelsRefs, labelRef)
				}
			}
		}
	}

	return labelsRefs, nil
}

func (d *LabelsFirestore) DeleteObjectWithLabels(ctx context.Context, deletedObjRef *firestore.DocumentRef) error {
	labelsRefs, err := d.GetObjectLabels(ctx, deletedObjRef)
	if err != nil {
		return err
	}

	labels := make([]*labels.Label, len(labelsRefs))

	for i, lRef := range labelsRefs {
		label, err := d.Get(ctx, lRef.ID)
		if err != nil {
			return err
		}

		labels[i] = label
	}

	wb := d.batchProvider.ProvideWithThreshold(ctx, len(labelsRefs)+1)

	for _, label := range labels {
		newLabelObjects := make([]*firestore.DocumentRef, 0, len(label.Objects)-1)

		for _, objRef := range label.Objects {
			if objRef.ID != deletedObjRef.ID {
				newLabelObjects = append(newLabelObjects, objRef)
			}
		}

		if err := wb.Update(ctx, label.Ref, []firestore.Update{{Path: "objects", Value: newLabelObjects}}); err != nil {
			return err
		}
	}

	if err := wb.Delete(ctx, deletedObjRef, firestore.Exists); err != nil {
		return err
	}

	return wb.Commit(ctx)
}

func (d *LabelsFirestore) DeleteManyObjectsWithLabels(ctx context.Context, objRefs []*firestore.DocumentRef) error {
	batchThreshold := len(objRefs)

	updatedLabelsMap := make(map[string]*labels.Label)
	deletedObjectsMaps := make(map[string]struct{})

	for _, objRef := range objRefs {
		labelsRefs, err := d.GetObjectLabels(ctx, objRef)
		if err != nil {
			return err
		}

		for _, lRef := range labelsRefs {
			label, err := d.Get(ctx, lRef.ID)
			if err != nil {
				return err
			}

			if _, exists := updatedLabelsMap[lRef.ID]; !exists {
				updatedLabelsMap[lRef.ID] = label
			}
		}

		deletedObjectsMaps[objRef.ID] = struct{}{}
	}

	for _, label := range updatedLabelsMap {
		newObjects := make([]*firestore.DocumentRef, 0, len(label.Objects))

		for _, objRef := range label.Objects {
			if _, exists := deletedObjectsMaps[objRef.ID]; !exists {
				newObjects = append(newObjects, objRef)
			}
		}

		batchThreshold += len(newObjects)
		updatedLabelsMap[label.Ref.ID].Objects = newObjects
	}

	wb := d.batchProvider.ProvideWithThreshold(ctx, batchThreshold)

	for _, objRef := range objRefs {
		if err := wb.Delete(ctx, objRef, firestore.Exists); err != nil {
			return err
		}
	}

	for _, label := range updatedLabelsMap {
		if err := wb.Update(ctx, label.Ref, []firestore.Update{{Path: "objects", Value: label.Objects}}); err != nil {
			return err
		}
	}

	return wb.Commit(ctx)
}
