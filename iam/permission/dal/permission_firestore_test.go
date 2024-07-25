package dal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/common"
	testPackage "github.com/doitintl/tests"
)

var ctx = context.Background()

const (
	permissionID = "sfmBZeLN8uXWooCqJ4NO"
)

func TestNewPermissionFirestoreDAL(t *testing.T) {
	_, err := NewPermissionFirestoreDAL(ctx, common.TestProjectID)
	assert.NoError(t, err)

	d := NewPermissionFirestoreDALWithClient(nil)
	assert.NotNil(t, d)
}

func TestPermissionFirestoreDAL_Get(t *testing.T) {
	type args struct {
		ctx          context.Context
		permissionID string
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "err on empty permissionID",
			args: args{
				ctx:          ctx,
				permissionID: "",
			},
			wantErr:     true,
			expectedErr: ErrMissingPermissionID,
		},
		{
			name: "err permissionID not found",
			args: args{
				ctx:          ctx,
				permissionID: "invalidID",
			},
			wantErr:     true,
			expectedErr: ErrPermissionNotFound,
		},
		{
			name: "success getting permission",
			args: args{
				ctx:          ctx,
				permissionID: permissionID,
			},
			wantErr: false,
		},
	}

	d, err := NewPermissionFirestoreDAL(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Permissions"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := d.Get(tt.args.ctx, tt.args.permissionID)

			if (err != nil) != tt.wantErr {
				t.Errorf("PermissionFirestoreDAL.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != tt.expectedErr {
				t.Errorf("PermissionFirestoreDAL.Get() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}
