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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_dal_GetCustomerEngagementDetailsByCustomerID(t *testing.T) {
	testTime := time.Now()

	type fields struct {
		firestoreClientFun connection.FirestoreFromContextFun
		documentsHandler   mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		fields  fields
		want    map[string]EngagementDetails
		wantErr bool
		on      func(f *fields)
	}{
		{
			name: "get all customer engagement details",
			fields: fields{
				firestoreClientFun: func(ctx context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   mocks.DocumentsHandler{},
			},
			want: map[string]EngagementDetails{
				"customer-1": {
					CustomerID:    "customer-1",
					NotifiedDates: []time.Time{testTime},
				},
			},
			wantErr: false,
			on: func(f *fields) {
				f.documentsHandler.On("GetAll", mock.Anything).
					Return(func() []iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("ID")
						snap.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*EngagementDetails)
								*arg = EngagementDetails{
									CustomerID:    "customer-1",
									NotifiedDates: []time.Time{testTime},
								}
							}).Once()
						return []iface.DocumentSnapshot{
							snap,
						}
					}(), nil).
					Once()
			},
		},

		{
			name: "triggers not found error",
			fields: fields{
				firestoreClientFun: func(ctx context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   mocks.DocumentsHandler{},
			},
			want:    map[string]EngagementDetails{},
			wantErr: false,
			on: func(f *fields) {
				f.documentsHandler.On("GetAll", mock.Anything).
					Return(func() []iface.DocumentSnapshot {
						return []iface.DocumentSnapshot{}
					}(), status.Error(codes.NotFound, "err")).
					Once()
			},
		},

		{
			name: "triggers fetch error",
			fields: fields{
				firestoreClientFun: func(ctx context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   mocks.DocumentsHandler{},
			},
			want:    nil,
			wantErr: true,
			on: func(f *fields) {
				f.documentsHandler.On("GetAll", mock.Anything).
					Return(func() []iface.DocumentSnapshot {
						return []iface.DocumentSnapshot{}
					}(), errors.New("err")).
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

			got, err := d.GetCustomerEngagementDetailsByCustomerID(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCustomerEngagementDetailsByCustomerID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCustomerEngagementDetailsByCustomerID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dal_AddLastCustomerEngagementTime(t *testing.T) {
	notifiedTime := time.Now()

	type fields struct {
		firestoreClientFun connection.FirestoreFromContextFun
		documentsHandler   mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		on      func(f *fields)
	}{
		{
			name: "updates existing document",
			fields: fields{
				firestoreClientFun: func(ctx context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   mocks.DocumentsHandler{},
			},
			wantErr: false,
			on: func(f *fields) {
				f.documentsHandler.On("Get", mock.Anything, mock.Anything).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("ID")
						snap.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*EngagementDetails)
								*arg = EngagementDetails{
									CustomerID:    "customer-1",
									NotifiedDates: []time.Time{},
								}
							}).Once()
						return snap
					}(), nil).
					Once()
				f.documentsHandler.On("Update", mock.Anything, mock.Anything, []firestore.Update{
					{
						Path:  "NotifiedDates",
						Value: firestore.ArrayUnion(notifiedTime),
					},
				}).Return(nil, nil).Once()
			},
		},

		{
			name: "creates document if does not exist yet",
			fields: fields{
				firestoreClientFun: func(ctx context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   mocks.DocumentsHandler{},
			},
			wantErr: false,
			on: func(f *fields) {
				f.documentsHandler.On("Get", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.NotFound, "err")).
					Once()

				f.documentsHandler.On("Create", mock.Anything, mock.Anything, EngagementDetails{
					CustomerID:    "customer-1",
					NotifiedDates: []time.Time{notifiedTime},
				},
				).Return(nil, nil).Once()
			},
		},

		{
			name: "getting document creates error",
			fields: fields{
				firestoreClientFun: func(ctx context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   mocks.DocumentsHandler{},
			},
			wantErr: true,
			on: func(f *fields) {
				f.documentsHandler.On("Get", mock.Anything, mock.Anything).
					Return(nil, errors.New("err")).
					Once()
			},
		},

		{
			name: "creating document creates error",
			fields: fields{
				firestoreClientFun: func(ctx context.Context) *firestore.Client { return &firestore.Client{} },
				documentsHandler:   mocks.DocumentsHandler{},
			},
			wantErr: true,
			on: func(f *fields) {
				f.documentsHandler.On("Get", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.NotFound, "err")).
					Once()

				f.documentsHandler.On("Create", mock.Anything, mock.Anything, EngagementDetails{
					CustomerID:    "customer-1",
					NotifiedDates: []time.Time{notifiedTime},
				},
				).Return(nil, errors.New("err")).Once()
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
			if err := d.AddLastCustomerEngagementTime(context.Background(), "customer-1", notifiedTime); (err != nil) != tt.wantErr {
				t.Errorf("AddLastCustomerEngagementTime() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
