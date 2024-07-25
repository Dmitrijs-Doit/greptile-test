package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/zeebo/assert"

	doitFirestoreMocks "github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDalMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	labelsMocks "github.com/doitintl/hello/scheduled-tasks/labels/dal/mocks"
	labels "github.com/doitintl/hello/scheduled-tasks/labels/domain"
)

type labelsFields struct {
	labelsDal     *labelsMocks.Labels
	customerDal   *customerDalMocks.Customers
	batchProvider *doitFirestoreMocks.BatchProvider
}

func TestLabelsService_CreateLabel(t *testing.T) {
	type args struct {
		req CreateLabelRequest
	}

	var (
		name         = "name"
		color        = labels.LightBlue
		customerID   = "customer-id"
		userEmail    = "email@test.com"
		validRequest = CreateLabelRequest{
			Name:       name,
			Color:      color,
			CustomerID: customerID,
			UserEmail:  userEmail,
		}
		customerRef = firestore.DocumentRef{
			ID: customerID,
		}
		expectedLabel = labels.Label{}
		testError     = errors.New("test error")
	)

	ctx := context.Background()

	tests := []struct {
		name           string
		fields         labelsFields
		args           args
		wantErr        bool
		expectedErr    error
		expectedResult *labels.Label
		on             func(context.Context, *labelsFields)
	}{
		{
			name: "success - create label ",
			args: args{
				req: validRequest,
			},
			wantErr: false,
			on: func(ctx context.Context, f *labelsFields) {
				f.customerDal.On("GetCustomer", ctx, customerID).Return(&common.Customer{
					Snapshot: &firestore.DocumentSnapshot{
						Ref: &customerRef,
					},
				}, nil)
				f.labelsDal.On("Create", ctx, &labels.Label{
					Name:      name,
					Color:     color,
					CreatedBy: userEmail,
					Customer:  &customerRef,
				}).Return(&expectedLabel, nil)
			},
			expectedResult: &expectedLabel,
		},
		{
			name: "error - get customer error",
			args: args{
				req: validRequest,
			},
			wantErr:     true,
			expectedErr: testError,
			on: func(ctx context.Context, f *labelsFields) {
				f.customerDal.On("GetCustomer", ctx, customerID).Return(nil, testError)
			},
			expectedResult: nil,
		},
		{
			name: "error - create label",
			args: args{
				req: validRequest,
			},
			wantErr:     true,
			expectedErr: testError,
			on: func(ctx context.Context, f *labelsFields) {
				f.customerDal.On("GetCustomer", ctx, customerID).Return(&common.Customer{
					Snapshot: &firestore.DocumentSnapshot{
						Ref: &customerRef,
					},
				}, nil)
				f.labelsDal.On("Create", ctx, &labels.Label{
					Name:      name,
					Color:     color,
					CreatedBy: userEmail,
					Customer:  &customerRef,
				}).Return(nil, testError)
			},
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = labelsFields{
				labelsDal:   &labelsMocks.Labels{},
				customerDal: &customerDalMocks.Customers{},
			}

			s := &LabelsService{
				labelsDal:   tt.fields.labelsDal,
				customerDal: tt.fields.customerDal,
			}

			if tt.on != nil {
				tt.on(ctx, &tt.fields)
			}

			got, err := s.CreateLabel(ctx, tt.args.req)
			assert.Equal(t, tt.expectedResult, got)

			if (err != nil) != tt.wantErr {
				t.Errorf("LabelsService.CreateLabel() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

func TestLabelsService_DeleteLabel(t *testing.T) {
	type args struct {
		labelID string
	}

	var (
		labelID       = "label-id"
		testError     = errors.New("test error")
		objectRef1    = &firestore.DocumentRef{ID: "object1"}
		expectedLabel = labels.Label{
			Ref: &firestore.DocumentRef{
				ID: labelID,
			},
			Objects: []*firestore.DocumentRef{objectRef1},
		}
		otherLabelRef1       = &firestore.DocumentRef{ID: "otherLabelRef1"}
		otherLabelRef2       = &firestore.DocumentRef{ID: "otherLabelRef2"}
		expectedObjectLabels = []*firestore.DocumentRef{
			{ID: labelID}, otherLabelRef1, otherLabelRef2,
		}
	)

	ctx := context.Background()

	tests := []struct {
		name        string
		fields      labelsFields
		args        args
		wantErr     bool
		expectedErr error
		on          func(context.Context, *labelsFields)
	}{
		{
			name: "success - delete label",
			args: args{
				labelID,
			},
			wantErr: false,
			on: func(ctx context.Context, f *labelsFields) {
				wb := &doitFirestoreMocks.Batch{}
				f.labelsDal.On("Get", ctx, labelID).Return(&expectedLabel, nil)
				f.batchProvider.On("ProvideWithThreshold", ctx, len(expectedLabel.Objects)+1).Return(wb)
				wb.On("Delete", ctx, expectedLabel.Ref).Return(nil)
				f.labelsDal.On("GetObjectLabels", ctx, objectRef1).Return(expectedObjectLabels, nil)
				wb.On("Update", ctx, objectRef1, []firestore.Update{{Path: "labels", Value: []*firestore.DocumentRef{otherLabelRef1, otherLabelRef2}}}).Return(nil)
				wb.On("Commit", ctx).Return(nil)
			},
		},
		{
			name: "error - get label",
			args: args{
				labelID,
			},
			expectedErr: testError,
			wantErr:     true,
			on: func(ctx context.Context, f *labelsFields) {
				f.labelsDal.On("Get", ctx, labelID).Return(nil, testError)
			},
		},
		{
			name: "error - batch writer delete label",
			args: args{
				labelID,
			},
			wantErr:     true,
			expectedErr: testError,
			on: func(ctx context.Context, f *labelsFields) {
				wb := &doitFirestoreMocks.Batch{}
				f.labelsDal.On("Get", ctx, labelID).Return(&expectedLabel, nil)
				f.batchProvider.On("ProvideWithThreshold", ctx, len(expectedLabel.Objects)+1).Return(wb)
				wb.On("Delete", ctx, expectedLabel.Ref).Return(testError)
			},
		},
		{
			name: "error - batch writer update object labels",
			args: args{
				labelID,
			},
			wantErr:     true,
			expectedErr: testError,
			on: func(ctx context.Context, f *labelsFields) {
				wb := &doitFirestoreMocks.Batch{}
				f.labelsDal.On("Get", ctx, labelID).Return(&expectedLabel, nil)
				f.batchProvider.On("ProvideWithThreshold", ctx, len(expectedLabel.Objects)+1).Return(wb)
				wb.On("Delete", ctx, expectedLabel.Ref).Return(nil)
				f.labelsDal.On("GetObjectLabels", ctx, objectRef1).Return(expectedObjectLabels, nil)
				wb.On("Update", ctx, objectRef1, []firestore.Update{{Path: "labels", Value: []*firestore.DocumentRef{otherLabelRef1, otherLabelRef2}}}).Return(testError)
			},
		},
		{
			name: "error - batch writer commit",
			args: args{
				labelID,
			},
			wantErr:     true,
			expectedErr: testError,
			on: func(ctx context.Context, f *labelsFields) {
				wb := &doitFirestoreMocks.Batch{}
				f.labelsDal.On("Get", ctx, labelID).Return(&expectedLabel, nil)
				f.batchProvider.On("ProvideWithThreshold", ctx, len(expectedLabel.Objects)+1).Return(wb)
				wb.On("Delete", ctx, expectedLabel.Ref).Return(nil)
				f.labelsDal.On("GetObjectLabels", ctx, objectRef1).Return(expectedObjectLabels, nil)
				wb.On("Update", ctx, objectRef1, []firestore.Update{{Path: "labels", Value: []*firestore.DocumentRef{otherLabelRef1, otherLabelRef2}}}).Return(nil)
				wb.On("Commit", ctx).Return(testError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = labelsFields{
				labelsDal:     &labelsMocks.Labels{},
				batchProvider: &doitFirestoreMocks.BatchProvider{},
			}

			s := &LabelsService{
				labelsDal:     tt.fields.labelsDal,
				batchProvider: tt.fields.batchProvider,
			}

			if tt.on != nil {
				tt.on(ctx, &tt.fields)
			}

			err := s.DeleteLabel(ctx, tt.args.labelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("LabelsService.DeleteLabel() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

func TestLabelsService_UpdateLabel(t *testing.T) {
	type args struct {
		req UpdateLabelRequest
	}

	var (
		name         = "name"
		color        = labels.LightBlue
		labelID      = "label-id"
		validRequest = UpdateLabelRequest{
			Name:    name,
			Color:   color,
			LabelID: labelID,
		}
		validRequestJustName = UpdateLabelRequest{
			Name:    name,
			LabelID: labelID,
		}
		validRequestJustColor = UpdateLabelRequest{
			Color:   color,
			LabelID: labelID,
		}
		expectedLabel = labels.Label{}
		nameUpdate    = firestore.Update{
			Path:  "name",
			Value: name,
		}
		colorUpdate = firestore.Update{
			Path:  "color",
			Value: color,
		}
	)

	ctx := context.Background()

	tests := []struct {
		name           string
		fields         labelsFields
		args           args
		wantErr        bool
		expectedErr    error
		expectedResult *labels.Label
		on             func(context.Context, *labelsFields)
	}{
		{
			name: "success - update label",
			args: args{
				req: validRequest,
			},
			wantErr: false,
			on: func(ctx context.Context, f *labelsFields) {
				f.labelsDal.On("Update", ctx, labelID, []firestore.Update{nameUpdate, colorUpdate}).Return(&expectedLabel, nil)
			},
			expectedResult: &expectedLabel,
		},
		{
			name: "success - update label just name",
			args: args{
				req: validRequestJustName,
			},
			wantErr: false,
			on: func(ctx context.Context, f *labelsFields) {
				f.labelsDal.On("Update", ctx, labelID, []firestore.Update{nameUpdate}).Return(&expectedLabel, nil)
			},
			expectedResult: &expectedLabel,
		},
		{
			name: "success - update label just color",
			args: args{
				req: validRequestJustColor,
			},
			wantErr: false,
			on: func(ctx context.Context, f *labelsFields) {
				f.labelsDal.On("Update", ctx, labelID, []firestore.Update{colorUpdate}).Return(&expectedLabel, nil)
			},
			expectedResult: &expectedLabel,
		},
		{
			name: "error updating label",
			args: args{
				req: validRequest,
			},
			wantErr:     true,
			expectedErr: errors.New("test error"),
			on: func(ctx context.Context, f *labelsFields) {
				f.labelsDal.On("Update", ctx, labelID, []firestore.Update{nameUpdate, colorUpdate}).Return(nil, errors.New("test error"))
			},
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = labelsFields{
				labelsDal:   &labelsMocks.Labels{},
				customerDal: &customerDalMocks.Customers{},
			}

			s := &LabelsService{
				labelsDal:   tt.fields.labelsDal,
				customerDal: tt.fields.customerDal,
			}

			if tt.on != nil {
				tt.on(ctx, &tt.fields)
			}

			got, err := s.UpdateLabel(ctx, tt.args.req)
			assert.Equal(t, tt.expectedResult, got)

			if (err != nil) != tt.wantErr {
				t.Errorf("LabelsService.UpdateLabel() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}
