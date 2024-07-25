package dal

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	labels "github.com/doitintl/hello/scheduled-tasks/labels/domain"
	testPackage "github.com/doitintl/tests"
	"github.com/zeebo/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ctx = context.Background()

var labelID = "1JVTWRkxliSSuLZYDVJQ"

func TestLabelsFirestore_Get(t *testing.T) {
	type args struct {
		ctx     context.Context
		labelID string
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "err on empty labelID",
			args: args{
				ctx:     ctx,
				labelID: "",
			},
			wantErr:     true,
			expectedErr: labels.ErrInvalidLabelID,
		},
		{
			name: "err label not found",
			args: args{
				ctx:     ctx,
				labelID: "invalidID",
			},
			wantErr:     true,
			expectedErr: labels.ErrLabelNotFound("invalidID"),
		},
		{
			name: "success getting label",
			args: args{
				ctx:     ctx,
				labelID: labelID,
			},
			wantErr: false,
		},
	}

	d, err := NewLabelsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Labels"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := d.Get(tt.args.ctx, tt.args.labelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("LabelsFirestore.Get() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

func TestLabelsFirestore_Create(t *testing.T) {
	type args struct {
		ctx   context.Context
		label *labels.Label
	}

	d, err := NewLabelsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Labels"); err != nil {
		t.Error(err)
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "success create label",
			args: args{
				ctx:   ctx,
				label: &labels.Label{},
			},
			wantErr: false,
		},
		{
			name: "error on nil label",
			args: args{
				ctx:   ctx,
				label: nil,
			},
			wantErr:     true,
			expectedErr: labels.ErrInvalidLabel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := d.Create(tt.args.ctx, tt.args.label); (err != nil) != tt.wantErr {
				t.Errorf("LabelsFirestore.Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLabelsFirestore_Update(t *testing.T) {
	type args struct {
		ctx     context.Context
		labelID string
		updates []firestore.Update
	}

	d, err := NewLabelsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Labels"); err != nil {
		t.Error(err)
	}

	var (
		existingLabelID = "1JVTWRkxliSSuLZYDVJQ"
		expectedLabel   = labels.Label{
			Name:  "name",
			Color: labels.Green,
		}
		updates = []firestore.Update{
			{Path: "name", Value: expectedLabel.Name},
			{Path: "color", Value: expectedLabel.Color},
		}
	)

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "success update label",
			args: args{
				ctx:     ctx,
				labelID: existingLabelID,
				updates: updates,
			},
			wantErr: false,
		},
		{
			name: "error if label doesn't exist",
			args: args{
				ctx:     ctx,
				labelID: "not-existing-label",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedLabel, err := d.Update(tt.args.ctx, tt.args.labelID, tt.args.updates)

			if (updatedLabel != nil) && tt.wantErr == false {
				assert.Equal(t, updatedLabel.Name, expectedLabel.Name)
				assert.Equal(t, updatedLabel.Color, expectedLabel.Color)
			} else if (err != nil) != tt.wantErr {
				t.Errorf("LabelsFirestore.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLabelsFirestore_GetLabels(t *testing.T) {
	type args struct {
		ctx      context.Context
		labelIDs []string
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "err on empty labelIDs",
			args: args{
				ctx:      ctx,
				labelIDs: []string{},
			},
			wantErr:     true,
			expectedErr: labels.ErrInvalidLabelID,
		},
		{
			name: "err on empty labelID",
			args: args{
				ctx:      ctx,
				labelIDs: []string{""},
			},
			wantErr:     true,
			expectedErr: labels.ErrInvalidLabelID,
		},
		{
			name: "err label not found",
			args: args{
				ctx:      ctx,
				labelIDs: []string{"invalidID"},
			},
			wantErr:     true,
			expectedErr: labels.ErrLabelNotFound("invalidID"),
		},
		{
			name: "success getting labels",
			args: args{
				ctx:      ctx,
				labelIDs: []string{"1JVTWRkxliSSuLZYDVJQ", "DdGDrqXMZzzrIacsEBsl"},
			},
			wantErr: false,
		},
	}

	d, err := NewLabelsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Labels"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := d.GetLabels(tt.args.ctx, tt.args.labelIDs)
			if (err != nil) != tt.wantErr {
				t.Errorf("LabelsFirestore.GetLabels() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

func TestLabelsFirestore_GetObjectLabels(t *testing.T) {
	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "success getting alert labels",
			args: args{
				ctx: ctx,
			},
			wantErr: false,
		},
	}

	d, err := NewLabelsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Alerts"); err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Labels"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alertRef := d.firestoreClientFun(ctx).Collection("cloudAnalytics/alerts/cloudAnalyticsAlerts").Doc("QDu1m7ouVQaOXrofAUwD")

			_, err := d.GetObjectLabels(tt.args.ctx, alertRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("LabelsFirestore.GetObjectLabels() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

func TestLabelsFirestore_DeleteObjectWithLabels(t *testing.T) {
	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success deleting object with labels",
			args: args{
				ctx,
			},
			wantErr: false,
		},
	}

	d, err := NewLabelsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Alerts"); err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Labels"); err != nil {
		t.Error(err)
	}

	label, err := d.Get(ctx, "DdGDrqXMZzzrIacsEBsl")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 2, len(label.Objects))

	deletedObjRef := d.firestoreClientFun(ctx).Collection("cloudAnalytics/alerts/cloudAnalyticsAlerts").Doc("QDu1m7ouVQaOXrofAUwD")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := d.DeleteObjectWithLabels(tt.args.ctx, deletedObjRef); (err != nil) != tt.wantErr {
				t.Errorf("LabelsFirestore.DeleteObjectWithLabels() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				_, err := deletedObjRef.Get(ctx)
				assert.Equal(t, codes.NotFound, status.Code(err))

				label, err := d.Get(ctx, "DdGDrqXMZzzrIacsEBsl")
				if err != nil {
					t.Error(err)
				}

				assert.Equal(t, 1, len(label.Objects))
			}
		})
	}
}

func TestLabelsFirestore_DeleteManyObjectsWithLabels(t *testing.T) {
	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success deleting many objects with labels",
			args: args{
				ctx,
			},
			wantErr: false,
		},
	}

	d, err := NewLabelsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Budgets"); err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Alerts"); err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Labels"); err != nil {
		t.Error(err)
	}

	label, err := d.Get(ctx, "DdGDrqXMZzzrIacsEBsl")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 2, len(label.Objects))

	deletedAlertRef := d.firestoreClientFun(ctx).Collection("cloudAnalytics/alerts/cloudAnalyticsAlerts").Doc("QDu1m7ouVQaOXrofAUwD")
	deletedBudgetRef := d.firestoreClientFun(ctx).Collection("cloudAnalytics/budgets/cloudAnalyticsBudgets").Doc("PnOD7lsJWD2IseckPdHM")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := d.DeleteManyObjectsWithLabels(tt.args.ctx, []*firestore.DocumentRef{deletedAlertRef, deletedBudgetRef}); (err != nil) != tt.wantErr {
				t.Errorf("LabelsFirestore.DeleteObjectWithLabels() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				_, err := deletedAlertRef.Get(ctx)
				assert.Equal(t, codes.NotFound, status.Code(err))

				_, err = deletedBudgetRef.Get(ctx)
				assert.Equal(t, codes.NotFound, status.Code(err))

				label, err := d.Get(ctx, "DdGDrqXMZzzrIacsEBsl")
				if err != nil {
					t.Error(err)
				}

				assert.Equal(t, 0, len(label.Objects))
			}
		})
	}
}
