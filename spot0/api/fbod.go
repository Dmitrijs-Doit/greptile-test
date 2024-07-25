package api

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/spot0/api/model"
)

func (s *SpotZeroService) ExecuteFallbackOnDemand(ctx context.Context, req *model.FallbackOnDemandRequest) error {
	logger := s.Logger(ctx)

	err := s.invokeLambda(ctx, req)
	if err != nil {
		logger.Errorf("could not invoke fallback to on demand lambda. error %s customer %s account %s",
			err, req.ExternalID, req.AccountID)
		return err
	}

	return nil
}

func (s *SpotZeroService) getAccountDetails(ctx context.Context, req *model.FallbackOnDemandRequest) error {
	logger := s.Logger(ctx)

	// get user aws-connect creds from firestore
	docSnaps, err := s.Firestore(ctx).CollectionGroup("cloudConnect").
		Where("cloudPlatform", "==", common.Assets.AmazonWebServices).
		Where("accountId", "==", req.AccountID).
		Documents(ctx).GetAll()
	if err != nil {
		logger.Errorf("could not get account from firestore. error %s account id %s", err, req.AccountID)
		return ErrInternalServerError
	}

	if docSnaps == nil || docSnaps[0] == nil {
		return ErrNotFound
	}

	var cred cloudconnect.AmazonWebServicesCredential
	if err := docSnaps[0].DataTo(&cred); err != nil {
		logger.Errorf("could not unmarshal account credentials. error %s", err)
		return ErrInternalServerError
	}

	req.RoleToAssumeArn = cred.Arn
	req.ExternalID = cred.Customer.ID

	return nil
}
