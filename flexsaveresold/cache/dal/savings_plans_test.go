package dal

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSavingsPlansDAL_CreateCustomerSavingsPlansCache(t *testing.T) {
	ref := mock.AnythingOfType("*firestore.DocumentRef")
	contextMock := mock.MatchedBy(func(_ context.Context) bool { return true })

	errExample := errors.New("things have gone terribly wrong")

	type fields struct {
		fsClient   firestore.Client
		docHandler mocks.DocumentsHandler
	}

	savingsPlans := []types.SavingsPlanData{
		{
			SavingsPlanID:    "5631fb63-2450-4656-91ac-9c77efceb341",
			UpfrontPayment:   0,
			RecurringPayment: 15,
			Commitment:       20,
			ExpirationDate:   time.Date(2022, 7, 5, 0, 0, 0, 0, time.UTC),
		},
	}

	var savingsPlanDoc = types.SavingsPlanDoc{
		SavingsPlans: savingsPlans,
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.docHandler.On("Set", contextMock, ref, savingsPlanDoc).Return(nil, nil)
			},
			wantErr: nil,
		},
		{
			name: "failed to set",
			on: func(f *fields) {
				f.docHandler.On("Set", contextMock, ref, savingsPlanDoc).Return(nil, errExample)
			},
			wantErr: errExample,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			d := &SavingsPlansDAL{
				firestoreClient:  &fields.fsClient,
				documentsHandler: &fields.docHandler,
			}

			err := d.CreateCustomerSavingsPlansCache(context.Background(), "mr_customer", savingsPlans)

			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
