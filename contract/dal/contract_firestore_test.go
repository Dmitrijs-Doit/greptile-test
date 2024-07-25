package dal

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/errors"
	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	testPackage "github.com/doitintl/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCancelContract(t *testing.T) {
	type fields struct {
		firestoreClientFun connection.FirestoreFromContextFun
		documentsHandler   *mocks.DocumentsHandler
	}

	tests := []struct {
		name       string
		fields     fields
		contractID string
		on         func(f *fields)
		wantErr    error
	}{
		{
			name:       "success",
			contractID: "contractID",
			fields: fields{
				firestoreClientFun: func(_ context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   &mocks.DocumentsHandler{},
			},
			on: func(f *fields) {
				f.documentsHandler.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
			},
			wantErr: nil,
		},
		{
			name:       "error",
			contractID: "contractID",
			fields: fields{
				firestoreClientFun: func(_ context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   &mocks.DocumentsHandler{},
			},
			on: func(f *fields) {
				f.documentsHandler.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("error"))
			},
			wantErr: errors.New("error"),
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)
			s := &ContractFirestore{
				firestoreClientFun: tt.fields.firestoreClientFun,
				documentsHandler:   tt.fields.documentsHandler,
			}

			err := s.CancelContract(context.Background(), tt.contractID)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestContractFirestore_DeleteContract(t *testing.T) {
	ctx := context.Background()

	type args struct {
		ctx        context.Context
		contractID string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success deleting contract",
			args: args{
				ctx:        ctx,
				contractID: "000V6tFZcnkv9gLq72sB",
			},
			wantErr: false,
		},
	}

	d, err := NewContractFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Contracts"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := d.DeleteContract(tt.args.ctx, tt.args.contractID); (err != nil) != tt.wantErr {
				t.Errorf("ContractFirestore.DeleteContract() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
