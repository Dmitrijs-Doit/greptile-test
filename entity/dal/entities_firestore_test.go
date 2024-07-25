package dal

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	doitfirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	testPackage "github.com/doitintl/tests"
)

var ctx = context.Background()

func TestNewFirestoreEntitiesDAL(t *testing.T) {
	_, err := NewEntitiesFirestore(ctx, common.TestProjectID)
	assert.NoError(t, err)

	d := NewEntitiesFirestoreWithClient(nil)
	assert.NotNil(t, d)
}

func TestEntitiesFirestore_GetRef(t *testing.T) {
	entityID := "asdfasdf"
	path := "projects/doitintl-cmp-dev/databases/(default)/documents/entities/asdfasdf"
	want := &firestore.DocumentRef{ID: entityID, Path: path}

	d, err := NewEntitiesFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	got := d.GetRef(ctx, entityID)
	assert.Equal(t, want.Path, got.Path)
}

func TestEntitiesFirestore_GetEntities(t *testing.T) {
	tests := []struct {
		name        string
		expectedErr error
	}{
		{
			name: "Successfully get all entities",
		},
	}

	d, err := NewEntitiesFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Entities"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.GetEntities(ctx)
			assert.Equal(t, tt.expectedErr, err)
			assert.Len(t, got, 2)
		})
	}
}

func TestEntitiesFirestore_GetEntity(t *testing.T) {
	type args struct {
		EntityID string
	}

	tests := []struct {
		name        string
		args        args
		expectedErr error
		want        *common.Entity
	}{
		{
			name: "Successfully get entity",
			args: args{EntityID: "s6g9FtA0c8ulr2EV9D3x"},
			want: &common.Entity{Name: "Rubrik, Inc"},
		},
		{
			name:        "Get invalid ID error",
			args:        args{EntityID: ""},
			expectedErr: ErrInvalidEntityID,
		},
		{
			name:        "Entity not found",
			args:        args{EntityID: "invalidEntityID"},
			expectedErr: doitfirestore.ErrNotFound,
		},
	}

	d, err := NewEntitiesFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Entities"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.GetEntity(ctx, tt.args.EntityID)
			assert.Equal(t, tt.expectedErr, err)

			if got != nil {
				assert.Equal(t, tt.want.Name, got.Name)
			}
		})
	}
}
