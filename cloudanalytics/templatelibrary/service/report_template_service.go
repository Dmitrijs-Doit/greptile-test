package service

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"

	attributionGroupsDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionsDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	reportDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/iface"
	reportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	reportValidatorIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reportvalidator/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	notificationcenter "github.com/doitintl/notificationcenter/pkg"
)

type ReportTemplateService struct {
	loggerProvider         logger.Provider
	conn                   *connection.Connection
	employeeService        doitemployees.ServiceInterface
	reportValidatorService reportValidatorIface.IReportValidatorService
	reportTemplateDAL      iface.ReportTemplateFirestore
	attributionDAL         attributionsDalIface.Attributions
	attributionGroupDAL    attributionGroupsDalIface.AttributionGroups
	reportDAL              reportDalIface.Reports
	notificationClient     notificationcenter.NotificationSender
}

func NewReportTemplateService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	reportValidatorService reportValidatorIface.IReportValidatorService,
	attributionDAL attributionsDalIface.Attributions,
	attributionGroupDAL attributionGroupsDalIface.AttributionGroups,
	reportDAL reportDalIface.Reports,
	notificationClient notificationcenter.NotificationSender,
) (*ReportTemplateService, error) {
	reportTemplateDAL := dal.NewReportTemplateFirestoreWithClient(conn.Firestore)

	return &ReportTemplateService{
		loggerProvider,
		conn,
		doitemployees.NewService(conn),
		reportValidatorService,
		reportTemplateDAL,
		attributionDAL,
		attributionGroupDAL,
		reportDAL,
		notificationClient,
	}, nil
}

func (s *ReportTemplateService) validateAttributionsAndAGs(ctx context.Context, reportConfig *reportDomain.Config) error {
	var attributionsIDs []string

	var attributionsRefs []*firestore.DocumentRef

	var attributionGroupsRefs []*firestore.DocumentRef

	for _, filter := range reportConfig.Filters {
		if filter.ID == "attribution:attribution" {
			attributionsIDs = *filter.Values
		} else if strings.Contains(filter.ID, string(metadataDomain.MetadataFieldTypeAttributionGroup)) {
			attributionGroupID := strings.Split(filter.ID, ":")[1]
			attributionGroupsRefs = append(attributionGroupsRefs, s.attributionGroupDAL.GetRef(ctx, attributionGroupID))
		}
	}

	for _, attributionID := range attributionsIDs {
		attributionsRefs = append(attributionsRefs, s.attributionDAL.GetRef(ctx, attributionID))
	}

	for _, val := range append(reportConfig.Rows, reportConfig.Cols...) {
		if strings.Contains(val, string(metadataDomain.MetadataFieldTypeAttributionGroup)) {
			attributionGroupID := strings.Split(val, ":")[1]
			attributionGroupsRefs = append(attributionGroupsRefs, s.attributionGroupDAL.GetRef(ctx, attributionGroupID))
		}
	}

	// check all attributions are presets
	var attributions []*attribution.Attribution

	var err error

	if len(attributionsRefs) > 0 {
		attributions, err = s.attributionDAL.GetAttributions(ctx, attributionsRefs)
		if err != nil {
			return err
		}
	}

	for _, attr := range attributions {
		if attr.Type != string(attribution.ObjectTypePreset) {
			return domain.ValidationErr{Name: attr.Name, Type: domain.CustomAttributionErrType}
		}
	}

	var attributionGroups []*attributiongroups.AttributionGroup

	// check all attributionGroups are presets
	if len(attributionGroupsRefs) > 0 {
		attributionGroups, err = s.attributionGroupDAL.GetAll(ctx, attributionGroupsRefs)
		if err != nil {
			return err
		}
	}

	for _, attributionGroup := range attributionGroups {
		if attributionGroup.Type != attribution.ObjectTypePreset {
			return domain.ValidationErr{Name: attributionGroup.Name, Type: domain.CustomAGErrType}
		}
	}

	return nil
}

func (s *ReportTemplateService) validateManagedReport(
	ctx context.Context,
	report *reportDomain.Report,
) ([]errormsg.ErrorMsg, error) {
	reportConfig := report.Config

	if reportConfig == nil {
		return nil, domain.ErrNoReportTemplateConfig
	}

	fullReportValidationErrors, err := s.reportValidatorService.Validate(
		ctx, report)
	if err != nil {
		return fullReportValidationErrors, domain.ErrInvalidReportTemplateConfig
	}

	if reportConfig.CalculatedMetric != nil {
		return nil, domain.ErrCustomMetric
	}

	err = s.validateAttributionsAndAGs(ctx, reportConfig)
	if err != nil {
		return nil, err
	}

	for _, val := range reportConfig.Optional {
		if val.Type != metadataDomain.MetadataFieldTypeSystemLabel {
			return []errormsg.ErrorMsg{
				{
					Field:   reportDomain.ConfigOptionalField,
					Message: fmt.Sprintf(ErrInvalidOptionalTpl, val.Key),
				},
			}, domain.ErrCustomLabel
		}
	}

	return nil, nil
}

func isAutoApprove(
	visibility domain.Visibility,
	isTemplateLibraryAdmin bool,
) bool {
	var autoApprove bool

	if visibility == domain.VisibilityPrivate {
		autoApprove = true
	}

	if isTemplateLibraryAdmin {
		autoApprove = true
	}

	return autoApprove
}

func validateVisibility(visibility domain.Visibility) []errormsg.ErrorMsg {
	switch visibility {
	case domain.VisibilityInternal, domain.VisibilityPrivate, domain.VisibilityGlobal:
		return nil
	}

	return []errormsg.ErrorMsg{
		{
			Field:   domain.TemplateVisibilityField,
			Message: fmt.Sprintf(domain.ErrInvalidReportTemplateVisibilityTpl, visibility),
		},
	}
}

func getReportTemplateUrl(
	reportTemplateID string,
	reportID string,
) string {
	const taboolaCustomerID = "ImoC9XkrutBysJvyqlBm"

	return fmt.Sprintf(
		"https://%s/customers/%s/analytics/reports/%s?templateId=%s&edit=true",
		common.Domain,
		taboolaCustomerID,
		reportID,
		reportTemplateID,
	)
}

func (s *ReportTemplateService) getReportTemplateWithLastVersion(ctx context.Context, reportTemplate *domain.ReportTemplate) (*domain.ReportTemplateWithVersion, error) {
	if reportTemplate == nil {
		return nil, domain.ErrInvalidReportTemplate
	}

	lastVersion, err := s.reportTemplateDAL.GetVersionByRef(ctx, reportTemplate.LastVersion)
	if err != nil {
		return nil, err
	}

	reportTemplate.SetPath()
	lastVersion.SetPath()

	reportTemplateWithVersion := &domain.ReportTemplateWithVersion{
		Template:    reportTemplate,
		LastVersion: lastVersion,
	}

	return reportTemplateWithVersion, nil
}
