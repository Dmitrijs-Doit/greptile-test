package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"

	dalMocks "github.com/doitintl/hello/scheduled-tasks/marketplace/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

const (
	CMPCustomerID    = "AAACMPCustomerID"
	accountID        = "aaaaaaaa-b7b8-b9ba-bbbc-bdbebfc0c1c2"
	billingAccountID = "AAAAAA-ABCDEF-123456"
	approveReason    = "Approved by user@doit.com via CMP"
	rejectReason     = "Rejected by user@doit.com via CMP"
)

func TestAccountService_GetAccount(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		accountDAL *dalMocks.IAccountFirestoreDAL
	}

	type args struct {
		ctx       context.Context
		accountID string
		email     string
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
		err     error
		on      func(*fields)
	}{
		{
			name: "Successfully get an account",
			args: args{
				ctx:       ctx,
				accountID: accountID,
				email:     email,
			},
			wantErr: false,
			err:     nil,
			on: func(f *fields) {
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, accountID).
					Return(&domain.AccountFirestore{}, nil).
					Once()
			},
		},
		{
			name: "error on getting an account",
			args: args{
				ctx:       ctx,
				accountID: accountID,
				email:     email,
			},
			wantErr: true,
			err:     ErrGetAccountFirestore,
			on: func(f *fields) {
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, accountID).
					Return(nil, errors.New("some firestore error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				accountDAL: &dalMocks.IAccountFirestoreDAL{},
			}

			s := &MarketplaceService{
				accountDAL: tt.fields.accountDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if _, err := s.GetAccount(tt.args.ctx, tt.args.accountID); (err != nil) != tt.wantErr {
				t.Errorf("AccountService.GetAccount() error = %v, tt.wantErr %v", err, tt.wantErr)
			} else if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("AccountService.GetAccount() error = %v, tt.err %v", err, tt.err)
			}
		})
	}
}

func TestAccountService_ApproveAccount(t *testing.T) {
	ctx := context.Background()

	validAccountFirestore := &domain.AccountFirestore{
		Customer:         &firestore.DocumentRef{ID: CMPCustomerID},
		BillingAccountID: billingAccountID,
		User:             &domain.UserDetails{},
	}

	invalidAccountFirestoreMissingCustomer := &domain.AccountFirestore{
		BillingAccountID: billingAccountID,
	}

	invalidAccountFirestoreMissingBillingAccount := &domain.AccountFirestore{
		Customer: &firestore.DocumentRef{
			ID: CMPCustomerID,
		},
		User: &domain.UserDetails{},
	}

	invalidAccountFirestoreMissingUser := &domain.AccountFirestore{
		Customer: &firestore.DocumentRef{
			ID: CMPCustomerID,
		},
	}

	type fields struct {
		procurementDAL *dalMocks.ProcurementDAL
		accountDAL     *dalMocks.IAccountFirestoreDAL
	}

	type args struct {
		ctx       context.Context
		accountID string
		email     string
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
		err     error
		on      func(*fields)
	}{
		{
			name: "Successfully approve an account",
			args: args{
				ctx:       ctx,
				accountID: accountID,
				email:     email,
			},
			wantErr: false,
			err:     nil,
			on: func(f *fields) {
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, accountID).
					Return(validAccountFirestore, nil).
					Once()
				f.procurementDAL.
					On("ApproveAccount", testutils.ContextBackgroundMock, accountID, approveReason).
					Return(nil).
					Once()
			},
		},
		{
			name: "Fail to get account from firestore",
			args: args{
				ctx:       ctx,
				accountID: accountID,
				email:     email,
			},
			wantErr: true,
			err:     ErrGetAccountFirestore,
			on: func(f *fields) {
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, accountID).
					Return(nil, errors.New("account dal error")).
					Once()
			},
		},
		{
			name: "Fail to get account with customer from firestore",
			args: args{
				ctx:       ctx,
				accountID: accountID,
				email:     email,
			},
			wantErr: true,
			err:     domain.ErrAccountCustomerMissing,
			on: func(f *fields) {
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, accountID).
					Return(invalidAccountFirestoreMissingCustomer, nil).
					Once()
			},
		},
		{
			name: "Fail to get account with billing account from firestore",
			args: args{
				ctx:       ctx,
				accountID: accountID,
				email:     email,
			},
			wantErr: true,
			err:     domain.ErrAccountBillingAccountMissing,
			on: func(f *fields) {
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, accountID).
					Return(invalidAccountFirestoreMissingBillingAccount, nil).
					Once()
			},
		},
		{
			name: "Fail to get account with user from firestore",
			args: args{
				ctx:       ctx,
				accountID: accountID,
				email:     email,
			},
			wantErr: true,
			err:     domain.ErrAccountUserMissing,
			on: func(f *fields) {
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, accountID).
					Return(invalidAccountFirestoreMissingUser, nil).
					Once()
			},
		},
		{
			name: "Fail to approve an account",
			args: args{
				ctx:       ctx,
				accountID: accountID,
				email:     email,
			},
			wantErr: true,
			err:     errors.New("approve account error"),
			on: func(f *fields) {
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, accountID).
					Return(validAccountFirestore, nil).
					Once()
				f.procurementDAL.
					On("ApproveAccount", testutils.ContextBackgroundMock, accountID, approveReason).
					Return(errors.New("approve account error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				procurementDAL: &dalMocks.ProcurementDAL{},
				accountDAL:     &dalMocks.IAccountFirestoreDAL{},
			}

			s := &MarketplaceService{
				procurementDAL: tt.fields.procurementDAL,
				accountDAL:     tt.fields.accountDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.ApproveAccount(tt.args.ctx, tt.args.accountID, tt.args.email); (err != nil) != tt.wantErr {
				t.Errorf("MarketplaceService.ApproveAccount() error = %v, tt.wantErr %v", err, tt.wantErr)
			} else if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("MarketplaceService.ApproveAccount() error = %v, tt.err %v", err, tt.err)
			}
		})
	}
}

func TestAccountService_RejectAccount(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		procurementDAL *dalMocks.ProcurementDAL
	}

	type args struct {
		ctx       context.Context
		accountID string
		email     string
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "Successfully reject an account",
			args: args{
				ctx:       ctx,
				accountID: accountID,
				email:     email,
			},
			on: func(f *fields) {
				f.procurementDAL.
					On("RejectAccount", testutils.ContextBackgroundMock, accountID, rejectReason).
					Return(nil).
					Once()
			},
		},
		{
			name: "Fail to reject an account",
			args: args{
				ctx:       ctx,
				accountID: accountID,
				email:     email,
			},
			wantErr: true,
			on: func(f *fields) {
				f.procurementDAL.
					On("RejectAccount", testutils.ContextBackgroundMock, accountID, rejectReason).
					Return(errors.New("error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				procurementDAL: &dalMocks.ProcurementDAL{},
			}

			s := &MarketplaceService{
				procurementDAL: tt.fields.procurementDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.RejectAccount(tt.args.ctx, tt.args.accountID, tt.args.email); (err != nil) != tt.wantErr {
				t.Errorf("MarketplaceService.RejectAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
