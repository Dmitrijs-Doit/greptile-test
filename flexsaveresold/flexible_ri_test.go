package flexsaveresold

import (
	"context"
	"errors"
	"strings"
	"testing"

	mpaMocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	amazonwebservicesDomain "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/testutils"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

func TestFetchInstanceFamilyAndFactorsReturnsCorrectValues(t *testing.T) {
	instanceFamily, normalizedFactor, err := InstanceFamilyNormalizationFactor("t3.2xlarge")
	assert.NoError(t, err)
	assert.Equal(t, "t3", instanceFamily)
	assert.Equal(t, float64(16), normalizedFactor)

	instanceFamily, normalizedFactor, err = InstanceFamilyNormalizationFactor("m5.xlarge")
	assert.NoError(t, err)
	assert.Equal(t, "m5", instanceFamily)
	assert.Equal(t, float64(8), normalizedFactor)

	instanceFamily, normalizedFactor, err = InstanceFamilyNormalizationFactor("c6gd.metal")
	assert.NoError(t, err)
	assert.Equal(t, "c6gd", instanceFamily)
	assert.Equal(t, float64(128), normalizedFactor)

	instanceFamily, normalizedFactor, err = InstanceFamilyNormalizationFactor(" c6gd.metal  	")
	assert.NoError(t, err)
	assert.Equal(t, "c6gd", instanceFamily)
	assert.Equal(t, float64(128), normalizedFactor)

	someRandomValue := strings.ToLower("randomValue" + randomNumberString())

	orderInstanceTypeTestString := someRandomValue + ".xlarge"
	instanceFamily, normalizedFactor, err = InstanceFamilyNormalizationFactor(orderInstanceTypeTestString)
	assert.NoError(t, err)
	assert.Equal(t, someRandomValue, instanceFamily)
	assert.Equal(t, float64(8), normalizedFactor)

	orderInstanceTypeTestString = "t3." + someRandomValue
	instanceFamily, normalizedFactor, err = InstanceFamilyNormalizationFactor(orderInstanceTypeTestString)
	assert.Equal(t, NewServiceError("invalid instance size: "+orderInstanceTypeTestString, web.ErrBadRequest), NewServiceError(err.Error(), web.ErrBadRequest))
	assert.Equal(t, "invalid instance size: "+orderInstanceTypeTestString, err.Error())
	assert.Equal(t, "t3", instanceFamily)
	assert.Equal(t, float64(0), normalizedFactor)

	orderInstanceTypeTestString = "c6gd." + someRandomValue
	instanceFamily, normalizedFactor, err = InstanceFamilyNormalizationFactor(orderInstanceTypeTestString)
	assert.Equal(t, NewServiceError("invalid instance size: "+orderInstanceTypeTestString, web.ErrBadRequest), NewServiceError(err.Error(), web.ErrBadRequest))
	assert.Equal(t, "invalid instance size: "+orderInstanceTypeTestString, err.Error())
	assert.Equal(t, "c6gd", instanceFamily)
	assert.Equal(t, float64(0), normalizedFactor)

	orderInstanceTypeTestString = "t3" + "dotMissing" + someRandomValue
	instanceFamily, normalizedFactor, err = InstanceFamilyNormalizationFactor(orderInstanceTypeTestString)
	assert.Equal(t, NewServiceError("invalid instance type: "+orderInstanceTypeTestString, web.ErrBadRequest), NewServiceError(err.Error(), web.ErrBadRequest))
	assert.Equal(t, "invalid instance type: "+orderInstanceTypeTestString, err.Error())
	assert.Equal(t, "", instanceFamily)
	assert.Equal(t, float64(0), normalizedFactor)

	orderInstanceTypeTestString = someRandomValue + ".metal"
	instanceFamily, normalizedFactor, err = InstanceFamilyNormalizationFactor(orderInstanceTypeTestString)
	assert.Equal(t, errors.New("invalid instance size: "+orderInstanceTypeTestString), err) // must return error
	assert.Equal(t, someRandomValue, instanceFamily)
	assert.Equal(t, float64(0), normalizedFactor)

	var nilString string
	instanceFamily, normalizedFactor, err = InstanceFamilyNormalizationFactor(nilString)
	assert.Equal(t, errors.New("invalid instance type: "+nilString), err) // must return error
	assert.Equal(t, "", instanceFamily)
	assert.Equal(t, float64(0), normalizedFactor)
}

func TestDedicatedGetPayerDetails(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		mpaDAL *mpaMocks.MasterPayerAccounts
	}

	mpaID := "ms_payer"

	masterPayerAccountShared := &amazonwebservicesDomain.MasterPayerAccount{
		TenancyType: "shared",
	}

	masterPayerAccount := &amazonwebservicesDomain.MasterPayerAccount{}

	tests := []struct {
		name    string
		wantErr error
		want    bool
		fields  fields
		orgInfo *pkg.OrganizationInfo
		on      func(*fields)
	}{
		{
			name: "returns true because MPA is shared (IDs do not match)",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", testutils.ContextBackgroundMock, mpaID).Return(masterPayerAccountShared, nil)
			},
			orgInfo: &pkg.OrganizationInfo{
				PayerAccount: &domain.PayerAccount{
					AccountID: "1",
				},
			},
			want: true,
		},
		{
			name: "returns true because IDs match",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", testutils.ContextBackgroundMock, mpaID).Return(masterPayerAccount, nil)
			},
			orgInfo: &pkg.OrganizationInfo{
				PayerAccount: &domain.PayerAccount{
					AccountID: "ms_payer",
				},
			},
			want: true,
		},
		{
			name: "returns false because IDs do not match and MPA is not shared",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", testutils.ContextBackgroundMock, mpaID).Return(masterPayerAccount, nil)
			},
			orgInfo: &pkg.OrganizationInfo{
				PayerAccount: &domain.PayerAccount{
					AccountID: "1",
				},
			},
			want: false,
		},
		{
			name: "returns false because payer account is nil",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", testutils.ContextBackgroundMock, mpaID).Return(masterPayerAccountShared, nil)
			},
			orgInfo: &pkg.OrganizationInfo{},
			want:    false,
		},
		{
			name: "MPA fetch error",
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccount", testutils.ContextBackgroundMock, mpaID).Return(masterPayerAccountShared, errors.New("something has gone terribly wrong!"))
			},
			orgInfo: &pkg.OrganizationInfo{},
			want:    false,
			wantErr: errors.New("something has gone terribly wrong!"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				mpaDAL: &mpaMocks.MasterPayerAccounts{},
			}

			s := &Service{
				mpaDAL: f.mpaDAL,
			}

			if tt.on != nil {
				tt.on(f)
			}

			want, err := s.isValidPayerAccount(ctx, mpaID, tt.orgInfo)

			if err != nil {
				expectedError := tt.wantErr
				if err.Error() != expectedError.Error() {
					t.Errorf("isValidPayerAccount() error = %v, wantErr %v", err, &expectedError)
					return
				}
			}

			assert.Equalf(t, tt.want, want, "isValidPayerAccount() got = %v, want %v", want, tt.want)
		})
	}
}
