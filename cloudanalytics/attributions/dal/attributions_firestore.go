package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/customerapi"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	labelsDal "github.com/doitintl/hello/scheduled-tasks/labels/dal"
)

const (
	AttributionsCollection = "dashboards/google-cloud-reports/attributions"
)

// NewAttributionsFirestore returns a new AttributionsFirestore instance with given project id.
func NewAttributionsFirestore(ctx context.Context, projectID string) (*AttributionsFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewAttributionsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewAttributionsFirestoreWithClient returns a new AttributionsFirestore using given client.
func NewAttributionsFirestoreWithClient(fun connection.FirestoreFromContextFun) *AttributionsFirestore {
	return &AttributionsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		labelsDal:          labelsDal.NewLabelsFirestoreWithClient(fun),
	}
}

func (d *AttributionsFirestore) GetRef(ctx context.Context, attributionID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(AttributionsCollection).Doc(attributionID)
}

// GetAttribution returns a cloud analytics attribution configuration.
func (d *AttributionsFirestore) GetAttribution(ctx context.Context, attributionID string) (*domainAttributions.Attribution, error) {
	if attributionID == "" {
		return nil, domainAttributions.ErrInvalidAttributionID
	}

	docRef := d.GetRef(ctx, attributionID)

	docSnap, err := d.documentsHandler.Get(ctx, docRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, domainAttributions.ErrNotFound
		}

		return nil, err
	}

	var attrb domainAttributions.Attribution

	if err := docSnap.DataTo(&attrb); err != nil {
		return nil, err
	}

	attrb.ID = docSnap.ID()
	attrb.Ref = docRef

	return &attrb, nil
}

func (d *AttributionsFirestore) GetAttributions(ctx context.Context, attributionsRefs []*firestore.DocumentRef) ([]*domainAttributions.Attribution, error) {
	if len(attributionsRefs) == 0 {
		return nil, domainAttributions.ErrEmptyAttributionRefsList
	}

	var queries []firestore.Query
	// we need to split the attributionsRefs in arrays of max len 30 to overcome a limitation using IN operator in firestore queries
	for i := 0; i < len(attributionsRefs); i += 30 {
		end := i + 30
		if end > len(attributionsRefs) {
			end = len(attributionsRefs)
		}

		queries = append(queries, d.firestoreClientFun(ctx).Collection(AttributionsCollection).Query.
			Where(firestore.DocumentID, "in", attributionsRefs[i:end]))
	}

	allDocs, err := firebase.ExecuteQueries(ctx, queries)

	if err != nil {
		return nil, err
	}

	var attrs []*domainAttributions.Attribution

	for _, attr := range allDocs {
		var a domainAttributions.Attribution
		if err := attr.DataTo(&a); err != nil {
			return nil, err
		}

		a.ID = attr.Ref.ID
		a.Ref = attr.Ref

		attrs = append(attrs, &a)
	}

	return attrs, nil
}

/*
ListAttributions returns a list of cloud analytics attributions for a customer.
Calling this function is a workaround for the fact that firestore does not support OR queries. It is assumed that no type filter is present
*/
func (d *AttributionsFirestore) ListAttributions(ctx context.Context, req *customerapi.Request, cRef *firestore.DocumentRef) ([]domainAttributions.Attribution, error) {
	queries := []firestore.Query{
		d.firestoreClientFun(ctx).Collection(AttributionsCollection).Query.
			Where("customer", "==", cRef).
			Where("type", "==", domainAttributions.ObjectTypeCustom).
			Where("draft", "==", false).
			Where("name", "!=", domainAttributions.DefaultAttributionName),

		d.firestoreClientFun(ctx).Collection(AttributionsCollection).Query.
			Where("customer", "==", nil).
			Where("type", "==", domainAttributions.ObjectTypePreset),
	}

	errChan := make(chan error)
	docChan := make(chan []iface.DocumentSnapshot)

	for _, query := range queries {
		go func(q firestore.Query) {
			customDocs, err := d.documentsHandler.GetAll(q.Documents(ctx))
			if err != nil {
				errChan <- err
			}

			docChan <- customDocs
		}(query)
	}

	var allDocs []iface.DocumentSnapshot

	for i := 0; i < 2; i++ {
		select {
		case docs := <-docChan:
			allDocs = append(allDocs, docs...)
		case err := <-errChan:
			return nil, err
		}
	}

	var attrs []domainAttributions.Attribution

	for _, attr := range allDocs {
		var a domainAttributions.Attribution
		if err := attr.DataTo(&a); err != nil {
			return nil, err
		}

		a.ID = attr.ID()
		a.Ref = attr.Snapshot().Ref

		if a.CanView(req.Email) {
			attrs = append(attrs, a)
		}
	}

	return attrs, nil
}

func (d *AttributionsFirestore) CreateAttribution(ctx context.Context, attribution *domainAttributions.Attribution) (*domainAttributions.Attribution, error) {
	attribution.TimeCreated = time.Now()
	attribution.TimeModified = time.Now()
	attributionPublic := collab.PublicAccessView
	attribution.Public = &attributionPublic

	if attribution.Type == "" {
		attribution.Type = "custom"
	}

	if attribution.Draft == nil {
		draft := false
		attribution.Draft = &draft
	}

	var ref *firestore.DocumentRef

	if attribution.ID != "" {
		ref = d.firestoreClientFun(ctx).Collection(AttributionsCollection).Doc(attribution.ID)
	} else {
		ref = d.firestoreClientFun(ctx).Collection(AttributionsCollection).NewDoc()
	}

	_, err := ref.Set(ctx, attribution)
	if err != nil {
		return nil, err
	}

	docSnap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	if err := docSnap.DataTo(&attribution); err != nil {
		return nil, err
	}

	attribution.ID = ref.ID
	attribution.Ref = ref

	return attribution, nil
}

func (d *AttributionsFirestore) UpdateAttribution(ctx context.Context, attributionID string, updates []firestore.Update) error {
	updates = append(updates, firestore.Update{
		Path:  "timeModified",
		Value: time.Now(),
	})

	docRef := d.GetRef(ctx, attributionID)

	if _, err := d.documentsHandler.Update(ctx, docRef, updates); err != nil {
		return err
	}

	return nil
}

func (d *AttributionsFirestore) DeleteAttribution(ctx context.Context, attributionID string) error {
	docRef := d.GetRef(ctx, attributionID)

	return d.labelsDal.DeleteObjectWithLabels(ctx, docRef)
}

func (d *AttributionsFirestore) CustomerHasCustomAttributions(ctx context.Context, customerRef *firestore.DocumentRef) (bool, error) {
	someRef, err := d.firestoreClientFun(ctx).Collection(AttributionsCollection).
		Where("customer", "==", customerRef).
		Where("type", "==", domainAttributions.ObjectTypeCustom).
		Where("draft", "==", false).
		Where("name", "!=", domainAttributions.DefaultAttributionName).
		Limit(1).
		Documents(ctx).GetAll()
	if err != nil {
		return false, err
	}

	return len(someRef) > 0, nil
}
