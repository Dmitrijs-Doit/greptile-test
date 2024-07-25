//go:generate mockery --name ReportTemplateFirestore --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
)

type ReportTemplateFirestore interface {
	Get(
		ctx context.Context,
		tx *firestore.Transaction,
		reportTemplateID string,
	) (*domain.ReportTemplate, error)
	CreateReportTemplate(
		ctx context.Context,
		tx *firestore.Transaction,
		reportTemplate *domain.ReportTemplate,
	) (*firestore.DocumentRef, error)
	CreateVersion(
		ctx context.Context,
		tx *firestore.Transaction,
		reportTemplateVersionID string,
		reportTemplateID string,
		reportTemplateVersion *domain.ReportTemplateVersion,
	) (*firestore.DocumentRef, error)
	GetVersionByRef(
		ctx context.Context,
		reportTemplateVersionRef *firestore.DocumentRef,
	) (*domain.ReportTemplateVersion, error)
	RunTransaction(ctx context.Context, f dal.TransactionFunc) (interface{}, error)
	UpdateReportTemplate(
		ctx context.Context,
		tx *firestore.Transaction,
		reportTemplateID string,
		reportTemplate *domain.ReportTemplate,
	) error
	UpdateReportTemplateVersion(
		ctx context.Context,
		tx *firestore.Transaction,
		reportTemplateVersion *domain.ReportTemplateVersion,
	) error
	DeleteReportTemplate(ctx context.Context, reportTemplateID string) error
	HideReportTemplate(ctx context.Context, reportTemplateID string) error
	GetTemplates(ctx context.Context) ([]domain.ReportTemplate, error)
	GetVersions(ctx context.Context, versionRefs []*firestore.DocumentRef) ([]domain.ReportTemplateVersion, error)
}
