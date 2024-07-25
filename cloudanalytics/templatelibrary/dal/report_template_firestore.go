package dal

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	reportTemplatesCollection = "cloudAnalytics/template-library/templateLibraryReportTemplates"
	reportTemplateVersions    = "reportTemplateVersions"
	reportTemplateHidden      = "hidden"
)

type reportTemplateVersionField = string

const (
	reportTemplateVersionFieldActive          reportTemplateVersionField = "active"
	reportTemplateVersionFieldApproval        reportTemplateVersionField = "approval"
	reportTemplateVersionFieldCategories      reportTemplateVersionField = "categories"
	reportTemplateVersionFieldCloud           reportTemplateVersionField = "cloud"
	reportTemplateVersionFieldCollaborators   reportTemplateVersionField = "collaborators"
	reportTemplateVersionFieldPreviousVersion reportTemplateVersionField = "previousVersion"
	reportTemplateVersionFieldVisibility      reportTemplateVersionField = "visibility"
	reportTemplateVersionFieldTimeModified    reportTemplateVersionField = "timeModified"
)

func constructVersionsPath(reportTemplateID string) string {
	return fmt.Sprintf("%s/%s/%s", reportTemplatesCollection, reportTemplateID, reportTemplateVersions)
}

type ReportTemplateFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

type TransactionFunc func(context.Context, *firestore.Transaction) (interface{}, error)

func NewReportTemplateFirestore(ctx context.Context, projectID string) (*ReportTemplateFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewReportTemplateFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

func NewReportTemplateFirestoreWithClient(fun connection.FirestoreFromContextFun) *ReportTemplateFirestore {
	return &ReportTemplateFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *ReportTemplateFirestore) getRef(ctx context.Context, reportTemplateID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(reportTemplatesCollection).Doc(reportTemplateID)
}

func (d *ReportTemplateFirestore) Get(
	ctx context.Context,
	tx *firestore.Transaction,
	reportTemplateID string,
) (*domain.ReportTemplate, error) {
	if reportTemplateID == "" {
		return nil, domain.ErrInvalidReportTemplateID
	}

	docRef := d.getRef(ctx, reportTemplateID)

	var docSnap iface.DocumentSnapshot

	var docSnapTx *firestore.DocumentSnapshot

	var reportTemplate domain.ReportTemplate

	var err error

	if tx != nil {
		docSnapTx, err = tx.Get(docRef)
	} else {
		docSnap, err = d.documentsHandler.Get(ctx, docRef)
	}

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	if tx != nil {
		if err := docSnapTx.DataTo(&reportTemplate); err != nil {
			return nil, err
		}

		reportTemplate.ID = docSnapTx.Ref.ID
	} else {
		if err := docSnap.DataTo(&reportTemplate); err != nil {
			return nil, err
		}

		reportTemplate.ID = docSnap.ID()
	}

	reportTemplate.Ref = docRef

	return &reportTemplate, nil
}

func (d *ReportTemplateFirestore) CreateReportTemplate(
	ctx context.Context,
	tx *firestore.Transaction,
	reportTemplate *domain.ReportTemplate,
) (*firestore.DocumentRef, error) {
	if reportTemplate == nil {
		return nil, domain.ErrInvalidReportTemplate
	}

	docRef := d.firestoreClientFun(ctx).Collection(reportTemplatesCollection).NewDoc()

	if tx != nil {
		if err := tx.Create(docRef, reportTemplate); err != nil {
			return nil, err
		}
	} else {
		if _, err := d.documentsHandler.Create(ctx, docRef, reportTemplate); err != nil {
			return nil, err
		}
	}

	return docRef, nil
}

func (d *ReportTemplateFirestore) CreateVersion(
	ctx context.Context,
	tx *firestore.Transaction,
	reportTemplateVersionID string,
	reportTemplateID string,
	reportTemplateVersion *domain.ReportTemplateVersion,
) (*firestore.DocumentRef, error) {
	path := constructVersionsPath(reportTemplateID)

	docRef := d.firestoreClientFun(ctx).Collection(path).Doc(reportTemplateVersionID)

	if tx != nil {
		if err := tx.Create(docRef, reportTemplateVersion); err != nil {
			return nil, err
		}
	} else {
		if _, err := d.documentsHandler.Create(ctx, docRef, reportTemplateVersion); err != nil {
			return nil, err
		}
	}

	return docRef, nil
}

func (d *ReportTemplateFirestore) GetVersionByRef(
	ctx context.Context,
	reportTemplateVersionRef *firestore.DocumentRef,
) (*domain.ReportTemplateVersion, error) {
	docSnap, err := d.documentsHandler.Get(ctx, reportTemplateVersionRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var reportTemplateVersion domain.ReportTemplateVersion

	if err := docSnap.DataTo(&reportTemplateVersion); err != nil {
		return nil, err
	}

	reportTemplateVersion.ID = docSnap.ID()
	reportTemplateVersion.Ref = reportTemplateVersionRef

	return &reportTemplateVersion, nil
}

func (d *ReportTemplateFirestore) DeleteReportTemplate(ctx context.Context, reportTemplateID string) error {
	if reportTemplateID == "" {
		return domain.ErrInvalidReportTemplateID
	}

	docRef := d.getRef(ctx, reportTemplateID)

	bulkWriter := d.firestoreClientFun(ctx).BulkWriter(ctx)

	err := d.documentsHandler.DeleteDocAndSubCollections(ctx, docRef, bulkWriter)

	bulkWriter.End()

	return err
}

func (d *ReportTemplateFirestore) HideReportTemplate(ctx context.Context, reportTemplateID string) error {
	if reportTemplateID == "" {
		return domain.ErrInvalidReportTemplateID
	}

	docRef := d.getRef(ctx, reportTemplateID)

	update := []firestore.Update{
		{
			FieldPath: []string{reportTemplateHidden},
			Value:     true,
		},
	}

	_, err := d.documentsHandler.Update(ctx, docRef, update)

	return err
}

func (d *ReportTemplateFirestore) RunTransaction(ctx context.Context, f TransactionFunc) (interface{}, error) {
	fs := d.firestoreClientFun(ctx)

	var result interface{}

	err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var err error

		result, err = f(ctx, tx)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

func (d *ReportTemplateFirestore) UpdateReportTemplate(
	ctx context.Context,
	tx *firestore.Transaction,
	reportTemplateID string,
	reportTemplate *domain.ReportTemplate,
) error {
	docRef := d.getRef(ctx, reportTemplateID)

	update := []firestore.Update{
		{
			FieldPath: []string{"activeReport"},
			Value:     reportTemplate.ActiveReport,
		},
		{
			FieldPath: []string{"activeVersion"},
			Value:     reportTemplate.ActiveVersion,
		},
		{
			FieldPath: []string{"lastVersion"},
			Value:     reportTemplate.LastVersion,
		},
	}

	if tx != nil {
		if err := tx.Update(
			docRef,
			update,
		); err != nil {
			return err
		}
	} else {
		if _, err := d.documentsHandler.Update(
			ctx,
			docRef,
			update,
		); err != nil {
			return err
		}
	}

	return nil
}

func (d *ReportTemplateFirestore) UpdateReportTemplateVersion(
	ctx context.Context,
	tx *firestore.Transaction,
	reportTemplateVersion *domain.ReportTemplateVersion,
) error {
	update := []firestore.Update{
		{
			FieldPath: []string{reportTemplateVersionFieldActive},
			Value:     reportTemplateVersion.Active,
		},
		{
			FieldPath: []string{reportTemplateVersionFieldApproval},
			Value:     reportTemplateVersion.Approval,
		},
		{
			FieldPath: []string{reportTemplateVersionFieldCategories},
			Value:     reportTemplateVersion.Categories,
		},
		{
			FieldPath: []string{reportTemplateVersionFieldCloud},
			Value:     reportTemplateVersion.Cloud,
		},
		{
			FieldPath: []string{reportTemplateVersionFieldCollaborators},
			Value:     reportTemplateVersion.Collaborators,
		},
		{
			FieldPath: []string{reportTemplateVersionFieldPreviousVersion},
			Value:     reportTemplateVersion.PreviousVersion,
		},
		{
			FieldPath: []string{reportTemplateVersionFieldVisibility},
			Value:     reportTemplateVersion.Visibility,
		},
		{
			FieldPath: []string{reportTemplateVersionFieldTimeModified},
			Value:     firestore.ServerTimestamp,
		},
	}

	if tx != nil {
		if err := tx.Update(
			reportTemplateVersion.Ref,
			update,
		); err != nil {
			return err
		}
	} else {
		if _, err := d.documentsHandler.Set(
			ctx,
			reportTemplateVersion.Ref,
			update,
		); err != nil {
			return err
		}
	}

	return nil
}

func (d *ReportTemplateFirestore) GetTemplates(ctx context.Context) ([]domain.ReportTemplate, error) {
	query := d.firestoreClientFun(ctx).Collection(reportTemplatesCollection).Where("hidden", "==", false)

	docSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var reportTemplates []domain.ReportTemplate
	for _, docSnap := range docSnaps {
		var reportTemplate domain.ReportTemplate

		if err := docSnap.DataTo(&reportTemplate); err != nil {
			return nil, err
		}

		reportTemplate.ID = docSnap.Ref.ID
		reportTemplate.Ref = docSnap.Ref

		reportTemplates = append(reportTemplates, reportTemplate)
	}

	return reportTemplates, nil
}

func (d *ReportTemplateFirestore) GetVersions(ctx context.Context, versionRefs []*firestore.DocumentRef) ([]domain.ReportTemplateVersion, error) {
	var templateVersions []domain.ReportTemplateVersion

	fs := d.firestoreClientFun(ctx)

	docSnaps, err := fs.GetAll(ctx, versionRefs)
	if err != nil {
		return nil, err
	}

	for _, docSnap := range docSnaps {
		var templateVersion domain.ReportTemplateVersion

		if err := docSnap.DataTo(&templateVersion); err != nil {
			return nil, err
		}

		templateVersion.ID = docSnap.Ref.ID
		templateVersion.Ref = docSnap.Ref

		templateVersions = append(templateVersions, templateVersion)
	}

	return templateVersions, nil
}
