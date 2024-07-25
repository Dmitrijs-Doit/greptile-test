package dal

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/stretchr/testify/mock"
)

func Test_dal_CreateCustomerBillingDataConfig(t *testing.T) {
	testData := BillingDataConfig{
		CustomerID:     "customer-1",
		Container:      "container-1",
		Account:        "account-1",
		ResourceGroup:  "resource-group-1",
		SubscriptionID: "subscription-id-1",
		CreatedAt:      time.Now(),
	}

	type fields struct {
		firestoreClientFun connection.FirestoreFromContextFun
		documentsHandler   mocks.DocumentsHandler
	}

	type args struct {
		config BillingDataConfig
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(f *fields)
	}{
		{
			name:    "create customer billing data config",
			wantErr: false,
			args: args{
				config: testData,
			},
			fields: fields{
				firestoreClientFun: func(ctx context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   mocks.DocumentsHandler{},
			},
			on: func(f *fields) {
				f.documentsHandler.On("Create", mock.Anything, mock.Anything, testData).Return(nil, nil).Once()
			},
		},

		{
			name:    "fails to create billing data config",
			wantErr: true,
			args: args{
				config: testData,
			},
			fields: fields{
				firestoreClientFun: func(ctx context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   mocks.DocumentsHandler{},
			},
			on: func(f *fields) {
				f.documentsHandler.On("Create", mock.Anything, mock.Anything, testData).Return(nil, errors.New("err")).Once()
			},
		},
	}

	for i := range tests {
		tt := &tests[i]

		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)

			d := &dal{
				firestoreClientFun: tt.fields.firestoreClientFun,
				documentsHandler:   &tt.fields.documentsHandler,
			}
			if err := d.CreateCustomerBillingDataConfig(context.Background(), tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("CreateCustomerBillingDataConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_dal_GetCustomerBillingDataConfigs(t *testing.T) {
	testData := BillingDataConfig{
		CustomerID:     "customer-1",
		Container:      "container-1",
		Account:        "account-1",
		ResourceGroup:  "resource-group-1",
		SubscriptionID: "subscription-id-1",
		CreatedAt:      time.Now(),
	}

	type fields struct {
		firestoreClientFun connection.FirestoreFromContextFun
		documentsHandler   mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		fields  fields
		want    []BillingDataConfig
		wantErr bool
		on      func(f *fields)
	}{
		{
			name:    "get customer billing data configs",
			wantErr: false,
			want:    []BillingDataConfig{testData},
			fields: fields{
				firestoreClientFun: func(ctx context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   mocks.DocumentsHandler{},
			},
			on: func(f *fields) {
				f.documentsHandler.On("GetAll", mock.Anything).
					Return(func() []iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("ID")
						snap.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*BillingDataConfig)
								*arg = testData
							}).Once()
						return []iface.DocumentSnapshot{
							snap,
						}
					}(), nil).
					Once()

			},
		},

		{
			name:    "return error when failed to get customer billing data configs",
			wantErr: true,
			want:    []BillingDataConfig{},
			fields: fields{
				firestoreClientFun: func(ctx context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   mocks.DocumentsHandler{},
			},
			on: func(f *fields) {
				f.documentsHandler.On("GetAll", mock.Anything).
					Return(nil, errors.New("err")).Once()
			},
		},

		{
			name:    "error decoding document",
			wantErr: true,
			want:    []BillingDataConfig{},
			fields: fields{
				firestoreClientFun: func(ctx context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   mocks.DocumentsHandler{},
			},
			on: func(f *fields) {
				f.documentsHandler.On("GetAll", mock.Anything).
					Return(nil, errors.New("err")).
					Once()
			},
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)

			d := &dal{
				firestoreClientFun: tt.fields.firestoreClientFun,
				documentsHandler:   &tt.fields.documentsHandler,
			}

			got, gotErr := d.GetCustomerBillingDataConfigs(context.Background(), "customer-1")
			if tt.wantErr {
				if gotErr == nil {
					t.Errorf("GetCustomerBillingDataConfigs() error = %v, wantErr %v", gotErr, tt.wantErr)
				}

				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCustomerEngagementDetailsByCustomerID() got = %v, want %v", got, tt.want)
			}
		})
	}
}
