package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/scheduler/apiv1/schedulerpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	durationpb "google.golang.org/protobuf/types/known/durationpb"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type Request struct {
	Schedule              *report.Schedule `json:"schedule"`
	CustomerPrimaryDomain string           `json:"customerPrimaryDomain"`
}

type RequestData struct {
	Email        string
	CustomerID   string
	ReportID     string
	UserID       string
	DoitEmployee bool
	Req          *Request
}

const (
	frequencyPattern string = `(?i)^([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9]) ([0-9]|1[0-9]|2[0-3]) (\*|([1-9]|[1-2][0-9]?|3[0-1])(-([1-9]|[1-2][0-9]?|3[0-1]))?)(\/[1-9][0-9]*)?(,(\*|([1-9]|[1-2][0-9]?|3[0-1])(-([1-9]|[1-2][0-9]?|3[0-1]))?)(\/[1-9][0-9]*)?)* (\*|([1-9]|1[0-2]|JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC?)(-([1-9]|1[0-2]|JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC?))?)(\/([1-9]|1[0-2]|JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC?)*)?(,(\*|([1-9]|1[0-2]|JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC?)(-([1-9]|1[0-2]|JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC?))?)(\/[1-9]|1[0-2]|JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC?)?)* (\*|[0-6]|MON|TUE|WED|THU|FRI|SAT|SUN(-([0-6]|MON|TUE|WED|THU|FRI|SAT|SUN))?)(\/([0-6]|MON|TUE|WED|THU|FRI|SAT|SUN))?(,(\*|[0-6]|MON|TUE|WED|THU|FRI|SAT|SUN(-[0-6]|MON|TUE|WED|THU|FRI|SAT|SUN)?)(\/[1-9][0-9]*)?)*$`
	htmlPattern      string = `(<\/?[a-zA-A]+?[^>]*\/?>)`
	jobsParentRoot   string = "projects/doitintl-cmp-reports-scheduler/locations/us-central1"
)

func getJobName(reportID string) string {
	if common.Production {
		return jobsParentRoot + "/jobs/" + reportID
	}

	return jobsParentRoot + "/jobs/dev_" + reportID
}

func (s *ScheduledReportsService) CreateSchedule(ctx context.Context, data *RequestData) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	if err := s.validateRequest(ctx, data, data.Req.Schedule.To, false); err != nil {
		return err
	}

	if _, err := s.createJob(ctx, data); err != nil {
		return err
	}

	reportRef := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("savedReports").Doc(data.ReportID)
	if _, err := reportRef.Update(ctx, []firestore.Update{{Path: "schedule", Value: data.Req.Schedule}}); err != nil {
		return err
	}

	l.Info("Schedule created successfully!")

	return nil
}

func (s *ScheduledReportsService) UpdateSchedule(ctx context.Context, data *RequestData) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	if err := s.validateRequest(ctx, data, data.Req.Schedule.To, true); err != nil {
		return err
	}

	if _, err := s.updateJob(ctx, data); err != nil {
		return err
	}

	reportRef := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("savedReports").Doc(data.ReportID)
	if _, err := reportRef.Update(ctx, []firestore.Update{{Path: "schedule", Value: data.Req.Schedule}}); err != nil {
		return err
	}

	l.Info("Schedule updated successfully!")

	return nil
}

func (s *ScheduledReportsService) DeleteSchedule(ctx context.Context, data *RequestData) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	if err := s.validateRequest(ctx, data, []string{}, false); err != nil {
		return err
	}

	if err := s.deleteJob(ctx, data.ReportID); err != nil {
		// If the job is not found, continue to delete from Firestore
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
	}

	reportRef := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("savedReports").Doc(data.ReportID)
	if _, err := reportRef.Update(ctx, []firestore.Update{{Path: "schedule", Value: nil}}); err != nil {
		return err
	}

	l.Info("Schedule deleted successfully!")

	return nil
}

func (s *ScheduledReportsService) createJob(ctx context.Context, data *RequestData) (*schedulerpb.Job, error) {
	job, err := s.buildSchedulerJob(data)
	if err != nil {
		return nil, err
	}

	req := &schedulerpb.CreateJobRequest{
		Parent: jobsParentRoot,
		Job:    job,
	}

	resp, err := s.cloudScheduler.CreateJob(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *ScheduledReportsService) updateJob(ctx context.Context, data *RequestData) (*schedulerpb.Job, error) {
	job, err := s.buildSchedulerJob(data)
	if err != nil {
		return nil, err
	}

	req := &schedulerpb.UpdateJobRequest{
		Job: job,
	}

	resp, err := s.cloudScheduler.UpdateJob(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *ScheduledReportsService) deleteJob(ctx context.Context, reportID string) error {
	req := &schedulerpb.DeleteJobRequest{
		Name: getJobName(reportID),
	}
	if err := s.cloudScheduler.DeleteJob(ctx, req); err != nil {
		return err
	}

	return nil
}

func (s *ScheduledReportsService) buildSchedulerJob(data *RequestData) (*schedulerpb.Job, error) {
	requestBody := struct {
		ReportID   string `json:"reportId"`
		CustomerID string `json:"customerId"`
	}{
		ReportID:   data.ReportID,
		CustomerID: data.CustomerID,
	}

	reqBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	if err := s.validateSchedule(data.Req.Schedule, data.Email); err != nil {
		return nil, err
	}

	oidcToken := &schedulerpb.HttpTarget_OidcToken{
		OidcToken: &schedulerpb.OidcToken{
			ServiceAccountEmail: "gcp-jobs@doitintl-cmp-reports-scheduler.iam.gserviceaccount.com",
			Audience:            common.GAEService,
		},
	}

	jobTarget := &schedulerpb.Job_HttpTarget{
		HttpTarget: &schedulerpb.HttpTarget{
			Uri:                 common.CreateCloudTaskURL("/tasks/analytics/reports/send"),
			HttpMethod:          1,
			AuthorizationHeader: oidcToken,
			Body:                reqBodyJSON,
		},
	}

	job := &schedulerpb.Job{
		Name:        getJobName(data.ReportID),
		Description: fmt.Sprintf("[%s] %s", data.Req.CustomerPrimaryDomain, data.Req.Schedule.Subject),
		Target:      jobTarget,
		Schedule:    data.Req.Schedule.Frequency,
		TimeZone:    data.Req.Schedule.Timezone,

		// Stops at 3 total retries (maximum) or total retry duration of one hour.
		// requests will retry at 30s, 60s, 120s.
		RetryConfig: &schedulerpb.RetryConfig{
			RetryCount:         3,
			MinBackoffDuration: durationpb.New(time.Second * 30),
			MaxBackoffDuration: durationpb.New(time.Minute * 2),
			MaxDoublings:       3,
		},
		AttemptDeadline: durationpb.New(time.Minute * 30),
	}

	return job, nil
}

func (s *ScheduledReportsService) validateSchedule(schedule *report.Schedule, email string) error {
	if schedule.To == nil {
		return ErrEmptyRecipientsList
	}

	if res, err := regexp.MatchString(frequencyPattern, schedule.Frequency); err != nil || !res {
		return ErrInvalidFrequency
	}

	if res, err := regexp.MatchString(htmlPattern, schedule.Body); err != nil || res {
		return ErrInvalidScheduleBody
	}

	return nil
}

func (s *ScheduledReportsService) validateRequest(ctx context.Context, data *RequestData, subscriberEmails []string, isUpdate bool) error {
	report, err := s.cloudAnalytics.GetReport(ctx, data.CustomerID, data.ReportID, false)
	if err != nil {
		return err
	}

	if report.Type != "custom" {
		return ErrNotCustomReport
	}

	if data.CustomerID != report.Customer.ID {
		return ErrNotCustomerReport
	}
	// validate that the current user's organization is the same as the report's organization
	if !data.DoitEmployee && len(subscriberEmails) > 0 {
		if data.UserID != "" {
			if valid, err := s.validateUserAndSubscribersInReportOrg(ctx, data.UserID, report, subscriberEmails); !valid {
				if err != nil {
					return err
				}

				return service.ErrReportOrganization
			}
		}
	}

	if isUpdate {
		if report.Public == nil && !data.DoitEmployee {
			isCollaborator := false

			for _, c := range report.Collaborators {
				if c.Email == data.Email {
					isCollaborator = true
				}
			}

			if !isCollaborator {
				return ErrNotCollaborator
			}
		}
	} else {
		var owner string

		for _, c := range report.Collaborators {
			if c.Role == collab.CollaboratorRoleOwner {
				owner = c.Email
			}
		}

		if owner != data.Email {
			return ErrNotReportOwner
		}
	}

	return nil
}

func (s *ScheduledReportsService) validateUserAndSubscribersInReportOrg(ctx context.Context, userID string, r *report.Report, subscriberEmails []string) (bool, error) {
	fs := s.conn.Firestore(ctx)

	userRef := fs.Collection("users").Doc(userID)

	user, err := common.GetUser(ctx, userRef)
	if err != nil {
		return false, err
	}

	var reportOrgID string
	if r.Organization != nil {
		reportOrgID = r.Organization.ID
		// Check if current user is member of report orgnization
		if len(user.Organizations) > 0 {
			if !user.MemberOfOrganization(reportOrgID) {
				return false, service.ErrReportOrganization
			}
			// Check if subscribers are members of organization
			for _, email := range subscriberEmails {
				if ok, err := s.validateSubscriberInReportOrg(ctx, userID, email, reportOrgID); !ok {
					if err != nil {
						return false, err
					}

					return false, service.ErrReportOrganization
				}
			}
		} else {
			return false, service.ErrReportOrganization
		}
	} else {
		if len(user.Organizations) > 0 && r.Type != "preset" {
			return false, service.ErrReportOrganization
		}
	}

	return true, nil
}

func (s *ScheduledReportsService) validateSubscriberInReportOrg(ctx context.Context, userID, subscriberEmail, reportOrgID string) (bool, error) {
	fs := s.conn.Firestore(ctx)

	collections := []string{"users", "invites"}
	for _, collection := range collections {
		docSnaps, err := fs.Collection(collection).Where("email", "==", subscriberEmail).Limit(1).Select("organizations").Documents(ctx).GetAll()
		if err != nil {
			return false, err
		}

		if len(docSnaps) == 0 {
			continue
		}
		// The current user has already been validated as a member of the report's organization.
		if docSnaps[0].Ref.ID == userID {
			break
		}
		// Validate user is member of reports organization
		var subscriberUser common.User
		if err := docSnaps[0].DataTo(&subscriberUser); err != nil {
			return false, err
		}

		if !subscriberUser.ValidateOrganization(reportOrgID) {
			return false, service.ErrReportOrganization
		}

		break
	}

	return true, nil
}
