package api

import (
	"context"

	"github.com/google/uuid"

	"github.com/doitintl/hello/scheduled-tasks/spot0/api/model"
)

func (s *SpotZeroService) ExecuteSpotScaling(ctx context.Context, req *model.ApplyConfigurationRequest) (*model.Response, error) {
	logger := s.Logger(ctx)

	execID, err := uuid.NewRandom()
	if err != nil {
		logger.Errorf("could not create new uuid. error %s", err)
		return nil, ErrInternalServerError
	}

	// verify customer
	err = s.verifyCustomer(ctx, req.CustomerID)
	if err != nil {
		logger.Errorf("could not verify customer permissions. %s", err)
		return nil, ErrForbidden
	}

	getAccountsEvent := GetAccountsEvent{
		Scope:            req.Scope,
		CustomerID:       req.CustomerID,
		Region:           req.Region,
		AsgName:          req.ASGName,
		AccountID:        req.AccountID,
		ExecID:           execID.String(),
		ForceManagedMode: req.ForceManagedMode,
		Configuration:    req.Configuration,
	}

	_, err = s.startAWSStepFunction(ctx, &getAccountsEvent)
	if err != nil {
		logger.Errorf("could not start spot zero step function. %s", err)
		return nil, err
	}

	resp, err := s.getExecutionStatus(ctx, &getAccountsEvent)
	if err != nil {
		logger.Errorf("could not get execution status. error %s", err)
		return nil, err
	}

	logger.Infof("successfully retrieved execution status - done %t", resp.Done)

	return resp, nil
}
