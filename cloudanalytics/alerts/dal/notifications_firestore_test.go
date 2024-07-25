package dal

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/utils"
	"github.com/doitintl/hello/scheduled-tasks/common"
	testPackage "github.com/doitintl/tests"
	"github.com/stretchr/testify/assert"
)

const (
	alertEtag                 = "123"
	existingNotificationID    = "2022-06-14-QVU="
	newNotificationID         = "newNotificationID"
	testAlertID               = "QhYxks1BkFSBYZNypWgo"
	testPeriod                = "2022-06-14"
	workingPathForBatchCommit = "projects/1/databases/1/documents/1/1"
	wrongEtagNotificationID   = "wrongEtagNotificationID"
)

func deleteDetectedAlerts(t *testing.T, d *NotificationsFirestore) {
	notifications, err := d.GetAlertDetectedNotifications(ctx, notificationCustomerID)
	if err != nil {
		t.Error(err)
	}

	for _, n := range notifications {
		for _, nn := range n {
			ref := d.getNotificationRef(ctx, nn)
			if ref.ID == existingNotificationID || ref.ID == newNotificationID {
				if _, err := ref.Delete(ctx); err != nil {
					t.Error(err)
				}
			}
		}
	}
}

func TestAlertsFirestore_GetAlertDetectedNotifications(t *testing.T) {
	type args struct {
		ctx     context.Context
		alertID string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx:     ctx,
				alertID: notificationID,
			},
			wantErr: false,
		},
	}

	d, err := NewNotificationsFirestore(ctx, common.ProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("CloudAnalyticsAlertsDetected"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.GetAlertDetectedNotifications(tt.args.ctx, notificationCustomerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AlertsFirestore.GetAlertDetectedNotifications() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(got[notificationID]) != 1 {
					t.Errorf("AlertsFirestore.GetAlertDetectedNotifications() = %v, want %v", got, "1")
				}
			}
		})
	}

	deleteDetectedAlerts(t, d)
}

func TestAlertsFirestore_UpdateNotificationTimeSent(t *testing.T) {
	type args struct {
		ctx          context.Context
		notification *domain.Notification
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx: ctx,
				notification: &domain.Notification{
					Alert: &firestore.DocumentRef{
						ID: testAlertID,
					},
					Period:    testPeriod,
					Breakdown: &[]string{"AU"}[0],
				},
			},
			wantErr: false,
		},
	}

	d, err := NewNotificationsFirestore(ctx, common.ProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("CloudAnalyticsAlertsDetected"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notificationsMap, err := d.GetAlertDetectedNotifications(ctx, notificationCustomerID)
			if err != nil {
				t.Error(err)
			} else if len(notificationsMap[notificationID]) == 0 {
				t.Errorf("AlertsFirestore.GetAlertDetectedNotifications() = %v, want %v", notificationsMap, "1")
			}

			if err := d.UpdateNotificationTimeSent(tt.args.ctx, tt.args.notification); (err != nil) != tt.wantErr {
				t.Errorf("AlertsFirestore.UpdateNotificationTimeSent() error = %v, wantErr %v", err, tt.wantErr)
			}

			notificationsMap, err = d.GetAlertDetectedNotifications(ctx, notificationCustomerID)
			if err != nil {
				t.Error(err)
			}

			if len(notificationsMap[notificationID]) != 0 {
				t.Errorf("AlertsFirestore.GetAlertDetectedNotifications() = %v, want %v", notificationsMap, "0")
			}
		})
	}

	deleteDetectedAlerts(t, d)
}

func TestNotificationsFirestore_AddDetectedNotifications(t *testing.T) {
	type args struct {
		ctx                context.Context
		notifications      []*domain.Notification
		addedNotifications []*domain.Notification
		alertEtag          string
	}

	n1 := &domain.Notification{
		Alert: &firestore.DocumentRef{
			ID:   testAlertID,
			Path: workingPathForBatchCommit,
		},
		Period:    testPeriod,
		Breakdown: &[]string{"AU"}[0],
		Customer: &firestore.DocumentRef{
			ID:   notificationCustomerID,
			Path: workingPathForBatchCommit,
		},
		Etag: alertEtag,
	}

	n2 := &domain.Notification{
		Alert: &firestore.DocumentRef{
			ID:   testAlertID,
			Path: workingPathForBatchCommit,
		},
		Period:    "2023-08-14",
		Breakdown: &[]string{"AU"}[0],
		Customer: &firestore.DocumentRef{
			ID:   notificationCustomerID,
			Path: workingPathForBatchCommit,
		},
		Etag: "i-exist",
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "set new notification",
			args: args{
				ctx:       ctx,
				alertEtag: testAlertID,
				notifications: []*domain.Notification{
					n1,
				},
				addedNotifications: []*domain.Notification{
					n1,
				},
			},
		},
		{
			name: "correctly returns added notifications",
			args: args{
				ctx:       ctx,
				alertEtag: n2.Etag,
				notifications: []*domain.Notification{
					n1,
					n2,
				},
				addedNotifications: []*domain.Notification{
					n1,
				},
			},
		},
	}

	d, err := NewNotificationsFirestore(ctx, common.ProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("CloudAnalyticsAlertsDetected"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, err := d.AddDetectedNotifications(tt.args.ctx, tt.args.notifications, tt.args.alertEtag)

			assert.Equal(t, tt.wantErr, err != nil, err)
			assert.Equal(t, tt.args.addedNotifications, added)
		})
	}

	deleteDetectedAlerts(t, d)
}

func TestAlertsFirestore_GetDetectedBreakdowns(t *testing.T) {
	d, err := NewNotificationsFirestore(ctx, common.ProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("CloudAnalyticsAlertsDetected"); err != nil {
		t.Error(err)
	}

	alert := utils.GenerateTestAlert()

	expectedExcludedNotifications := []string{"AU"}

	got, unsentDetectedNotifications, err := d.GetDetectedBreakdowns(ctx, alert.Etag, testAlertID, testPeriod)
	if err != nil || unsentDetectedNotifications != 1 {
		t.Errorf("AlertsFirestore.GetDetectedBreakdowns() error = %v ", err)
		return
	}

	assert.Equal(t, expectedExcludedNotifications, got)
}

func TestAnalyticsAlertsService_getNotificationID(t *testing.T) {
	breakdown := "test_breakdown"
	period := "2020-01-01"
	notification := &domain.Notification{
		Period:    period,
		Breakdown: &breakdown,
	}
	encodedBreakdown := base64.StdEncoding.EncodeToString([]byte(*notification.Breakdown))
	id := fmt.Sprintf("%s-%s", period, encodedBreakdown)

	// breakdown exists
	assert.Equal(t, id, getNotificationID(notification))
	// no breakdown
	assert.Equal(t, period, getNotificationID(&domain.Notification{Period: period}))
}
