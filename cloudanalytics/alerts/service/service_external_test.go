package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	attributionMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metadataIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/iface"
	metadataServiceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/mocks"
	metricsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal/mocks"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDalMock "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	userMocks "github.com/doitintl/hello/scheduled-tasks/user/dal/mocks"
)

func TestAnalyticsAlertsExternalService_GetAlert(t *testing.T) {
	type fields struct {
		alertsDal *mocks.Alerts
	}

	type args struct {
		ctx     context.Context
		alertID string
	}

	var alertID = "alert-id"

	var ctx = context.Background()

	var alertTime = time.Now().UTC()

	var alertTimeAPI = alertTime.UnixMilli()

	alert := &domain.Alert{
		ID:              alertID,
		Name:            "Alert Name",
		TimeCreated:     alertTime,
		TimeModified:    alertTime,
		TimeLastAlerted: &alertTime,
		Config: &domain.Config{
			Values:    []float64{100},
			Operator:  report.MetricFilterGreaterThan,
			Condition: domain.ConditionValue,
			Filters:   []*report.ConfigFilter{},
			Rows: []string{
				"breakdown:123",
			},
			TimeInterval: report.TimeIntervalDay,
			Scope: []*firestore.DocumentRef{{
				ID:   "attributionId",
				Path: "attributions/attributionId",
			}},
			DataSource: report.DataSourceBilling,
		},
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		expectedErr error
		want        *AlertAPI

		on func(*fields)
	}{
		{
			name: "Get expected alert with config",
			args: args{
				ctx:     ctx,
				alertID: alertID,
			},
			want: &AlertAPI{
				ID:          "alert-id",
				Name:        "Alert Name",
				CreateTime:  alertTimeAPI,
				UpdateTime:  alertTimeAPI,
				LastAlerted: &alertTimeAPI,
				Recipients:  nil,
				Config: &AlertConfigAPI{
					Attributions: []string{"attributionId"},
					Metric: MetricConfig{
						Type:  "basic",
						Value: "cost",
					},
					Currency:        "",
					Scopes:          []Scope{},
					TimeInterval:    "day",
					Condition:       "value",
					Operator:        "gt",
					Value:           100,
					EvaluateForEach: "breakdown:123",
					DataSource:      domainExternalReport.ExternalDataSourceBilling,
				},
			},
			on: func(f *fields) {
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(alert, nil).
					Once()
			},
		},
		{
			name: "Get expected alert without config",
			args: args{
				ctx:     ctx,
				alertID: alertID,
			},
			want: &AlertAPI{
				ID:          "alert-id",
				Name:        "Alert Name",
				CreateTime:  alertTimeAPI,
				UpdateTime:  alertTimeAPI,
				LastAlerted: &alertTimeAPI,
				Recipients:  nil,
			},
			on: func(f *fields) {
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(&domain.Alert{
						ID:              alertID,
						Name:            "Alert Name",
						TimeCreated:     alertTime,
						TimeModified:    alertTime,
						TimeLastAlerted: &alertTime,
					}, nil).
					Once()
			},
		},
		{
			name: "Get expected error if no alert found",
			args: args{
				ctx:     ctx,
				alertID: alertID,
			},
			expectedErr: errors.New("not found"),
			on: func(f *fields) {
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(nil, errors.New("not found")).
					Once()
			},
		},
		{
			name: "Get expected error if wrong alert data",
			args: args{
				ctx:     ctx,
				alertID: alertID,
			},
			expectedErr: errors.New("no metric found"),
			on: func(f *fields) {
				wrongAlert := alert
				wrongAlert.Config.Metric = 10
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(alert, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				alertsDal: &mocks.Alerts{},
			}
			s := &AnalyticsAlertsService{
				alertsDal: tt.fields.alertsDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.GetAlert(tt.args.ctx, tt.args.alertID)

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.want, got, "not equal")
		})
	}
}

func TestAnalyticsAlertsExternalService_ListAlerts(t *testing.T) {
	type fields struct {
		alertsDal    *mocks.Alerts
		customersDAL *customerDalMock.Customers
	}

	var ctx = context.Background()

	var alertTime = time.Now().UTC()

	var alertTimeAPI = alertTime.UnixMilli()

	var alert2Time = time.Now().UTC().Add(1)

	var alert2TimeAPI = alert2Time.UnixMilli()

	var ownerEmail = "owner@doit.com"

	var documentID = "customerID"

	errorText := "expected error"
	expectedError := errors.New(errorText)

	alert := &domain.Alert{
		Name:            "Alert Name",
		ID:              "alert-id",
		TimeCreated:     alertTime,
		TimeModified:    alertTime,
		TimeLastAlerted: &alertTime,
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{
					Email: ownerEmail,
					Role:  collab.CollaboratorRoleOwner,
				},
				{
					Email: "viewer@doit.com",
					Role:  collab.CollaboratorRoleViewer,
				},
			},
		},
		Config: &domain.Config{
			Currency:  fixer.USD,
			Values:    []float64{100},
			Operator:  report.MetricFilterGreaterThan,
			Condition: domain.ConditionValue,
			Rows: []string{
				"breakdown:123",
			},
			TimeInterval: report.TimeIntervalDay,
			DataSource:   report.DataSourceBilling,
		},
	}
	alert2 := &domain.Alert{
		Name:            "Alert Name 2",
		ID:              "alert-id-2",
		TimeCreated:     alert2Time,
		TimeModified:    alert2Time,
		TimeLastAlerted: &alert2Time,
		Config:          alert.Config,
		Access:          collab.Access{Collaborators: alert.Collaborators},
	}
	alert3 := &domain.Alert{
		Name:            "Alert Name 3",
		ID:              "alert-id-3",
		TimeCreated:     alert2Time,
		TimeModified:    alert2Time,
		TimeLastAlerted: &alert2Time,
		Config: &domain.Config{
			Metric: -1,
		},
		Access: collab.Access{Collaborators: alert.Collaborators},
	}
	alert3Error := errors.New("no metric found")

	config := &AlertConfigAPI{
		Metric:          MetricConfig{Type: BasicMetric, Value: "cost"},
		Currency:        fixer.USD,
		TimeInterval:    "day",
		Condition:       "value",
		Operator:        "gt",
		Value:           100,
		EvaluateForEach: "breakdown:123",
		Attributions:    []string{},
		Scopes:          []Scope{},
		DataSource:      domainExternalReport.ExternalDataSourceBilling,
	}

	tests := []struct {
		name        string
		fields      fields
		args        ExternalAPIListArgsReq
		expectedErr error
		want        ExternalAlertList

		on func(*fields)
	}{
		{
			name: "List expected alerts",
			args: ExternalAPIListArgsReq{
				CustomerID: "customer-id",
				Email:      "email@test.com",
				SortBy:     "createTime",
				SortOrder:  firestore.Desc,
			},
			want: ExternalAlertList{
				Alerts: []customerapi.SortableItem{
					ListAlertAPI{
						ID:          "alert-id-2",
						Name:        "Alert Name 2",
						CreateTime:  alert2TimeAPI,
						UpdateTime:  alert2TimeAPI,
						LastAlerted: &alert2TimeAPI,
						Config:      config,
						Owner:       ownerEmail,
					},
					ListAlertAPI{
						ID:          "alert-id",
						Name:        "Alert Name",
						CreateTime:  alertTimeAPI,
						UpdateTime:  alertTimeAPI,
						LastAlerted: &alertTimeAPI,
						Config:      config,
						Owner:       ownerEmail,
					},
				},
				RowCount: 2,
			},
			on: func(f *fields) {
				f.customersDAL.
					On("GetRef", ctx, "customer-id").
					Return(&firestore.DocumentRef{
						ID: documentID,
					}).
					Once()

				f.alertsDal.
					On("GetAlertsByCustomer", ctx, &iface.AlertsByCustomerArgs{
						CustomerRef: &firestore.DocumentRef{
							ID: documentID,
						},
						Email: "email@test.com",
					}).
					Return([]domain.Alert{*alert, *alert2}, nil).
					Once()
			},
		},
		{
			name: "List alerts with page information, correctly handles negative max results",
			args: ExternalAPIListArgsReq{
				CustomerID: "customer-id",
				Email:      "email@test.com",
				SortBy:     "createTime",
				SortOrder:  firestore.Desc,
				MaxResults: -1,
			},
			want: ExternalAlertList{
				Alerts: []customerapi.SortableItem{
					ListAlertAPI{
						ID:          "alert-id-2",
						Name:        "Alert Name 2",
						CreateTime:  alert2TimeAPI,
						UpdateTime:  alert2TimeAPI,
						LastAlerted: &alert2TimeAPI,
						Config:      config,
						Owner:       ownerEmail,
					},
					ListAlertAPI{
						ID:          "alert-id",
						Name:        "Alert Name",
						CreateTime:  alertTimeAPI,
						UpdateTime:  alertTimeAPI,
						LastAlerted: &alertTimeAPI,
						Config:      config,
						Owner:       ownerEmail,
					},
				},
				RowCount: 2,
			},
			on: func(f *fields) {
				f.customersDAL.
					On("GetRef", ctx, "customer-id").
					Return(&firestore.DocumentRef{
						ID: documentID,
					}).
					Once()

				f.alertsDal.
					On("GetAlertsByCustomer", ctx, &iface.AlertsByCustomerArgs{
						CustomerRef: &firestore.DocumentRef{
							ID: documentID,
						},
						Email: "email@test.com",
					}).
					Return([]domain.Alert{*alert, *alert2}, nil).
					Once()
			},
		},
		{
			name: "List alerts with page information",
			args: ExternalAPIListArgsReq{
				CustomerID: "customer-id",
				Email:      "email@test.com",
				SortBy:     "createTime",
				SortOrder:  firestore.Desc,
				MaxResults: 1,
			},
			want: ExternalAlertList{
				Alerts: []customerapi.SortableItem{
					ListAlertAPI{
						ID:          "alert-id-2",
						Name:        "Alert Name 2",
						CreateTime:  alert2TimeAPI,
						UpdateTime:  alert2TimeAPI,
						LastAlerted: &alert2TimeAPI,
						Config:      config,
						Owner:       ownerEmail,
					},
				},
				RowCount:  1,
				PageToken: "YWxlcnQtaWQ",
			},
			on: func(f *fields) {
				f.customersDAL.
					On("GetRef", ctx, "customer-id").
					Return(&firestore.DocumentRef{
						ID: documentID,
					}).
					Once()

				f.alertsDal.
					On("GetAlertsByCustomer", ctx, &iface.AlertsByCustomerArgs{
						CustomerRef: &firestore.DocumentRef{
							ID: documentID,
						},
						Email: "email@test.com",
					}).
					Return([]domain.Alert{*alert, *alert2}, nil).
					Once()
			},
		},
		{
			name: "List alerts with page information correctly follows page token",
			args: ExternalAPIListArgsReq{
				CustomerID:    "customer-id",
				Email:         "email@test.com",
				SortBy:        "createTime",
				SortOrder:     firestore.Desc,
				MaxResults:    1,
				NextPageToken: "YWxlcnQtaWQ",
			},
			want: ExternalAlertList{
				Alerts: []customerapi.SortableItem{
					ListAlertAPI{
						ID:          "alert-id",
						Name:        "Alert Name",
						CreateTime:  alertTimeAPI,
						UpdateTime:  alertTimeAPI,
						LastAlerted: &alertTimeAPI,
						Config:      config,
						Owner:       ownerEmail,
					},
				},
				RowCount: 1,
			},
			on: func(f *fields) {
				f.customersDAL.
					On("GetRef", ctx, "customer-id").
					Return(&firestore.DocumentRef{
						ID: documentID,
					}).
					Once()

				f.alertsDal.
					On("GetAlertsByCustomer", ctx, &iface.AlertsByCustomerArgs{
						CustomerRef: &firestore.DocumentRef{
							ID: documentID,
						},
						Email: "email@test.com",
					}).
					Return([]domain.Alert{*alert, *alert2}, nil).
					Once()
			},
		},
		{
			name: "Handle GetAlertsByCustomer error",
			args: ExternalAPIListArgsReq{
				CustomerID: "customer-id",
				Email:      "email@test.com",
				SortBy:     "createTime",
				SortOrder:  firestore.Desc,
			},
			expectedErr: expectedError,
			on: func(f *fields) {
				f.customersDAL.
					On("GetRef", ctx, "customer-id").
					Return(&firestore.DocumentRef{
						ID: documentID,
					}).
					Once()

				f.alertsDal.
					On("GetAlertsByCustomer", ctx, &iface.AlertsByCustomerArgs{
						CustomerRef: &firestore.DocumentRef{
							ID: documentID,
						},
						Email: "email@test.com",
					}).
					Return(nil, expectedError).
					Once()
			},
		},
		{
			name: "Handle invalid alert error",
			args: ExternalAPIListArgsReq{
				CustomerID: "customer-id",
				Email:      "email@test.com",
				SortBy:     "createTime",
				SortOrder:  firestore.Desc,
			},
			expectedErr: alert3Error,
			on: func(f *fields) {
				f.customersDAL.
					On("GetRef", ctx, "customer-id").
					Return(&firestore.DocumentRef{
						ID: documentID,
					}).
					Once()

				f.alertsDal.
					On("GetAlertsByCustomer", ctx, &iface.AlertsByCustomerArgs{
						CustomerRef: &firestore.DocumentRef{
							ID: documentID,
						},
						Email: "email@test.com",
					}).
					Return([]domain.Alert{*alert3}, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				alertsDal:    &mocks.Alerts{},
				customersDAL: &customerDalMock.Customers{},
			}
			s := &AnalyticsAlertsService{
				alertsDal:    tt.fields.alertsDal,
				customersDAL: tt.fields.customersDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.ListAlerts(ctx, tt.args)

			assert.Equal(t, err, tt.expectedErr)

			if got != nil {
				assert.Equal(t, tt.want.RowCount, got.RowCount, "row counts are not equal")
				assert.Equal(t, tt.want.PageToken, got.PageToken, "page tokens are not equal")
				assert.Equal(t, tt.want.Alerts, got.Alerts, "Alerts are not equal")
			}
		})
	}
}

func TestAnalyticsAlertsExternalService_DeleteAlert(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		alertsDal      *mocks.Alerts
		customerDAL    *customerMocks.Customers
	}

	type args struct {
		ctx        context.Context
		customerID string
		email      string
		alertID    string
	}

	ctx := context.Background()

	email := "requester@example.com"
	alertID := "test_alert_id"
	customerID := "test_customer_id"

	customerRef := &firestore.DocumentRef{
		ID: customerID,
	}
	expectedError := errors.New("error retrieving alert")

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successful deletion with owner",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				alertID:    alertID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.alertsDal.On("GetAlert", testutils.ContextBackgroundMock, alertID).
					Return(&domain.Alert{
						Customer: customerRef,
						Access: collab.Access{Collaborators: []collab.Collaborator{
							{
								Email: email,
								Role:  collab.CollaboratorRoleOwner,
							},
						}},
					}, nil).
					Once()
				f.alertsDal.On("DeleteAlert", testutils.ContextBackgroundMock, alertID).
					Return(nil).
					Once()
			},
		},
		{
			name: "error retrieving alert",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				alertID:    alertID,
			},
			expectedErr: expectedError,
			on: func(f *fields) {
				f.alertsDal.On("GetAlert", testutils.ContextBackgroundMock, alertID).
					Return(nil, expectedError).
					Once()
			},
		},
		{
			name: "error requester not owner",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				alertID:    alertID,
			},
			wantErr:     true,
			expectedErr: domain.ErrorUnAuthorized,
			on: func(f *fields) {
				f.alertsDal.On("GetAlert", testutils.ContextBackgroundMock, alertID).
					Return(&domain.Alert{
						Customer: customerRef,
						Access: collab.Access{Collaborators: []collab.Collaborator{
							{
								Email: email,
								Role:  collab.CollaboratorRoleEditor,
							},
						}},
					}, nil).
					Once()
			},
		},
		{
			name: "error deleting report",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				alertID:    alertID,
			},
			expectedErr: expectedError,
			on: func(f *fields) {
				f.alertsDal.On("GetAlert", testutils.ContextBackgroundMock, alertID).
					Return(&domain.Alert{
						Customer: customerRef,
						Access: collab.Access{Collaborators: []collab.Collaborator{
							{
								Email: email,
								Role:  collab.CollaboratorRoleOwner,
							},
						}},
					}, nil).
					Once()
				f.alertsDal.On("DeleteAlert", testutils.ContextBackgroundMock, alertID).
					Return(expectedError).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				alertsDal:      &mocks.Alerts{},
				customerDAL:    &customerMocks.Customers{},
			}

			s := &AnalyticsAlertsService{
				loggerProvider: tt.fields.loggerProvider,
				alertsDal:      tt.fields.alertsDal,
				customersDAL:   tt.fields.customerDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.DeleteAlert(
				ctx,
				tt.args.customerID,
				tt.args.email,
				tt.args.alertID,
			)

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("AnalyticsAlertsService.DeleteAlert() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestAnalyticsAlertsExternalService_CreateAlert(t *testing.T) {
	type fields struct {
		alertsDal       *mocks.Alerts
		customersDAL    *customerMocks.Customers
		attributionsDAL *attributionMocks.Attributions
		metricsDAL      *metricsMocks.Metrics
		metadataService *metadataServiceMock.MetadataIface
		userDAL         *userMocks.IUserFirestoreDAL
	}

	type args struct {
		ctx          context.Context
		alertRequest *AlertRequest
		customerID   string
		userID       string
		email        string
	}

	ctx := context.Background()
	customerID := "customerID"
	userID := "userID"
	customerEmail := "customer@test.com"
	pathForCustomerRef := "customers/" + customerID
	customerRef := &firestore.DocumentRef{
		ID:   customerID,
		Path: pathForCustomerRef,
	}

	orgRef := &firestore.DocumentRef{
		ID:   "orgId",
		Path: pathForCustomerRef + "/customerOrgs/" + rootOrgID,
	}

	alertID := "alert-id"
	alertTime := time.Now().UTC()
	alertTimeAPI := alertTime.UnixMilli()
	isDoitEmployee := false
	attributionID := "test-attribution-id"
	attributionIDNonExistent := "test-attribution-id-non-existent"

	tests := []struct {
		name          string
		fields        fields
		args          args
		validationErr []error
		want          *AlertAPI

		on func(*fields)
	}{
		{
			name: "Get validation errors on create Alert",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				userID:     userID,
				email:      customerEmail,
				alertRequest: &AlertRequest{
					Config: &AlertConfigAPI{
						Condition: "percentage1",
						Currency:  "TBH",
						Metric: MetricConfig{
							Type:  "custom",
							Value: "metricId",
						},
						Operator:        "gt1",
						EvaluateForEach: "fixed:service_description",
						Attributions:    []string{attributionID, attributionIDNonExistent},
						TimeInterval:    "hour1",
						Value:           423,
						DataSource:      "non-existent",
					},
					Name:       "Alert name",
					Recipients: []string{"name@test2.com"},
				},
			},
			validationErr: []error{
				errormsg.ErrorMsg{Field: "recipients", Message: ErrForbiddenEmail},
				errormsg.ErrorMsg{Field: "config.condition", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.currency", Message: ErrNotSupportedCurrency},
				errormsg.ErrorMsg{Field: "config.metric.value", Message: ErrUnknown},
				errormsg.ErrorMsg{Field: "config.operator", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.timeInterval", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.dataSource", Message: domainExternalReport.ErrInvalidDatasourceValue + ": non-existent"},
				errormsg.ErrorMsg{Field: "config.attributions", Message: attributionID + " not permitted for the user, " + attributionIDNonExistent + " not found"},
				errormsg.ErrorMsg{Field: "config.scopes", Message: ErrNotFound},
				errormsg.ErrorMsg{Field: "config.evaluateForEach", Message: ErrNotFound},
			},

			on: func(f *fields) {
				f.customersDAL.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()

				f.customersDAL.
					On("GetCustomer", ctx, customerID).
					Return(&common.Customer{
						Domains: []string{"test.com", "test1.com"},
					}, nil).
					Once()

				f.attributionsDAL.On("GetAttribution", ctx, attributionID).
					Return(
						&attribution.Attribution{
							Type: "custom",
							Customer: &firestore.DocumentRef{
								ID:   "otherCustomerID",
								Path: pathForCustomerRef,
							},
						}, nil).
					Once()

				f.attributionsDAL.On("GetAttribution", ctx, attributionIDNonExistent).
					Return(
						nil, errors.New("not found")).
					Once()

				f.metricsDAL.On("GetCustomMetric", ctx, "metricId").
					Return(nil, errors.New("some error")).
					Once()

				f.metadataService.On("ExternalAPIGet", mock.AnythingOfType("iface.ExternalAPIGetArgs")).
					Return(nil, errors.New("some error")).
					Once()

				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "id",
							Type: "type",
						},
					},
						nil,
					)

				f.userDAL.On("Get", ctx, userID).
					Return(&common.User{}, nil).
					Once()
			},
		},
		{
			name: "Get validation errors on create Alert 2",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				userID:     userID,
				email:      customerEmail,
				alertRequest: &AlertRequest{
					Config: &AlertConfigAPI{
						Condition: "forecast",
						Currency:  "EUR",
						Metric: MetricConfig{
							Type:  "basic",
							Value: "something",
						},
						Operator:        "lt1",
						Scopes:          []Scope{{Key: "key", Type: "type", Regexp: new(string)}},
						EvaluateForEach: "fixed:service_description",
						Attributions:    []string{attributionID},
						TimeInterval:    "hour",
						Value:           423,
					},
					Name:       "Alert name",
					Recipients: []string{"name@test2"},
				},
			},
			validationErr: []error{
				errormsg.ErrorMsg{Field: "recipients", Message: ErrNotValidEmail},
				errormsg.ErrorMsg{Field: "config.metric.value", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.operator", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.timeInterval", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.attributions", Message: attributionID + " not permitted for the user"},
				errormsg.ErrorMsg{Field: "config.scopes", Message: ErrInvalidScopeMetadataType},
				errormsg.ErrorMsg{Field: "config.evaluateForEach", Message: ErrForecastMetadataIncompatible},
			},

			on: func(f *fields) {
				f.customersDAL.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()

				f.customersDAL.
					On("GetCustomer", ctx, customerID).
					Return(&common.Customer{
						Domains: []string{"test.com", "test1.com"},
					}, nil).
					Once()

				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "key",
							Type: "type",
						},
					},
						nil,
					)

				f.attributionsDAL.On("GetAttribution", ctx, attributionID).
					Return(
						&attribution.Attribution{
							Type: "custom",
							Customer: &firestore.DocumentRef{
								ID:   "otherCustomerID",
								Path: pathForCustomerRef,
							},
						}, nil).
					Once()
			},
		},
		{
			name: "Get validation errors on create Alert 3",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				userID:     userID,
				email:      customerEmail,
				alertRequest: &AlertRequest{
					Config: &AlertConfigAPI{
						Condition: "forecast",
						Currency:  "EUR",
						Metric: MetricConfig{
							Type:  "basic",
							Value: "something",
						},
						Operator:        "gt",
						Scopes:          []Scope{{Key: "key", Type: "type", Regexp: new(string)}},
						EvaluateForEach: "fixed:service_description",
						Attributions:    []string{attributionID},
						TimeInterval:    "year",
						Value:           0.9,
					},
					Name:       "Alert name",
					Recipients: []string{"test@test.com"},
				},
			},
			validationErr: []error{
				errormsg.ErrorMsg{Field: "config.metric.value", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.attributions", Message: attributionID + " invalid: managed attributions cannot be used"},
				errormsg.ErrorMsg{Field: "config.scopes", Message: ErrInvalidScopeMetadataType},
				errormsg.ErrorMsg{Field: "config.evaluateForEach", Message: ErrForecastMetadataIncompatible},
			},

			on: func(f *fields) {
				f.customersDAL.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()

				f.customersDAL.
					On("GetCustomer", ctx, customerID).
					Return(&common.Customer{
						Domains: []string{"test.com", "test1.com"},
					}, nil).
					Once()

				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "key",
							Type: "type",
						},
					},
						nil,
					)

				f.attributionsDAL.On("GetAttribution", ctx, attributionID).
					Return(
						&attribution.Attribution{
							Type: "managed",
							Customer: &firestore.DocumentRef{
								ID:   "otherCustomerID",
								Path: pathForCustomerRef,
							},
						}, nil).
					Once()
			},
		},
		{
			name: "Successful Alert Creation",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				userID:     userID,
				email:      customerEmail,
				alertRequest: &AlertRequest{
					Config: &AlertConfigAPI{
						Condition: "percentage-change",
						Currency:  "USD",
						Metric: MetricConfig{
							Type:  "basic",
							Value: "usage",
						},
						Operator:        "gt",
						Scopes:          []Scope{{Key: "team", Type: "label", Values: &[]string{"team_kraken"}}},
						EvaluateForEach: "fixed:service_description",
						Attributions:    []string{attributionID},
						TimeInterval:    "year",
						Value:           423,
						DataSource:      domainExternalReport.ExternalDataSourceBilling,
					},
					Name:       "Alert name",
					Recipients: []string{"name@test2.com", "name@test1.slack.com"},
				},
			},
			want: &AlertAPI{
				ID:          alertID,
				Name:        "Alert Name",
				CreateTime:  alertTimeAPI,
				UpdateTime:  alertTimeAPI,
				LastAlerted: nil,
				Recipients:  []string{"name@test2.com", "name@test1.slack.com"},
				Config: &AlertConfigAPI{
					Attributions: []string{attributionID},
					Metric: MetricConfig{
						Type:  "basic",
						Value: "usage",
					},
					Currency:        "USD",
					TimeInterval:    "year",
					Condition:       "percentage-change",
					Operator:        "gt",
					Value:           423,
					EvaluateForEach: "fixed:service_description",
					Scopes:          []Scope{{Key: "team", Type: "label", Values: &[]string{"team_kraken"}}},
					DataSource:      domainExternalReport.ExternalDataSourceBilling,
				},
			},

			on: func(f *fields) {
				f.customersDAL.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()

				f.customersDAL.
					On("GetCustomer", ctx, customerID).
					Return(&common.Customer{
						Domains: []string{"test.com", "test1.com", "test2.com"},
					}, nil).
					Once()

				f.attributionsDAL.On("GetAttribution", ctx, attributionID).
					Return(
						&attribution.Attribution{
							Type:     "custom",
							Customer: customerRef,
						}, nil).
					Once()

				f.attributionsDAL.On("GetRef", ctx, attributionID).
					Return(
						&firestore.DocumentRef{
							ID:   attributionID,
							Path: "attributions/" + attributionID,
						}, nil).
					Once()

				f.metricsDAL.On("GetCustomMetric", ctx, "metricId").
					Return(&metrics.CalculatedMetric{Customer: customerRef}, nil).
					Once()

				f.metricsDAL.On("GetRef", ctx, "metricId").
					Return(&firestore.DocumentRef{
						ID:   "metricId",
						Path: "metrics/metricId",
					}, nil).
					Once()

				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "team",
							Type: "label",
						},
					},
						nil,
					)

				f.alertsDal.
					On("CreateAlert", ctx, mock.AnythingOfType("*domain.Alert")).
					Return(&domain.Alert{
						ID:              alertID,
						Name:            "Alert Name",
						TimeCreated:     alertTime,
						TimeModified:    alertTime,
						TimeLastAlerted: nil,
						Recipients:      []string{"name@test2.com", "name@test1.slack.com"},
						Config: &domain.Config{
							Scope: []*firestore.DocumentRef{{
								ID:   attributionID,
								Path: "attributions/" + attributionID,
							}},
							Metric:       1,
							Currency:     "USD",
							TimeInterval: "year",
							Condition:    "percentage",
							Operator:     ">",
							Values:       []float64{423},
							Rows:         []string{"fixed:service_description"},
							Filters: []*report.ConfigFilter{
								{
									BaseConfigFilter: report.BaseConfigFilter{
										Key:    "team",
										Type:   "label",
										Values: &[]string{"team_kraken"},
									}},
							},
							DataSource: report.DataSourceBilling,
						},
					}, nil).
					Once()

				f.alertsDal.
					On("GetCustomerOrgRef", ctx, customerID, rootOrgID).
					Return(orgRef).Once()

				f.metadataService.On("ExternalAPIGet", mock.AnythingOfType("iface.ExternalAPIGetArgs")).
					Return(&metadataIface.ExternalAPIGetRes{}, nil).Once()

				f.userDAL.On("Get", ctx, userID).
					Return(&common.User{}, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				alertsDal:       &mocks.Alerts{},
				customersDAL:    &customerMocks.Customers{},
				attributionsDAL: &attributionMocks.Attributions{},
				metricsDAL:      &metricsMocks.Metrics{},
				metadataService: &metadataServiceMock.MetadataIface{},
				userDAL:         &userMocks.IUserFirestoreDAL{},
			}

			s := &AnalyticsAlertsService{
				alertsDal:       tt.fields.alertsDal,
				customersDAL:    tt.fields.customersDAL,
				attributionsDAL: tt.fields.attributionsDAL,
				metrics:         tt.fields.metricsDAL,
				metadataService: tt.fields.metadataService,
				userDal:         tt.fields.userDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			resp := s.CreateAlert(tt.args.ctx, ExternalAPICreateUpdateArgsReq{
				AlertRequest:   tt.args.alertRequest,
				CustomerID:     tt.args.customerID,
				UserID:         tt.args.userID,
				Email:          tt.args.email,
				IsDoitEmployee: isDoitEmployee,
			})

			assert.Equal(t, tt.validationErr, resp.ValidationErrors)
			assert.Equal(t, tt.want, resp.Alert)
		})
	}
}

func TestAnalyticsAlertsExternalService_UpdateAlert(t *testing.T) {
	type fields struct {
		alertsDal       *mocks.Alerts
		customersDAL    *customerMocks.Customers
		attributionsDAL *attributionMocks.Attributions
		metricsDAL      *metricsMocks.Metrics
		metadataService *metadataServiceMock.MetadataIface
		userDAL         *userMocks.IUserFirestoreDAL
	}

	type args struct {
		ctx          context.Context
		alertRequest *AlertRequest
		customerID   string
		userID       string
		email        string
	}

	ctx := context.Background()
	customerID := "customerID"
	userID := "userID"
	customerEmail := "customer@test.com"
	pathForCustomerRef := "customers/" + customerID
	customerRef := &firestore.DocumentRef{
		ID:   customerID,
		Path: pathForCustomerRef,
	}
	alertID := "alert-id"
	alertTime := time.Now().UTC()
	isDoitEmployee := false
	attributionID := "test-attribution-id"
	attributionIDNonExistent := "test-attribution-id-non-existent"

	currentAlert := &domain.Alert{
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{
					Email: customerEmail,
					Role:  collab.CollaboratorRoleOwner,
				},
			},
		}}

	updatedAlert := &domain.Alert{
		ID:              alertID,
		Name:            "Alert Name",
		TimeCreated:     alertTime,
		TimeModified:    alertTime,
		TimeLastAlerted: nil,
		Recipients:      []string{"name@test2.com", "name@test1.slack.com"},
		Config: &domain.Config{
			Scope: []*firestore.DocumentRef{{
				ID:   attributionID,
				Path: "attributions/" + attributionID,
			}},
			Metric:       1,
			Currency:     "USD",
			TimeInterval: "year",
			Condition:    "percentage",
			Operator:     ">",
			Values:       []float64{423},
			Rows:         []string{"fixed:service_description"},
		}}

	updatedAlertAPI, _ := toAlertAPI(updatedAlert)

	tests := []struct {
		name          string
		fields        fields
		args          args
		validationErr []error
		expectedErr   error
		want          *AlertAPI
		on            func(*fields)
	}{
		{
			name: "User has no permissions to update alert",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				userID:       userID,
				email:        email,
				alertRequest: &AlertRequest{},
			},
			on: func(f *fields) {
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(currentAlert, nil).
					Once()
			},
			expectedErr: domain.ErrForbidden,
		},
		{
			name: "Get validation errors on update Alert",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				userID:     userID,
				email:      customerEmail,
				alertRequest: &AlertRequest{
					Config: &AlertConfigAPI{
						Condition: "percentage1",
						Currency:  "TBH",
						Metric: MetricConfig{
							Type:  "custom",
							Value: "metricId",
						},
						Operator:        "gt1",
						EvaluateForEach: "fixed:service_description",
						Attributions:    []string{attributionID, attributionIDNonExistent},
						Scopes:          []Scope{{Key: "test", Type: "test", Values: &[]string{"empty"}}},
						TimeInterval:    "hour1",
						Value:           423,
						DataSource:      "non-existent",
					},
					Name:       "Alert name",
					Recipients: []string{"name@test2.com"},
				},
			},
			expectedErr: domain.ErrValidationErrors,
			validationErr: []error{
				errormsg.ErrorMsg{Field: "recipients", Message: ErrForbiddenEmail},
				errormsg.ErrorMsg{Field: "config.condition", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.currency", Message: ErrNotSupportedCurrency},
				errormsg.ErrorMsg{Field: "config.metric.value", Message: ErrUnknown},
				errormsg.ErrorMsg{Field: "config.operator", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.timeInterval", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.attributions", Message: attributionID + " not permitted for the user, " + attributionIDNonExistent + " not found"},
				errormsg.ErrorMsg{Field: "config.scopes", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.evaluateForEach", Message: ErrNotFound},
				errormsg.ErrorMsg{Field: "config.dataSource", Message: ErrInvalidValue},
			},

			on: func(f *fields) {

				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(currentAlert, nil).
					Once()

				f.customersDAL.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()

				f.customersDAL.
					On("GetCustomer", ctx, customerID).
					Return(&common.Customer{
						Domains: []string{"test.com", "test1.com"},
					}, nil).
					Once()

				f.attributionsDAL.On("GetAttribution", ctx, attributionID).
					Return(
						&attribution.Attribution{
							Type: "custom",
							Customer: &firestore.DocumentRef{
								ID:   "otherCustomerID",
								Path: pathForCustomerRef,
							},
						}, nil).
					Once()

				f.attributionsDAL.On("GetAttribution", ctx, attributionIDNonExistent).
					Return(
						nil, errors.New("not found")).
					Once()

				f.metricsDAL.On("GetCustomMetric", ctx, "metricId").
					Return(nil, errors.New("some error")).
					Once()

				f.metadataService.On("ExternalAPIGet", mock.AnythingOfType("iface.ExternalAPIGetArgs")).
					Return(nil, errors.New("some error")).
					Once()

				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "team",
							Type: "label",
						},
					},
						nil,
					)

				f.userDAL.On("Get", ctx, userID).
					Return(&common.User{}, nil).
					Once()
			},
		},
		{
			name: "Get validation errors on update Alert 2",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				userID:     userID,
				email:      customerEmail,
				alertRequest: &AlertRequest{
					Config: &AlertConfigAPI{
						Condition: "forecast",
						Currency:  "EUR",
						Metric: MetricConfig{
							Type:  "basic",
							Value: "something",
						},
						Operator:        "lt1",
						EvaluateForEach: "fixed:service_description",
						Scopes:          []Scope{{Key: "test", Type: "test", Values: &[]string{"empty"}}},
						Attributions:    []string{attributionID},
						TimeInterval:    "hour",
						Value:           423,
					},
					Name:       "Alert name",
					Recipients: []string{"name@test2"},
				},
			},
			expectedErr: domain.ErrValidationErrors,
			validationErr: []error{
				errormsg.ErrorMsg{Field: "recipients", Message: ErrNotValidEmail},
				errormsg.ErrorMsg{Field: "config.metric.value", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.operator", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.timeInterval", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.attributions", Message: attributionID + " not permitted for the user"},
				errormsg.ErrorMsg{Field: "config.scopes", Message: ErrInvalidValue},
				errormsg.ErrorMsg{Field: "config.evaluateForEach", Message: ErrForecastMetadataIncompatible},
			},

			on: func(f *fields) {
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(currentAlert, nil).
					Once()

				f.customersDAL.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()

				f.customersDAL.
					On("GetCustomer", ctx, customerID).
					Return(&common.Customer{
						Domains: []string{"test.com", "test1.com"},
					}, nil).
					Once()

				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "team",
							Type: "label",
						},
					},
						nil,
					)

				f.attributionsDAL.On("GetAttribution", ctx, attributionID).
					Return(
						&attribution.Attribution{
							Type: "custom",
							Customer: &firestore.DocumentRef{
								ID:   "otherCustomerID",
								Path: pathForCustomerRef,
							},
						}, nil).
					Once()
			},
		},
		{
			name: "Successful Alert Name update",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				userID:     userID,
				email:      customerEmail,
				alertRequest: &AlertRequest{
					Name: "New Name",
				},
			},
			on: func(f *fields) {
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(currentAlert, nil).
					Once()

				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(updatedAlert, nil).Once()

				f.alertsDal.
					On("UpdateAlert", ctx, alertID, mock.AnythingOfType("[]firestore.Update")).
					Return(nil).
					Once()
			},
			want: updatedAlertAPI,
		},
		{
			name: "Successful Alert Update",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				userID:     userID,
				email:      customerEmail,
				alertRequest: &AlertRequest{
					Config: &AlertConfigAPI{
						Condition: "percentage-change",
						Currency:  "USD",
						Metric: MetricConfig{
							Type:  "basic",
							Value: "usage",
						},
						Operator:        "gt",
						EvaluateForEach: "fixed:service_description",
						Attributions:    []string{attributionID},
						TimeInterval:    "year",
						Value:           423,
					},
					Name:       "Alert name",
					Recipients: []string{"name@test2.com", "name@test1.slack.com"},
				},
			},
			want: updatedAlertAPI,

			on: func(f *fields) {
				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(currentAlert, nil).
					Once()

				f.alertsDal.
					On("GetAlert", ctx, alertID).
					Return(updatedAlert, nil).Once()

				f.customersDAL.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()

				f.customersDAL.
					On("GetCustomer", ctx, customerID).
					Return(&common.Customer{
						Domains: []string{"test.com", "test1.com", "test2.com"},
					}, nil).
					Once()

				f.attributionsDAL.On("GetAttribution", ctx, attributionID).
					Return(
						&attribution.Attribution{
							Type:     "custom",
							Customer: customerRef,
						}, nil).
					Once()

				f.attributionsDAL.On("GetRef", ctx, attributionID).
					Return(
						&firestore.DocumentRef{
							ID:   attributionID,
							Path: "attributions/" + attributionID,
						}, nil).
					Once()

				f.metricsDAL.On("GetCustomMetric", ctx, "metricId").
					Return(&metrics.CalculatedMetric{Customer: customerRef}, nil).
					Once()

				f.metricsDAL.On("GetRef", ctx, "metricId").
					Return(&firestore.DocumentRef{
						ID:   "metricId",
						Path: "metrics/metricId",
					}, nil).
					Once()

				f.alertsDal.
					On("UpdateAlert", ctx, alertID, mock.AnythingOfType("[]firestore.Update")).
					Return(nil).
					Once()

				f.metadataService.On("ExternalAPIGet", mock.AnythingOfType("iface.ExternalAPIGetArgs")).
					Return(&metadataIface.ExternalAPIGetRes{}, nil).Once()

				f.userDAL.On("Get", ctx, userID).
					Return(&common.User{}, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				alertsDal:       &mocks.Alerts{},
				customersDAL:    &customerMocks.Customers{},
				attributionsDAL: &attributionMocks.Attributions{},
				metricsDAL:      &metricsMocks.Metrics{},
				metadataService: &metadataServiceMock.MetadataIface{},
				userDAL:         &userMocks.IUserFirestoreDAL{},
			}

			s := &AnalyticsAlertsService{
				alertsDal:       tt.fields.alertsDal,
				customersDAL:    tt.fields.customersDAL,
				attributionsDAL: tt.fields.attributionsDAL,
				metrics:         tt.fields.metricsDAL,
				metadataService: tt.fields.metadataService,
				userDal:         tt.fields.userDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			resp := s.UpdateAlert(tt.args.ctx, alertID, ExternalAPICreateUpdateArgsReq{
				AlertRequest:   tt.args.alertRequest,
				CustomerID:     tt.args.customerID,
				UserID:         tt.args.userID,
				Email:          tt.args.email,
				IsDoitEmployee: isDoitEmployee,
			})

			assert.Equal(t, tt.expectedErr, resp.Error)
			assert.Equal(t, tt.validationErr, resp.ValidationErrors)
			assert.Equal(t, tt.want, resp.Alert)
		})
	}
}
