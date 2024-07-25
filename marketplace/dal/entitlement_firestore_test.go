package dal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	testPackage "github.com/doitintl/tests"
)

func TestEntitlementFirestoreDAL_NewEntitlementFirestoreDAL(t *testing.T) {
	_, err := NewEntitlementFirestoreDAL(context.Background(), "doitintl-cmp-dev")
	assert.NoError(t, err)

	d := NewEntitlementFirestoreDALWithClient(nil)
	assert.NotNil(t, d)
}

func TestEntitlementFirestoreDAL_GetEntitlement(t *testing.T) {
	ctx := context.Background()

	if err := testPackage.LoadTestData("GCPMarketplace"); err != nil {
		t.Error(err)
	}

	entitlementFirestoreDAL, _ := NewEntitlementFirestoreDAL(ctx, "doitintl-cmp-dev")

	existingEntitlementID := "b868d7a5-f97a-42a4-9795-35d34536500e"
	nonExistingEntitlementID := "6666666666-f97a-42a4-9795-6666666666"

	type args struct {
		ctx           context.Context
		entitlementID string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "get existing entitlement",
			args: args{
				ctx:           ctx,
				entitlementID: existingEntitlementID,
			},
			wantErr: false,
		},
		{
			name: "get non-existing entitlement",
			args: args{
				ctx:           ctx,
				entitlementID: nonExistingEntitlementID,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := entitlementFirestoreDAL.GetEntitlement(tt.args.ctx, tt.args.entitlementID); (err != nil) != tt.wantErr {
				t.Errorf("entitlementFirestoreDAL.GetEntitlement() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
