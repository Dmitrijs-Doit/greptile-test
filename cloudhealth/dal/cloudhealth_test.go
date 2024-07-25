package dal

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCloudHealthDAL_GetCustomerCloudHealthID(t *testing.T) {
	customerRef := &firestore.DocumentRef{ID: "mr_cmp_customer"}
	iter := mock.AnythingOfType("*firestore.DocumentIterator")

	errExample := errors.New("things have gone terribly wrong")

	type fields struct {
		fsClient   firestore.Client
		docHandler mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
		want    string
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.docHandler.On("GetAll", iter).
					Return(func() []iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("ID").Return("mr_cloudhealth_customer")

						return []iface.DocumentSnapshot{
							snap,
						}
					}(), nil).
					Once()
			},
			want: "mr_cloudhealth_customer",
		},
		{
			name: "failed to get snapshot from iterator",
			on: func(f *fields) {
				f.docHandler.On("GetAll", iter).
					Return(nil, errExample).Once()
			},
			wantErr: errExample,
			want:    "",
		},
		{
			name: "no CloudHealth customer doc found for customer",
			on: func(f *fields) {
				f.docHandler.On("GetAll", iter).
					Return(func() []iface.DocumentSnapshot {
						return []iface.DocumentSnapshot{}
					}(), nil).
					Once()
			},
			wantErr: fmt.Errorf("no CloudHealth customer doc found for customer: %v", "mr_cmp_customer"),
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &CloudHealthDAL{
				firestoreClient:  &fields.fsClient,
				documentsHandler: &fields.docHandler,
			}

			got, err := s.GetCustomerCloudHealthID(context.Background(), customerRef)

			assert.Equal(t, got, tt.want)

			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
