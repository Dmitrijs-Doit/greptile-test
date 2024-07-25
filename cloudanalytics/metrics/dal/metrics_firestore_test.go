package dal

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	labelsDALIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
	labelsDALMocks "github.com/doitintl/hello/scheduled-tasks/labels/dal/mocks"
	testPackage "github.com/doitintl/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const metricName = "EC2 RI Coverage"
const metricID = "P3SH89nIVuTTYZo9oQAL"

var ctx = context.Background()

func NewFirestoreWithMockLabels(labelsMock labelsDALIface.Labels) (*MetricsFirestore, error) {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	fun := func(ctx context.Context) *firestore.Client {
		return fs
	}

	return &MetricsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		labelsDal:          labelsMock,
	}, nil
}

func TestMetricsFirestore_TestNewMetricsFirestore(t *testing.T) {
	ctx := context.Background()
	_, err := NewMetricsFirestore(ctx, common.ProjectID)
	assert.NoError(t, err)

	d := NewMetricsFirestoreWithClient(nil)
	assert.NotNil(t, d)
}

func TestMetricsFirestore_TestGetRef(t *testing.T) {
	ctx := context.Background()
	d, err := NewMetricsFirestore(ctx, common.ProjectID)
	assert.NoError(t, err)

	ref := d.GetRef(ctx, metricID)
	assert.NotNil(t, ref)
}

func TestMetricsFirestore_GetCustomMetric(t *testing.T) {
	type args struct {
		ctx                context.Context
		calculatedMetricID string
	}

	ctx := context.Background()

	tests := []struct {
		name           string
		args           args
		wantMetricName string
		wantErr        bool
	}{
		{
			name: "Get custom metric",
			args: args{
				ctx:                ctx,
				calculatedMetricID: metricID,
			},
			wantMetricName: metricName,
		},
		{
			name: "Custom metric not found error",
			args: args{
				ctx:                ctx,
				calculatedMetricID: "wrongID",
			},
			wantErr: true,
		},
	}

	d, err := NewMetricsFirestore(ctx, common.ProjectID)
	assert.NoError(t, err)

	if err := testPackage.LoadTestData("CustomMetrics"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.GetCustomMetric(tt.args.ctx, tt.args.calculatedMetricID)
			if (err != nil) != tt.wantErr {
				t.Errorf("MetricsFirestore.GetCustomMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != nil && got.Name != tt.wantMetricName {
				t.Errorf("MetricsFirestore.GetCustomMetric() = %v, want %v", got, tt.wantMetricName)
			}
		})
	}
}

func TestMetricsFirestore_Exists(t *testing.T) {
	type args struct {
		ctx                context.Context
		calculatedMetricID string
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Custom metric exists",
			args: args{
				ctx:                ctx,
				calculatedMetricID: metricID,
			},
			want: true,
		},
		{
			name: "Custom metric not found",
			args: args{
				ctx:                ctx,
				calculatedMetricID: "wrongID",
			},
		},
	}

	d, err := NewMetricsFirestore(ctx, common.ProjectID)
	assert.NoError(t, err)

	if err := testPackage.LoadTestData("CustomMetrics"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.Exists(tt.args.ctx, tt.args.calculatedMetricID)
			if (err != nil) != tt.wantErr {
				t.Errorf("MetricsFirestore.GetCustomMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMetricsFirestore_DeleteMany(t *testing.T) {
	type fields struct {
		labelsDal *labelsDALMocks.Labels
	}

	type args struct {
		ctx context.Context
		IDs []string
	}

	ctx := context.Background()

	tests := []struct {
		name           string
		args           args
		wantMetricName string
		wantErr        bool
		fields         fields
		on             func(f *fields)
	}{
		{
			name: "Success delete many",
			args: args{
				ctx: ctx,
				IDs: []string{"P3SH89nIVuTTYZo9oQAL", "00xe2ZT0uHBht1n3Q0Tm"},
			},
			on: func(f *fields) {
				f.labelsDal.On("DeleteManyObjectsWithLabels", ctx, mock.AnythingOfType("[]*firestore.DocumentRef")).Return(nil)
			},
		},
		{
			name: "Error - delete objects with labels error",
			args: args{
				ctx: ctx,
				IDs: []string{"P3SH89nIVuTTYZo9oQAL", "00xe2ZT0uHBht1n3Q0Tm"},
			},
			wantErr: true,
			on: func(f *fields) {
				f.labelsDal.On("DeleteManyObjectsWithLabels", ctx, mock.AnythingOfType("[]*firestore.DocumentRef")).Return(errors.New("error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				labelsDal: &labelsDALMocks.Labels{},
			}

			d, err := NewFirestoreWithMockLabels(tt.fields.labelsDal)
			assert.NoError(t, err)

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err = d.DeleteMany(tt.args.ctx, tt.args.IDs)
			if (err != nil) != tt.wantErr {
				t.Errorf("MetricsFirestore.DeleteMany() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
