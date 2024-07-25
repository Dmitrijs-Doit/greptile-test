package service

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
)

// ApproveAccount approves an account in the procurement system if the account is linked
// to a customer and billing account in CMP.
func (s *MarketplaceService) ApproveAccount(ctx context.Context, accountID, email string) error {
	account, err := s.GetAccount(ctx, accountID)
	if err != nil {
		return err
	}

	if err := account.Validate(); err != nil {
		return err
	}

	reason := fmt.Sprintf("Approved by %s via CMP", email)

	if err := s.procurementDAL.ApproveAccount(ctx, accountID, reason); err != nil {
		return err
	}

	return nil
}

// RejectAccount rejects an account in the procurement system.
func (s *MarketplaceService) RejectAccount(ctx context.Context, accountID, email string) error {
	reason := fmt.Sprintf("Rejected by %s via CMP", email)

	if err := s.procurementDAL.RejectAccount(ctx, accountID, reason); err != nil {
		return err
	}

	return nil
}

// GetAccount returns account.
func (s *MarketplaceService) GetAccount(ctx context.Context, accountID string) (*domain.AccountFirestore, error) {
	account, err := s.accountDAL.GetAccount(ctx, accountID)
	if err != nil {
		return nil, ErrGetAccountFirestore
	}

	return account, nil
}
