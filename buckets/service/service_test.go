package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/buckets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	entityDalMock "github.com/doitintl/hello/scheduled-tasks/entity/dal/mocks"
	"github.com/stretchr/testify/assert"
)

func TestBucketsService_GetCustomerBuckets(t *testing.T) {
	type fields struct {
		bucketsDal *mocks.Buckets
		entityDal  *entityDalMock.Entites
	}

	type args struct {
		ctx         context.Context
		customerRef *firestore.DocumentRef
	}

	ctx := context.Background()
	customerRef := firestore.DocumentRef{
		ID: "customer-id",
	}

	getCustomerEntitiesResult := []*common.Entity{}

	for i := 0; i < 200; i++ {
		mode := "GROUP"
		if i%2 == 0 {
			mode = "CUSTOM"
		}

		getCustomerEntitiesResult = append(getCustomerEntitiesResult, &common.Entity{
			Snapshot: &firestore.DocumentSnapshot{
				Ref: &firestore.DocumentRef{
					ID: fmt.Sprintf("entity-%d", i),
				},
			},
			Invoicing: common.Invoicing{
				Mode: mode,
			},
		})
	}

	allBuckets := []common.Bucket{}

	for i := 0; i < 400; i++ {
		allBuckets = append(allBuckets, common.Bucket{Name: fmt.Sprintf("Bucket %d", i)})
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error

		on func(*fields)
	}{
		{
			name: "Successfully get customer buckets",
			args: args{
				ctx:         ctx,
				customerRef: &customerRef,
			},
			wantErr: false,
			on: func(f *fields) {
				f.entityDal.On("GetCustomerEntities", ctx, &customerRef).Return(getCustomerEntitiesResult, nil)
				for i := 0; i < 200; i++ {
					if i%2 == 0 {
						f.bucketsDal.On("GetBuckets", ctx, fmt.Sprintf("entity-%d", i)).Return(allBuckets[i*2:i*2+2], nil)
					}
				}
			},
			expectedErr: nil,
		},
		{
			name: "Error getting customer entities",
			args: args{
				ctx:         ctx,
				customerRef: &customerRef,
			},
			wantErr: true,
			on: func(f *fields) {
				f.entityDal.On("GetCustomerEntities", ctx, &customerRef).Return(nil, errors.New("error getting customer entitites"))
			},
			expectedErr: errors.New("error getting customer entitites"),
		},
		{
			name: "Error getting customer buckets",
			args: args{
				ctx:         ctx,
				customerRef: &customerRef,
			},
			wantErr: true,
			on: func(f *fields) {
				f.entityDal.On("GetCustomerEntities", ctx, &customerRef).Return(getCustomerEntitiesResult, nil)
				for i := 0; i < 200; i++ {
					if i == 102 {
						f.bucketsDal.On("GetBuckets", ctx, "entity-102").Return(nil, errors.New("error getting customer buckets"))
						continue
					}

					if i%2 == 0 {
						f.bucketsDal.On("GetBuckets", ctx, fmt.Sprintf("entity-%d", i)).Return(allBuckets[i*2:i*2+2], nil)
					}
				}
			},
			expectedErr: errors.New("error getting customer buckets"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				bucketsDal: &mocks.Buckets{},
				entityDal:  &entityDalMock.Entites{},
			}
			s := &BucketsService{
				bucketsDal: tt.fields.bucketsDal,
				entityDal:  tt.fields.entityDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.GetCustomerBuckets(tt.args.ctx, tt.args.customerRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("BucketsService.GetCustomerBuckets() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else if !tt.wantErr {
				for i, bucket := range allBuckets {
					if (i/2)%2 == 0 {
						assert.Contains(t, got, bucket)
					}
				}
			}
		})
	}
}
