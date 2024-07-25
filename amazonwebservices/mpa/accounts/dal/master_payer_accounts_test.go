package dal

import (
	"context"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"

	amazonwebservices "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	testPackage "github.com/doitintl/tests"
)

const mpaID = "Gf2yZMi3c29YICoMrQ5a"
const accountNumber = "000746511415"
const mpaIDBadData = "0v4tdZOJBt9rEchjCpA2"
const accountNumberBadData = "000746511416"

func TestMasterPayerAccountDAL_GetMasterPayerAccount(t *testing.T) {
	type args struct {
		ctx           context.Context
		accountNumber string
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		args    args
		want    *amazonwebservices.MasterPayerAccount
		wantErr bool
	}{
		{
			name: "GetMasterPayerAccount",
			args: args{
				ctx:           ctx,
				accountNumber: accountNumber,
			},
			want: &amazonwebservices.MasterPayerAccount{
				AccountNumber: accountNumber,
			},
		},
		{
			name: "GetMasterPayerAccount - error - bad data",
			args: args{
				ctx:           ctx,
				accountNumber: accountNumberBadData,
			},
			wantErr: true,
		},
		{
			name: "GetMasterPayerAccount - error - account not found",
			args: args{
				ctx:           ctx,
				accountNumber: "not-found",
			},
			wantErr: true,
		},
	}

	fs, err := firestore.NewClient(context.Background(), common.ProjectID)
	if err != nil {
		t.Error(err)
	}

	d := NewMasterPayerAccountDALWithClient(fs)

	if err := testPackage.LoadTestData("MasterPayerAccounts"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.GetMasterPayerAccount(tt.args.ctx, tt.args.accountNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("MasterPayerAccountDAL.GetMasterPayerAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want != nil && !reflect.DeepEqual(got.AccountNumber, tt.want.AccountNumber) {
				t.Errorf("MasterPayerAccountDAL.GetMasterPayerAccount() = %v, want %v", got, tt.want)
				return
			}
		})
	}

	// delete test data
	iter, err := fs.Collection("app").Doc("master-payer-accounts").Collection("mpaAccounts").Documents(ctx).GetAll()
	if err != nil {
		t.Error(err)
	}

	for _, doc := range iter {
		if doc.Ref.ID == mpaID || doc.Ref.ID == mpaIDBadData {
			if _, err = doc.Ref.Delete(ctx); err != nil {
				t.Error(err)
			}
		}
	}
}
