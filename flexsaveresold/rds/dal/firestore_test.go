package firestore

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/mocks"
	rdsIface "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/iface"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func Test_dal_collection(t *testing.T) {
	type fields struct {
		firestoreClient  firestore.Client
		documentsHandler mocks.DocumentsHandler
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "returns correct path",
			fields: fields{},
			want:   "projects//databases//documents/integrations/flexsave/configuration-rds",
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			d := &dal{
				firestoreClient:  &tt.fields.firestoreClient,
				documentsHandler: &tt.fields.documentsHandler,
			}
			assert.Equalf(t, tt.want, d.collection().Path, "collection()")
		})
	}
}

func Test_dal_Get(t *testing.T) {
	type fields struct {
		firestoreClient  firestore.Client
		documentsHandler mocks.DocumentsHandler
	}

	wantedTime := time.Time{}

	tests := []struct {
		name    string
		fields  fields
		want    *rdsIface.FlexsaveRDSCache
		wantErr error
		on      func(f *fields)
	}{
		{
			on: func(f *fields) {
				f.documentsHandler.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*rdsIface.FlexsaveRDSCache)
								arg.TimeEnabled = &wantedTime
							}).Once()

						return snap
					}(), nil).Once()

			},
			name: "returns data",
			want: &rdsIface.FlexsaveRDSCache{
				TimeEnabled: &wantedTime,
			},
			wantErr: nil,
		},
		{
			on: func(f *fields) {
				f.documentsHandler.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(nil, errors.New("we have a problemo")).Once()

			},
			name:    "firestore returns error",
			want:    nil,
			wantErr: errors.New("we have a problemo"),
		},

		{
			on: func(f *fields) {
				f.documentsHandler.On("Get", testutils.ContextBackgroundMock, mock.Anything).
					Return(nil, status.Errorf(codes.NotFound, "The item was not found")).Once()
			},
			name:    "firestore returns not found",
			want:    nil,
			wantErr: errors.New("not found"),
		},

		{
			on: func(f *fields) {
				f.documentsHandler.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo", mock.Anything).Return(errors.New("problemo"))
						return snap
					}(), nil).Once()

			},
			name:    "dataTo returns error",
			want:    nil,
			wantErr: errors.New("problemo"),
		},

		{
			on: func(f *fields) {
				f.documentsHandler.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo", mock.Anything).Return(errors.New("problemo"))
						return snap
					}(), nil).Once()

			},
			name:    "dataTo returns error",
			want:    nil,
			wantErr: errors.New("problemo"),
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)
			d := &dal{
				firestoreClient:  &tt.fields.firestoreClient,
				documentsHandler: &tt.fields.documentsHandler,
			}

			result, err := d.Get(context.Background(), "kawa")
			assert.Equal(t, tt.want, result)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_dal_Update(t *testing.T) {
	type fields struct {
		firestoreClient  firestore.Client
		documentsHandler mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr error
		on      func(f *fields)
	}{
		{
			on: func(f *fields) {
				f.documentsHandler.On("Set", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef"), map[string]interface{}{
					"timeEnabled": nil,
				}, firestore.MergeAll).Return(nil, nil).Once()
			},
			name:    "updates correctly",
			wantErr: nil,
		},
		{
			on: func(f *fields) {
				f.documentsHandler.On("Set", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef"), map[string]interface{}{
					"timeEnabled": nil,
				}, firestore.MergeAll).Return(nil, errors.New("asterix")).Once()
			},
			name:    "returns error",
			wantErr: errors.New("asterix"),
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)
			d := &dal{
				firestoreClient:  &tt.fields.firestoreClient,
				documentsHandler: &tt.fields.documentsHandler,
			}

			err := d.Update(context.Background(), "kawa", map[string]interface{}{
				"timeEnabled": nil,
			})
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_dal_AddReasonCantEnable(t *testing.T) {
	type fields struct {
		firestoreClient  firestore.Client
		documentsHandler mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr error
		on      func(f *fields)
	}{
		{
			on: func(f *fields) {
				f.documentsHandler.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*rdsIface.FlexsaveRDSCache)
								arg.ReasonCantEnable = []rdsIface.FlexsaveRDSReasonCantEnable{"foo"}
							}).Once()

						return snap
					}(), nil).Once()

				f.documentsHandler.On(
					"Set", testutils.ContextBackgroundMock, mock.Anything,
					map[string]interface{}{"reasonCantEnable": []rdsIface.FlexsaveRDSReasonCantEnable{"foo", "no_billing_table"}}, firestore.MergeAll).
					Return(nil, nil).Once()
			},
			name:    "updates correctly",
			wantErr: nil,
		},
		{
			on: func(f *fields) {
				f.documentsHandler.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(nil, errors.New("we have a problemo")).Once()

			},
			name:    "returns error",
			wantErr: errors.New("we have a problemo"),
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)
			d := &dal{
				firestoreClient:  &tt.fields.firestoreClient,
				documentsHandler: &tt.fields.documentsHandler,
			}

			err := d.AddReasonCantEnable(context.Background(), "kawa", rdsIface.FlexsaveRDSReasonCantEnableNoBillingTable)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_dal_Exists(t *testing.T) {
	type fields struct {
		firestoreClient  firestore.Client
		documentsHandler mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr error
		on      func(f *fields)
	}{
		{
			on: func(f *fields) {
				f.documentsHandler.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("Exists", mock.Anything).Return(true).Once()
						return snap
					}(), nil).Once()
			},
			name:    "doc exists",
			want:    true,
			wantErr: nil,
		},

		{
			on: func(f *fields) {
				f.documentsHandler.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("Exists", mock.Anything).Return(false).Once()
						return snap
					}(), nil).Once()
			},
			name:    "doc does not exists",
			want:    false,
			wantErr: nil,
		},

		{
			on: func(f *fields) {
				f.documentsHandler.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(nil, errors.New("err")).Once()
			},
			name:    "returns error",
			want:    false,
			wantErr: errors.New("err"),
		},

		{
			on: func(f *fields) {
				f.documentsHandler.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(nil, status.Errorf(codes.NotFound, "The item was not found")).Once()
			},
			name:    "returns error",
			want:    false,
			wantErr: nil,
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)
			d := &dal{
				firestoreClient:  &tt.fields.firestoreClient,
				documentsHandler: &tt.fields.documentsHandler,
			}

			result, err := d.Exists(context.Background(), "zimbabwe")
			assert.Equal(t, tt.want, result)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_dal_Create(t *testing.T) {
	type fields struct {
		firestoreClient  firestore.Client
		documentsHandler mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr error
		on      func(f *fields)
	}{
		{
			on: func(f *fields) {
				f.documentsHandler.On("Create", testutils.ContextBackgroundMock, mock.Anything, rdsIface.FlexsaveRDSCache{
					ReasonCantEnable: []rdsIface.FlexsaveRDSReasonCantEnable{},
					TimeEnabled:      nil,
					SavingsSummary: rdsIface.FlexsaveSavingsSummary{
						CurrentMonth:     "",
						NextMonthSavings: 0,
					},
					SavingsHistory:      map[string]rdsIface.MonthSummary{},
					DailySavingsHistory: map[string]rdsIface.MonthSummary{},
				}).Return(nil, nil).Once()
			},
			name:    "creates successfully",
			wantErr: nil,
		},
		{
			on: func(f *fields) {
				f.documentsHandler.On("Create", testutils.ContextBackgroundMock, mock.Anything, rdsIface.FlexsaveRDSCache{
					ReasonCantEnable: []rdsIface.FlexsaveRDSReasonCantEnable{},
					TimeEnabled:      nil,
					SavingsSummary: rdsIface.FlexsaveSavingsSummary{
						CurrentMonth:     "",
						NextMonthSavings: 0,
					},
					SavingsHistory:      map[string]rdsIface.MonthSummary{},
					DailySavingsHistory: map[string]rdsIface.MonthSummary{},
				}).Return(nil, errors.New("oh no")).Once()
			},
			name:    "unable to create",
			wantErr: errors.New("oh no"),
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)
			d := &dal{
				firestoreClient:  &tt.fields.firestoreClient,
				documentsHandler: &tt.fields.documentsHandler,
			}

			err := d.Create(context.Background(), "zimbabwe")
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_dal_Enable(t *testing.T) {
	type fields struct {
		firestoreClient  firestore.Client
		documentsHandler mocks.DocumentsHandler
	}

	timeEnabled := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)
	ctx := context.Background()

	tests := []struct {
		name    string
		fields  fields
		wantErr error
		on      func(f *fields)
	}{
		{
			on: func(f *fields) {
				f.documentsHandler.On("Set", ctx, mock.Anything, map[string]interface{}{
					"timeEnabled": timeEnabled,
				}, firestore.MergeAll).Return(&firestore.WriteResult{}, nil).Once()
			},
			name:    "updates successfully",
			wantErr: nil,
		},
		{
			on: func(f *fields) {
				f.documentsHandler.On("Set", ctx, mock.Anything, map[string]interface{}{
					"timeEnabled": timeEnabled,
				}, firestore.MergeAll).Return(nil, errors.New("update failed")).Once()
			},
			name:    "unable to update",
			wantErr: errors.New("update failed"),
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)
			d := &dal{
				firestoreClient:  &tt.fields.firestoreClient,
				documentsHandler: &tt.fields.documentsHandler,
			}

			err := d.Enable(ctx, "mr_customer", timeEnabled)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}
