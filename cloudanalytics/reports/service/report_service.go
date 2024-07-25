package service

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service"
	caownercheckerIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	domainExternalAPI "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	externalAPIServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/service/iface"
	postProcessingAggregationService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/aggregation/service"
	postProcessingAggregationServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/aggregation/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/iface"
	externalReportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	reportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	externalReportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport"
	externalReportServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport/iface"
	reportValidatorService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reportvalidator/iface"
	widgetIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	labelsDal "github.com/doitintl/hello/scheduled-tasks/labels/dal"
	labelsIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
)

type ReportService struct {
	loggerProvider                   logger.Provider
	conn                             *connection.Connection
	cloudAnalyticsService            cloudanalytics.CloudAnalytics
	externalReportService            externalReportServiceIface.IExternalReportService
	externalAPIService               externalAPIServiceIface.IExternalAPIService
	reportValidatorService           reportValidatorService.IReportValidatorService
	widgetService                    widgetIface.WidgetService
	postProcessingAggregationService postProcessingAggregationServiceIface.AggregationService
	reportDAL                        iface.Reports
	customerDAL                      customerDAL.Customers
	caOwnerChecker                   caownercheckerIface.CheckCAOwnerInterface
	collab                           collab.Icollab
	labelsDal                        labelsIface.Labels
	employeeService                  doitemployees.ServiceInterface
}

func NewReportService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	cloudAnalyticsService cloudanalytics.CloudAnalytics,
	externalReportService externalReportServiceIface.IExternalReportService,
	externalAPIService externalAPIServiceIface.IExternalAPIService,
	reportValidatorService reportValidatorService.IReportValidatorService,
	widgetService widgetIface.WidgetService,
	postProcessingAggregationService *postProcessingAggregationService.AggregationService,
	reportDAL iface.Reports,
	customerDAL customerDAL.Customers,
) (*ReportService, error) {
	return &ReportService{
		loggerProvider,
		conn,
		cloudAnalyticsService,
		externalReportService,
		externalAPIService,
		reportValidatorService,
		widgetService,
		postProcessingAggregationService,
		reportDAL,
		customerDAL,
		service.NewCAOwnerChecker(conn),
		&collab.Collab{},
		labelsDal.NewLabelsFirestoreWithClient(conn.Firestore),
		doitemployees.NewService(conn),
	}, nil
}

func (s *ReportService) CreateReportWithExternal(
	ctx context.Context,
	externalReportPayload *externalReportDomain.ExternalReport,
	customerID string,
	email string,
) (*externalReportDomain.ExternalReport, []errormsg.ErrorMsg, error) {
	l := s.loggerProvider(ctx)

	reportPayload, individualFieldsReportValidationErrors, err := s.externalReportService.UpdateReportWithExternalReport(
		ctx,
		customerID,
		reportDomain.NewDefaultReport(),
		externalReportPayload,
	)
	if err != nil {
		return nil, individualFieldsReportValidationErrors, err
	}

	fullReportValidationErrors, err := s.reportValidatorService.Validate(ctx, reportPayload)
	if err != nil {
		return nil, fullReportValidationErrors, err
	}

	customer, err := s.customerDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, nil, err
	}

	customerRef := customer.Snapshot.Ref

	reportPayload.Customer = customerRef
	reportPayload.AddCollaborator(email, collab.CollaboratorRoleOwner)

	l.Infof("creating report: %+v", reportPayload)

	report, err := s.reportDAL.Create(ctx, nil, reportPayload)
	if err != nil {
		return nil, nil, err
	}

	externalReport, externalReportValidationErrors, err := s.externalReportService.NewExternalReportFromInternal(
		ctx,
		customerID,
		report,
	)
	if err != nil {
		if errors.Is(err, externalReportService.ErrValidation) {
			l.Errorf("%s. Report id: %v. Validation errors: %v", ErrExternalFromInternalValidationErrorsMsg, report.ID, externalReportValidationErrors)
			return nil, nil, ErrInternalToExternal
		}

		return nil, nil, err
	}

	return externalReport, nil, nil
}

func (s *ReportService) GetReportConfig(
	ctx context.Context,
	reportID string,
	customerID string,
) (*externalReportDomain.ExternalReport, error) {
	l := s.loggerProvider(ctx)

	report, err := s.reportDAL.Get(ctx, reportID)
	if err != nil {
		return nil, err
	}

	if report.Type == reportDomain.ReportTypeCustom && customerID != report.Customer.ID {
		return nil, ErrInvalidCustomerID
	}

	externalReport, externalReportValidationErrors, err := s.externalReportService.NewExternalReportFromInternal(
		ctx,
		customerID,
		report,
	)
	if err != nil {
		if errors.Is(err, externalReportService.ErrValidation) {
			l.Errorf(
				"%s. Report id: %v. Validation errors: %v",
				ErrExternalFromInternalValidationErrorsMsg,
				report.ID,
				externalReportValidationErrors,
			)

			return nil, ErrInternalToExternal
		}

		return nil, err
	}

	return externalReport, nil
}

func (s *ReportService) UpdateReportWithExternal(
	ctx context.Context,
	reportID string,
	externalReportPayload *externalReportDomain.ExternalReport,
	customerID string,
	email string,
) (*externalReportDomain.ExternalReport, []errormsg.ErrorMsg, error) {
	l := s.loggerProvider(ctx)

	existingReport, err := s.reportDAL.Get(ctx, reportID)
	if err != nil {
		return nil, nil, err
	}

	if existingReport.Type != string(domainAttributions.ObjectTypeCustom) {
		return nil, nil, ErrInvalidReportType
	}

	if customerID != existingReport.Customer.ID {
		return nil, nil, ErrInvalidCustomerID
	}

	var isOwner bool

	for _, collaborator := range existingReport.Collaborators {
		if collaborator.Email == email {
			isOwner = collaborator.Role == collab.CollaboratorRoleOwner
			break
		}
	}

	if !isOwner {
		return nil, nil, ErrUnauthorizedDelete
	}

	if externalReportPayload.Name == "" {
		externalReportPayload.Name = existingReport.Name
	}

	report, individualFieldsReportValidationErrors, err := s.externalReportService.UpdateReportWithExternalReport(
		ctx,
		customerID,
		existingReport,
		externalReportPayload,
	)
	if err != nil {
		return nil, individualFieldsReportValidationErrors, err
	}

	fullReportValidationErrors, err := s.reportValidatorService.Validate(ctx, report)
	if err != nil {
		return nil, fullReportValidationErrors, err
	}

	l.Infof("updating report: $s,  %+v", reportID, report)

	err = s.reportDAL.Update(ctx, reportID, report)
	if err != nil {
		return nil, nil, err
	}

	externalReport, externalReportValidationErrors, err := s.externalReportService.NewExternalReportFromInternal(
		ctx,
		customerID,
		report,
	)
	if err != nil {
		if errors.Is(err, externalReportService.ErrValidation) {
			l.Errorf("%s. Report id: %v . Validation errors: %v", ErrExternalFromInternalValidationErrorsMsg, report.ID, externalReportValidationErrors)
			return nil, nil, ErrInternalToExternal
		}

		return nil, nil, err
	}

	return externalReport, nil, nil
}

func (s *ReportService) DeleteReport(
	ctx context.Context,
	customerID string,
	requesterEmail string,
	reportID string,
) error {
	report, err := s.reportDAL.Get(ctx, reportID)
	if err != nil {
		return err
	}

	if report.Type != string(domainAttributions.ObjectTypeCustom) {
		return ErrInvalidReportType
	}

	if customerID != report.Customer.ID {
		return ErrInvalidCustomerID
	}

	var isOwner bool

	for _, collaborator := range report.Collaborators {
		if collaborator.Email == requesterEmail {
			isOwner = collaborator.Role == collab.CollaboratorRoleOwner
			break
		}
	}

	if !isOwner {
		return ErrUnauthorizedDelete
	}

	if err := s.reportDAL.Delete(ctx, reportID); err != nil {
		return err
	}

	if err := s.widgetService.DeleteReportWidget(ctx, customerID, reportID); err != nil {
		return err
	}

	return nil
}

func (s *ReportService) DeleteMany(
	ctx context.Context,
	customerID string,
	email string,
	reportIDs []string,
) error {
	reports := make([]*reportDomain.Report, 0, len(reportIDs))

	for _, id := range reportIDs {
		r, err := s.reportDAL.Get(ctx, id)
		if err != nil {
			return err
		}

		reports = append(reports, r)
	}

	for _, report := range reports {
		if report.Type != string(domainAttributions.ObjectTypeCustom) {
			return ErrInvalidReportType
		}

		if !report.IsOwner(email) {
			return ErrUnauthorizedDelete
		}

		if customerID != report.Customer.ID {
			return ErrInvalidCustomerID
		}
	}

	reportRefs := make([]*firestore.DocumentRef, 0, len(reportIDs))
	for _, id := range reportIDs {
		reportRefs = append(reportRefs, s.reportDAL.GetRef(ctx, id))
	}

	if err := s.labelsDal.DeleteManyObjectsWithLabels(ctx, reportRefs); err != nil {
		return err
	}

	if err := s.widgetService.DeleteReportsWidgets(
		ctx,
		customerID,
		reportIDs,
	); err != nil {
		return err
	}

	return nil
}

func (s *ReportService) RunReportFromExternalConfig(
	ctx context.Context,
	externalConfig *externalReportDomain.ExternalConfig,
	customerID string,
	requesterEmail string,
) (*domainExternalAPI.RunReportResult, []errormsg.ErrorMsg, error) {
	config, configValidationErrors, err := s.externalReportService.MergeConfigWithExternalConfig(
		ctx,
		customerID,
		reportDomain.NewConfig(),
		externalConfig)
	if err != nil {
		return nil, configValidationErrors, err
	}

	reportPayload := reportDomain.NewDefaultReport()
	reportPayload.Config = config

	fullReportValidationErrors, err := s.reportValidatorService.Validate(ctx, reportPayload)
	if err != nil {
		return nil, fullReportValidationErrors, err
	}

	qr, err := s.cloudAnalyticsService.NewQueryRequestFromFirestoreReport(ctx, customerID, reportPayload)
	if err != nil {
		return nil, nil, err
	}

	result, err := s.cloudAnalyticsService.GetQueryResult(ctx, qr, customerID, requesterEmail)
	if err != nil {
		return nil, nil, err
	}

	numRows := len(qr.Rows)
	numCols := len(qr.Cols)

	queryResult := s.externalAPIService.ProcessResult(qr, reportPayload, result)

	err = s.postProcessingAggregationService.ApplyAggregation(config.Aggregator, numRows, numCols, queryResult.Rows)
	if err != nil {
		return nil, nil, err
	}

	return &queryResult, nil, nil
}

func (s *ReportService) ShareReport(ctx context.Context, args reportDomain.ShareReportArgsReq) error {
	l := s.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		logger.LabelEmail:      args.RequesterEmail,
		logger.LabelCustomerID: args.CustomerID,
		"reportId":             args.ReportID,
	})

	isCAOwner, err := s.caOwnerChecker.CheckCAOwner(ctx, s.employeeService, args.UserID, args.RequesterEmail)
	if err != nil {
		return err
	}

	report, err := s.reportDAL.Get(ctx, args.ReportID)
	if err != nil {
		return err
	}

	if err := s.collab.ShareAnalyticsResource(ctx, report.Collaborators, args.Access.Collaborators, args.Access.Public, args.ReportID, args.RequesterEmail, s.reportDAL, isCAOwner); err != nil {
		return err
	}

	if common.Production {
		notifyNewlyAddedCollaborators(report, args, l)
	}

	return nil
}

// notify newly added collaborators which didn't exist before and existing collaborator if he stopped being an owner but got another role
func notifyNewlyAddedCollaborators(report *reportDomain.Report, args reportDomain.ShareReportArgsReq, l logger.ILogger) {
	var collaboratorsToNotify []collab.Collaborator

	for _, newCollaborator := range args.Access.Collaborators {
		ifNotify := true

		for _, oldCollaborator := range report.Collaborators {
			if newCollaborator.Email == oldCollaborator.Email {
				isOwnershipTransferred := oldCollaborator.Role == collab.CollaboratorRoleOwner && newCollaborator.Role != collab.CollaboratorRoleOwner

				if !isOwnershipTransferred {
					ifNotify = false
				}

				break
			}
		}

		if ifNotify {
			collaboratorsToNotify = append(collaboratorsToNotify, newCollaborator)
		}
	}

	if len(collaboratorsToNotify) == 0 {
		return
	}

	sendEmailsToCollaborators(report, collaboratorsToNotify, args, l)
}

func sendEmailsToCollaborators(report *reportDomain.Report, collaborators []collab.Collaborator, args reportDomain.ShareReportArgsReq, l logger.ILogger) {
	personalizations := make([]*mail.Personalization, 0)

	for _, collaborator := range collaborators {
		p := mail.NewPersonalization()
		p.AddTos(mail.NewEmail("", collaborator.Email))
		p.SetDynamicTemplateData("email", args.RequesterEmail)
		p.SetDynamicTemplateData("name", args.RequesterName)
		p.SetDynamicTemplateData("report_name", report.Name)
		p.SetDynamicTemplateData("customer_id", args.CustomerID)
		p.SetDynamicTemplateData("report_id", report.ID)
		p.SetDynamicTemplateData("role", collaborator.Role[:len(collaborator.Role)-2])
		p.SetDynamicTemplateData("is_owner", collaborator.Role == collab.CollaboratorRoleOwner)
		personalizations = append(personalizations, p)
	}

	if err := mailer.SendEmailWithPersonalizations(personalizations, mailer.Config.DynamicTemplates.CloudReportShare, []string{mailer.CatagoryReports}); err != nil {
		l.Error("Error sending mail", err)
	}
}
