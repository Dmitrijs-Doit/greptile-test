package dal

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	labelsDALIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
	labelsDALMocks "github.com/doitintl/hello/scheduled-tasks/labels/dal/mocks"
	testPackage "github.com/doitintl/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var ctx = context.Background()
var alertID = "QDu1m7ouVQaOXrofAUwD"
var notificationID = "QhYxks1BkFSBYZNypWgo"
var notificationCustomerID = "ImoC9XkrutBysJvyqlBm"

func NewFirestoreWithMockLabels(labelsMock labelsDALIface.Labels) (*AlertsFirestore, error) {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	fun := func(ctx context.Context) *firestore.Client {
		return fs
	}

	return &AlertsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		labelsDal:          labelsMock,
	}, nil
}

func TestNewFirestoreAlertsDAL(t *testing.T) {
	_, err := NewAlertsFirestore(ctx, common.TestProjectID)
	assert.NoError(t, err)

	d := NewAlertsFirestoreWithClient(nil)
	assert.NotNil(t, d)
}

func TestAlertsFirestore_GetAlert(t *testing.T) {
	type args struct {
		ctx     context.Context
		alertID string
	}

	tests := []struct {
		name     string
		args     args
		wantRole collab.CollaboratorRole
		wantErr  bool
	}{
		{
			name: "err on empty alertID",
			args: args{
				ctx:     ctx,
				alertID: "",
			},
			wantErr: true,
		},
		{
			name: "err alert not found",
			args: args{
				ctx:     ctx,
				alertID: "invalidID",
			},
			wantErr: true,
		},
		{
			name: "err on snap data to alert",
			args: args{
				ctx:     ctx,
				alertID: "invalidAlertData",
			},
			wantErr: true,
		},
		{
			name: "success",
			args: args{
				ctx:     ctx,
				alertID: alertID,
			},
			wantErr:  false,
			wantRole: collab.CollaboratorRoleOwner,
		},
	}

	d, err := NewAlertsFirestore(ctx, common.ProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Alerts"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.GetAlert(tt.args.ctx, tt.args.alertID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AlertsFirestore.GetAlert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if got.Collaborators[0].Role != tt.wantRole {
					t.Errorf("AlertsFirestore.GetAlert() = %v, want %v", got, tt.wantRole)
				}
			}
		})
	}
}

func TestAlertsFirestore_Share(t *testing.T) {
	type args struct {
		ctx           context.Context
		id            string
		collaborators []collab.Collaborator
		public        *collab.PublicAccess
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
				id:  alertID,
				collaborators: []collab.Collaborator{
					{
						Email: "test1@a.com",
						Role:  collab.CollaboratorRoleOwner,
					},
				},
				public: nil,
			},
			wantErr: false,
		},
		{
			name: "error on update",
			args: args{
				ctx: ctx,
				id:  "invalid ID",
				collaborators: []collab.Collaborator{
					{
						Email: "test1@a.com",
						Role:  collab.CollaboratorRoleOwner,
					},
				},
				public: nil,
			},
			wantErr: true,
		},
	}

	if err := testPackage.LoadTestData("Alerts"); err != nil {
		t.Error(err)
	}

	d, err := NewAlertsFirestore(ctx, common.ProjectID)
	if err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := d.Share(tt.args.ctx, tt.args.id, tt.args.collaborators, tt.args.public); (err != nil) != tt.wantErr {
				t.Errorf("AlertsFirestore.Share() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAlertsFirestore_GetAlerts(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		testNum int
	}{
		{
			name:    "success",
			wantErr: false,
		},
	}

	d, err := NewAlertsFirestore(ctx, common.ProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Alerts"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alerts, err := d.GetAlerts(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("AlertsFirestore.GetAlerts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(alerts) == 0 {
					t.Errorf("AlertsFirestore.GetAlerts() = %v, want %v", alerts, "not empty")
				}
			}
		})
	}
}

func TestAlertsFirestore_DeleteAlert(t *testing.T) {
	type fields struct {
		labelsDal *labelsDALMocks.Labels
	}

	type args struct {
		ctx     context.Context
		alertID string
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
		fields      fields
		on          func(f *fields)
	}{
		{
			name: "success - delete alert",
			args: args{
				ctx:     ctx,
				alertID: alertID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.labelsDal.On("DeleteObjectWithLabels", ctx, mock.AnythingOfType("*firestore.DocumentRef")).Return(nil)
			},
		},
		{
			name: "error - missing alert id",
			args: args{
				ctx:     ctx,
				alertID: "",
			},
			wantErr:     true,
			expectedErr: domain.ErrMissingAlertID,
			on: func(f *fields) {
			},
		},
		{
			name: "error - delete object with labels error",
			args: args{
				ctx:     ctx,
				alertID: alertID,
			},
			wantErr:     true,
			expectedErr: errors.New("error"),
			on: func(f *fields) {
				f.labelsDal.On("DeleteObjectWithLabels", ctx, mock.AnythingOfType("*firestore.DocumentRef")).Return(errors.New("error"))
			},
		},
	}

	if err := testPackage.LoadTestData("Alerts"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				labelsDal: &labelsDALMocks.Labels{},
			}

			d, err := NewFirestoreWithMockLabels(tt.fields.labelsDal)
			if err != nil {
				t.Error(err)
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := d.DeleteAlert(tt.args.ctx, tt.args.alertID); (err != nil) != tt.wantErr {
				t.Errorf("AlertsFirestore.DeleteAlert() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				if err != nil && tt.expectedErr != nil {
					assert.Equal(t, err, tt.expectedErr)
				}
			}
		})
	}
}
