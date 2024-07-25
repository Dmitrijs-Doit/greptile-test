package schedule

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domainHighCharts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
)

type SendReportRequest struct {
	ReportID   string `json:"reportId"`
	CustomerID string `json:"customerId"`
}

type scheduledReportExecution struct {
	State     executionState `firestore:"state"`
	Timestamp time.Time      `firestore:"timestamp,serverTimestamp"`
}

type executionState int

const (
	executionStatePending executionState = iota
	executionStateInProgress
	executionStateSuccess
)

func (s *ScheduledReportsService) checkExecutionState(ctx context.Context, executionRef *firestore.DocumentRef) (bool, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	var ok bool

	err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		ok = false

		docSnap, err := tx.Get(executionRef)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}

		var exec scheduledReportExecution

		if docSnap.Exists() {
			if err := docSnap.DataTo(&exec); err != nil {
				return err
			}

			timeSince := time.Since(exec.Timestamp)
			l.Debugf("%#v", exec)
			l.Debugf("since timestamp: %v", timeSince)

			// If it has been more than 30 minutes since the last update
			// then allow to proceed
			if timeSince >= time.Duration(time.Minute*30) {
				exec.State = executionStatePending
			}
		}

		// If already in progress or completed successfuly then stop execution
		if exec.State == executionStateInProgress || exec.State == executionStateSuccess {
			return nil
		}

		ok = true

		return tx.Set(executionRef, scheduledReportExecution{
			State: executionStateInProgress,
		})
	})
	if err != nil {
		return false, err
	}

	l.Debugf("lock ok: %v", ok)

	return ok, nil
}

func (s *ScheduledReportsService) updateExecutionState(ctx context.Context, executionRef *firestore.DocumentRef, state *executionState) error {
	_, err := executionRef.Set(ctx, scheduledReportExecution{
		State: *state,
	})

	return err
}

// SendReport - main function that is trigered by cloud scheduler job
func (s *ScheduledReportsService) SendReport(ctx context.Context, reportReq *SendReportRequest) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	executionID := fmt.Sprintf("%s_%s", reportReq.CustomerID, reportReq.ReportID)
	executionRef := fs.Collection("cloudAnalytics").Doc("reports").Collection("cloudAnalyticsScheduledReportsExecutions").Doc(executionID)

	reportID := reportReq.ReportID

	report, err := s.cloudAnalytics.GetReport(ctx, reportReq.CustomerID, reportID, false)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			l.Infof("report %s was not found - deleting cloud scheduler job", reportID)

			if err := s.deleteJob(ctx, reportID); err != nil {
				return err
			}

			l.Infof("report %s scheduled job deleted successfully", reportID)

			return nil
		}

		return err
	}

	if report.Schedule == nil {
		return ErrScheduleNotFound
	}

	err = s.reportDAL.UpdateTimeLastRun(ctx, reportID, domainOrigin.QueryOriginScheduledReports)
	if err != nil {
		l.Errorf("failed to update last time run for report %v; %s", reportID, err)
	}

	ok, err := s.checkExecutionState(ctx, executionRef)
	if err != nil || !ok {
		return err
	}

	newState := executionStatePending

	defer func() {
		err := s.updateExecutionState(ctx, executionRef, &newState)
		if err != nil {
			l.Errorf("failed to unlock with error %s", err)
		}
	}()

	imageURL, err := s.highCharts.GetReportImage(ctx, reportReq.ReportID, reportReq.CustomerID, &domainHighCharts.SendReportFontSettings)
	if err != nil {
		l.Errorf("image url not retrieved properly error: %s", err)

		if status.Code(err) == codes.NotFound {
			if err := s.deleteJob(ctx, reportReq.ReportID); err != nil {
				l.Errorf("deleted job for report %s with error: %s", reportID, err)
				return err
			}
		}

		return err
	}

	if imageURL == "" {
		return ErrImageEmpty
	}

	if err := s.sendReport(ctx, reportReq.CustomerID, reportReq.ReportID, report, imageURL); err != nil {
		return err
	}

	newState = executionStateSuccess

	return nil
}

func (s *ScheduledReportsService) sendReport(ctx context.Context, customerID, reportID string, r *domainReport.Report, imageURL string) error {
	l := s.loggerProvider(ctx)

	m := mail.NewV3Mail()
	m.SetFrom(mail.NewEmail(mailer.Config.NoReplyName, mailer.Config.NoReplyEmail))

	enable := false
	m.SetTrackingSettings(&mail.TrackingSettings{SubscriptionTracking: &mail.SubscriptionTrackingSetting{Enable: &enable}})
	m.SetTemplateID(mailer.Config.DynamicTemplates.ScheduledCloudReport)
	m.AddCategories([]string{mailer.CatagoryReports, mailer.CatagoryScheduledReports}...)

	personalizations := make([]*mail.Personalization, 0)

	filteredEmails, err := s.validateRecipientsOrganization(ctx, r)
	if err != nil {
		return err
	}

	for _, recipient := range filteredEmails {
		if !common.Production && !common.IsDoitDomain(recipient) {
			l.Info("mail to <" + recipient + "> didn't send while in development")
			continue
		}

		p := mail.NewPersonalization()
		p.AddTos(mail.NewEmail("", recipient))
		p.SetDynamicTemplateData("subject", r.Schedule.Subject)
		p.SetDynamicTemplateData("from", r.Schedule.From)
		p.SetDynamicTemplateData("body", strings.Replace(r.Schedule.Body, "\n", "<br>", -1))
		p.SetDynamicTemplateData("customer_id", customerID)
		p.SetDynamicTemplateData("report_id", reportID)
		p.SetDynamicTemplateData("image_url", imageURL)
		p.SetDynamicTemplateData("domain", common.Domain)
		p.SetDynamicTemplateData("timestamp", time.Now().Format(time.RFC3339))
		personalizations = append(personalizations, p)
	}

	if len(personalizations) == 0 {
		l.Info("personalizations slice is empty; will not send report.")
		return nil
	}

	m.AddPersonalizations(personalizations...)

	request := sendgrid.GetRequest(mailer.Config.APIKey, mailer.Config.MailSendPath, mailer.Config.BaseURL)
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)

	if _, err := sendgrid.MakeRequestRetry(request); err != nil {
		return err
	}

	l.Info("Mail sent successfully!")

	return nil
}

// Validate that recipients are from report organization (in case of change after jobs scheduled)
func (s *ScheduledReportsService) validateRecipientsOrganization(ctx context.Context, report *report.Report) ([]string, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	collections := []string{"users", "invites"}

	var allowedEmails []string

	var reportOrgID string
	if report.Organization != nil {
		reportOrgID = report.Organization.ID
	}

	for _, recipient := range report.Schedule.To {
		if common.IsDoitDomain(recipient) {
			allowedEmails = append(allowedEmails, recipient)
			continue
		}

		for _, collection := range collections {
			docSnaps, err := fs.Collection(collection).Where("email", "==", recipient).Limit(1).Select("organizations").Documents(ctx).GetAll()
			if err != nil {
				return nil, err
			}

			if len(docSnaps) == 0 {
				continue
			}
			// Validate user is member of reports organization
			var user common.User
			if err := docSnaps[0].DataTo(&user); err != nil {
				return nil, err
			}

			if user.ValidateOrganization(reportOrgID) {
				allowedEmails = append(allowedEmails, recipient)
			} else {
				l.Infof("mail to <%s> didn't send they are not member of organization", recipient)
			}

			break
		}
	}

	return allowedEmails, nil
}
