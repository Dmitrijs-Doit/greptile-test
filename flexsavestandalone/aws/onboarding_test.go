package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/firestore/pkg"
)

func TestAwsStandaloneService_DeleteAWSEstimation(t *testing.T) {
	contextMock := mock.MatchedBy(func(_ context.Context) bool { return true })

	type fields struct {
		flexsaveStandaloneDAL mocks.FlexsaveStandalone
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr bool
	}{
		{
			name: "service returns error",
			on: func(f *fields) {
				f.flexsaveStandaloneDAL.On("GetAWSAccount",
					contextMock,
					"111",
					"222").
					Return(nil, errors.New("err"))
			},
			wantErr: true,
		},

		{
			name: "account not found",
			on: func(f *fields) {
				f.flexsaveStandaloneDAL.On("GetAWSAccount",
					contextMock,
					"111",
					"222").
					Return(nil, nil)
			},
			wantErr: true,
		},

		{
			name: "account already completed",
			on: func(f *fields) {
				onboarding := pkg.AWSStandaloneOnboarding{
					BaseStandaloneOnboarding: pkg.BaseStandaloneOnboarding{
						Completed: true,
					},
				}
				f.flexsaveStandaloneDAL.On("GetAWSAccount",
					contextMock,
					"111",
					"222").
					Return(&onboarding, nil)
			},
			wantErr: true,
		},

		{
			name: "delete is called with no error",
			on: func(f *fields) {
				onboarding := pkg.AWSStandaloneOnboarding{}
				f.flexsaveStandaloneDAL.On("GetAWSAccount",
					contextMock,
					"111",
					"222").
					Return(&onboarding, nil)
				f.flexsaveStandaloneDAL.On("DeleteAWSAccount", contextMock, "amazon-web-services-111", "222").
					Return(nil)
			},
			wantErr: false,
		},

		{
			name: "delete is called with error",
			on: func(f *fields) {
				onboarding := pkg.AWSStandaloneOnboarding{}
				f.flexsaveStandaloneDAL.On("GetAWSAccount",
					contextMock,
					"111",
					"222").
					Return(&onboarding, nil)
				f.flexsaveStandaloneDAL.On("DeleteAWSAccount", contextMock, "amazon-web-services-111", "222").
					Return(errors.New("zip zag"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &AwsStandaloneService{
				flexsaveStandaloneDAL: &fields.flexsaveStandaloneDAL,
			}

			err := s.DeleteAWSEstimation(context.Background(), "111", "222")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
