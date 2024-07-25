package mpa

import (
	"context"
)

func (s *MasterPayerAccountService) RetireMPA(ctx context.Context, payerID string) error {
	l := s.loggerProvider(ctx)

	// Check mpa account exists
	_, err := s.mpaDAL.GetMasterPayerAccount(ctx, payerID)
	if err != nil {
		return err
	}

	l.Infof("Retire MPA == Retiring MPA started for payerID: %v", payerID)
	if err := s.mpaDAL.RetireMPAandDeleteAssets(ctx, l, payerID); err != nil {
		return err
	}

	return nil
}
