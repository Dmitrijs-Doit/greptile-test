package rows_validator

import (
	"context"
	"fmt"
	"sort"
	"time"

	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
)

func (r *RowsValidator) sendValidUpdateNotification(ctx context.Context, billingAccountID string, segments sortableSegments) {
	logger := r.loggerProvider(ctx)
	body := []string{}

	header := "Rows Validator successfully validated latest data for "
	if billingAccountID != googleCloudConsts.MasterBillingAccount {
		header = fmt.Sprintf("%s*%s*", header, billingAccountID)
	} else {
		header = fmt.Sprintf("%s*Raw Billing*", header)
	}

	body = append(body, header)

	sort.Sort(segments)
	body = append(body, fmt.Sprintf("* Segment: %s - %s", segments[0].StartTime.Format(consts.ExportTimeLayoutWithMillis),
		segments[len(segments)-1].EndTime.Format(consts.ExportTimeLayoutWithMillis)))

	sn := &mailer.SimpleNotification{
		Subject:   fmt.Sprintf("IMPORTANT - %s - Flexsave Billing Data Updated Successfully", utils.GetProjectName()),
		Preheader: fmt.Sprintf("Billing data updated successfully for %s", billingAccountID),
		CCs:       defaultCC,
	}

	mnt := &service.MailNotificationTarget{
		To:                 defaultTo,
		SimpleNotification: sn,
	}

	r.notification.SendNotification(ctx, body, mnt)
	logger.Infof("%sSent data validated successfully for %s", logPrefix, billingAccountID)
}

func (r *RowsValidator) sendValidUpdateNotification2(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) {
	logger := r.loggerProvider(ctx)
	body := []string{}

	header := "Monitor successfully validated latest data for "
	if billingAccountID != googleCloudConsts.MasterBillingAccount {
		header = fmt.Sprintf("%s*%s*", header, billingAccountID)
	} else {
		header = fmt.Sprintf("%s*Raw Billing*", header)
	}

	body = append(body, header)

	body = append(body, fmt.Sprintf("* Segment: %s - %s", segment.StartTime.Format(consts.ExportTimeLayoutWithMillis),
		segment.EndTime.Format(consts.ExportTimeLayoutWithMillis)))

	sn := &mailer.SimpleNotification{
		Subject:   fmt.Sprintf("IMPORTANT - %s - Flexsave Billing Data Updated Successfully", utils.GetProjectName()),
		Preheader: fmt.Sprintf("Billing data updated successfully for %s", billingAccountID),
		CCs:       defaultCC,
	}

	mnt := &service.MailNotificationTarget{
		To:                 defaultTo,
		SimpleNotification: sn,
	}

	//TODO lionel remove this
	snt := &service.SlackNotificationTarget{
		Channel: defaultSlackChannel,
	}

	r.notification.SendNotification(ctx, body, mnt, snt)
	//r.notification.SendNotification(ctx, body, mnt)
	logger.Infof("%sSent data validated successfully for %s", logPrefix, billingAccountID)
}

type TestType string

const (
	VerifyRows  TestType = "Row-Validation-Test"
	DataFlowing TestType = "Data-Flow-Test"
	DataExists  TestType = "Data-Exists-Test"
	AllTests    TestType = "All-Tests"
)

func (r *RowsValidator) sendMonitorIssueNotification(ctx context.Context, billingAccountID string, segment *dataStructures.Segment, testType TestType, cause error) {
	logger := r.loggerProvider(ctx)

	body := fmt.Sprintf("Monitor detected a problem during it's execution and was unable to verity %s for  ", testType)
	if billingAccountID != googleCloudConsts.MasterBillingAccount {
		body = fmt.Sprintf("%s*%s*", body, billingAccountID)
	} else {
		body = fmt.Sprintf("%s*Raw Billing*", body)
	}

	if segment != nil {
		body = fmt.Sprintf("%s\n* Segment: %s - %s", body, segment.StartTime.Format(consts.ExportTimeLayoutWithMillis), segment.EndTime.Format(consts.ExportTimeLayoutWithMillis))
	}

	body = fmt.Sprintf("%s\n* Caused by: %s", body, cause)
	sn := &mailer.SimpleNotification{
		Subject:   fmt.Sprintf("URGENT - %s - Flexsave Billing Data Is Not Updated", utils.GetProjectName()),
		Preheader: fmt.Sprintf("Billing Data is not updated for %s", billingAccountID),
		CCs:       defaultCC,
	}

	mnt := &service.MailNotificationTarget{
		To:                 defaultTo,
		SimpleNotification: sn,
	}

	snt := &service.SlackNotificationTarget{
		Channel: defaultSlackChannel,
	}

	r.notification.SendNotification(ctx, []string{body}, mnt, snt)
	logger.Infof("%sSent data not updated notification successfully for %s", logPrefix, billingAccountID)
}

func (r *RowsValidator) sendPastWeekNotification(ctx context.Context, billingAccountID string, date string) {
	logger := r.loggerProvider(ctx)

	body := fmt.Sprintf("Monitor detected that there is no data on date %s for ", date)
	if billingAccountID != googleCloudConsts.MasterBillingAccount {
		body = fmt.Sprintf("%s*%s*", body, billingAccountID)
	} else {
		body = fmt.Sprintf("%s*Raw Billing*", body)
	}

	sn := &mailer.SimpleNotification{
		Subject:   fmt.Sprintf("URGENT - %s - Missing Flexsave data", utils.GetProjectName()),
		Preheader: fmt.Sprintf("No Data on Date %s for %s", date, billingAccountID),
		CCs:       defaultCC,
	}

	mnt := &service.MailNotificationTarget{
		To:                 defaultTo,
		SimpleNotification: sn,
	}

	snt := &service.SlackNotificationTarget{
		Channel: defaultSlackChannel,
	}

	r.notification.SendNotification(ctx, []string{body}, mnt, snt)
	logger.Infof("%sSent data not updated notification successfully for %s", logPrefix, billingAccountID)
}

// TODO lionel remove this
func (r *RowsValidator) sendRowsMismatchNotification(ctx context.Context, data *invalidSegmentData) {
	logger := r.loggerProvider(ctx)
	body := []string{}

	header := "Rows Validator detected the following invalid segments for "
	if data.billingAccountID != googleCloudConsts.MasterBillingAccount {
		header = fmt.Sprintf("%s*%s*:", header, data.billingAccountID)
	} else {
		header = fmt.Sprintf("%s*Raw Billing*:", header)
	}

	body = append(body, header)

	sort.Sort(data.segments)

	for _, s := range data.segments {
		reason := "Duplicated"
		para := fmt.Sprintf("* %s - %s:  \n", s.segment.StartTime.Format(consts.ExportTimeLayoutWithMillis), s.segment.EndTime.Format(consts.ExportTimeLayoutWithMillis))

		if data.billingAccountID != googleCloudConsts.MasterBillingAccount {
			if s.rowsCount[customerTableType] != s.rowsCount[localTableType] {
				if s.rowsCount[customerTableType] > s.rowsCount[localTableType] {
					reason = "Missing"
				}

				para = fmt.Sprintf("%s*%s* rows in local table.  \nCustomer: %d, Local: %d", para, reason, s.rowsCount[customerTableType], s.rowsCount[localTableType])
			} else {
				if s.rowsCount[customerTableType] > s.rowsCount[unifiedTableType] {
					reason = "Missing"
				}

				para = fmt.Sprintf("%s*%s* rows in unified table.  \nCustomer: %d, Unified: %d", para, reason, s.rowsCount[customerTableType], s.rowsCount[unifiedTableType])
			}
		} else {
			if s.rowsCount[localTableType] > s.rowsCount[unifiedTableType] {
				reason = "Missing"
			}

			para = fmt.Sprintf("%s*%s* rows in unified table.  \nLocal: %d, Unified: %d", para, reason, s.rowsCount[localTableType], s.rowsCount[unifiedTableType])
		}

		body = append(body, para)
	}

	sn := &mailer.SimpleNotification{
		Subject:   fmt.Sprintf("URGENT - %s - Flexsave Billing Data Rows Mismatch", utils.GetProjectName()),
		Preheader: fmt.Sprintf("Detected %d invalid segments for %s", len(data.segments), data.billingAccountID),
		CCs:       defaultCC,
	}

	mnt := &service.MailNotificationTarget{
		To:                 defaultTo,
		SimpleNotification: sn,
	}

	snt := &service.SlackNotificationTarget{
		Channel: defaultSlackChannel,
	}

	r.notification.SendNotification(ctx, body, mnt, snt)
	logger.Infof("%sSent rows mismatch notification notification for %s", logPrefix, data.billingAccountID)
}

func (r *RowsValidator) sendRowsMismatchNotification2(ctx context.Context, billingAccount string, invalidCounts []*InvalidCounts) {
	logger := r.loggerProvider(ctx)
	body := []string{}

	header := "Monitor detected the following invalid row counts for "
	if billingAccount != googleCloudConsts.MasterBillingAccount {
		header = fmt.Sprintf("%s*%s*:", header, billingAccount)
	} else {
		header = fmt.Sprintf("%s*Raw Billing*:", header)
	}

	body = append(body, header)

	for _, invalidCount := range invalidCounts {
		reason := "Duplicated"
		para := fmt.Sprintf("* %s:  \n", invalidCount.timestamp)

		if invalidCount.expected > invalidCount.found {
			reason = "Missing"
		}

		para = fmt.Sprintf("%s*%s* rows for export_time=\"%s\".\nExpected: %d, Found: %d", para, reason, invalidCount.timestamp, invalidCount.expected, invalidCount.found)
		body = append(body, para)
	}

	sn := &mailer.SimpleNotification{
		Subject:   fmt.Sprintf("URGENT - %s - Monitor Flexsave Billing Data Rows Mismatch", utils.GetProjectName()),
		Preheader: fmt.Sprintf("Detected %d invalid export_time timestamps for %s", len(invalidCounts), billingAccount),
		CCs:       defaultCC,
	}

	mnt := &service.MailNotificationTarget{
		To:                 defaultTo,
		SimpleNotification: sn,
	}

	snt := &service.SlackNotificationTarget{
		Channel: defaultSlackChannel,
	}

	r.notification.SendNotification(ctx, body, mnt, snt)
	logger.Infof("%sSent rows mismatch notification notification for %s", logPrefix, billingAccount)
}

func (r *RowsValidator) sendDataNotUpdatedNotification(ctx context.Context, billingAccountID string, duration time.Duration) {
	logger := r.loggerProvider(ctx)

	body := fmt.Sprintf("Rows Validator detected that data is not updated for last %s hours for ", duration)
	if billingAccountID != googleCloudConsts.MasterBillingAccount {
		body = fmt.Sprintf("%s*%s*", body, billingAccountID)
	} else {
		body = fmt.Sprintf("%s*Raw Billing*", body)
	}

	sn := &mailer.SimpleNotification{
		Subject:   fmt.Sprintf("URGENT - %s - Flexsave Billing Data Is Not Updated", utils.GetProjectName()),
		Preheader: fmt.Sprintf("Billing Data is not updated for %s", billingAccountID),
		CCs:       defaultCC,
	}

	mnt := &service.MailNotificationTarget{
		To:                 defaultTo,
		SimpleNotification: sn,
	}

	snt := &service.SlackNotificationTarget{
		Channel: defaultSlackChannel,
	}

	r.notification.SendNotification(ctx, []string{body}, mnt, snt)
	logger.Infof("%sSent data not updated notification successfully for %s", logPrefix, billingAccountID)
}

func (r *RowsValidator) sendQueryFailedNotification(ctx context.Context, billingAccountID string, failedOn *tableQueryErrors) {
	logger := r.loggerProvider(ctx)
	body := []string{}

	for t, failedErrs := range *failedOn {
		if len(failedErrs) > 0 {
			errData := fmt.Sprintf("* Rows Validator detected that query failed on *%s* table with error(s):  \n", t)
			for _, se := range failedErrs {
				errData = fmt.Sprintf("%s%s - %s: %s\n", errData, se.segment.StartTime.Format(consts.ExportTimeLayoutWithMillis), se.segment.EndTime.Format(consts.ExportTimeLayoutWithMillis), se.err)
			}

			body = append(body, errData)
		}
	}

	footer := "for  "
	if billingAccountID != googleCloudConsts.MasterBillingAccount {
		footer = fmt.Sprintf("%s*%s*", footer, billingAccountID)
	} else {
		footer = fmt.Sprintf("%s*Raw Billing*", footer)
	}

	body = append(body, footer)

	sn := &mailer.SimpleNotification{
		Subject:   fmt.Sprintf("IMPORTANT - %s - Flexsave Billing Data Row Validator query failed", utils.GetProjectName()),
		Preheader: fmt.Sprintf("Row Validator query failed for %s", billingAccountID),
		CCs:       defaultCC,
	}

	mnt := &service.MailNotificationTarget{
		To:                 defaultTo,
		SimpleNotification: sn,
	}

	snt := &service.SlackNotificationTarget{
		Channel: defaultSlackChannel,
	}

	r.notification.SendNotification(ctx, body, mnt, snt)

	logger.Infof("%sSent Row Validator query failed notification successfully for %s", logPrefix, billingAccountID)
}
