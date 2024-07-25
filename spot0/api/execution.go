package api

import (
	"context"
	"errors"
	"regexp"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/spot0/api/model"
)

var (
	retryCount                           = 20
	customerScope                        = "customer"
	customerScopeTimeDelta time.Duration = 3
)

type RegionExecutionDetails struct {
	Started int `firestore:"started"`
	Ended   int `firestore:"ended"`
}

func (s *SpotZeroService) getExecutionStatus(ctx context.Context, evt *GetAccountsEvent) (*model.Response, error) {
	logger := s.Logger(ctx)

	if evt == nil || len(evt.ExecID) == 0 {
		err := errors.New("could not respond with execution status. execution id is empty")
		logger.Error(err)

		return nil, err
	}

	var customDelta *time.Duration

	isCustomerScope := evt.Scope == customerScope
	if isCustomerScope {
		customDelta = &customerScopeTimeDelta
	}

	docRef := s.Firestore(ctx).Collection("spot0").Doc("spotApp").Collection("executions").Doc(evt.ExecID)

	var (
		docSnap     *firestore.DocumentSnapshot
		done        bool
		shouldRetry = errors.New("should retry")
	)

	err := retry(func(attempt int) (bool, error) {
		var er error

		docSnap, er = docRef.Get(ctx)
		if er != nil {
			return attempt < retryCount, er
		}

		var exec map[string]RegionExecutionDetails

		er = docSnap.DataTo(&exec)
		if er != nil {
			return attempt < retryCount, er
		}

		if docSnap == nil {
			return attempt < retryCount, shouldRetry
		}

		done = isExecutionDone(exec)
		if done {
			_, er = docRef.Delete(ctx)
			if er != nil {
				logger.Errorf("could nod delete execution %s status, error: %s", evt.ExecID, er)
			}

			return false, nil
		}

		return attempt < retryCount, shouldRetry
	}, customDelta)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}

		if err == shouldRetry {
			return &model.Response{
				Done: done,
			}, nil
		}

		logger.Error(err)

		return nil, err
	}

	if !isCustomerScope {
		err = s.getExecutionErrors(ctx, evt)
		if err != nil {
			return &model.Response{
				Done:         false,
				ErrorMessage: err.Error(),
			}, nil
		}
	}

	return &model.Response{
		Done: done,
	}, nil
}

func (s *SpotZeroService) getExecutionErrors(ctx context.Context, evt *GetAccountsEvent) error {
	idBase := evt.AccountID + "_" + evt.Region + "_" + evt.AsgName
	id := regexp.MustCompile(`[\.\/\?\*\(\)\&=]`).ReplaceAll([]byte(idBase), []byte("-"))

	doc, err := s.Firestore(ctx).Collection("spot0").Doc("spotApp").Collection("asgs").Doc(string(id)).Get(ctx)
	if err != nil {
		return &model.ApplyConfigurationError{Code: "500", Err: err}
	}

	var asg model.AsgState

	err = doc.DataTo(&asg)
	if err != nil {
		return &model.ApplyConfigurationError{Code: "500", Err: err}
	}

	if asg.SpotisizeNotSupported {
		return &model.ApplyConfigurationError{Code: "400", Err: errors.New("spotisize not supported")}
	}

	if asg.ManagedStatus != "success" {
		if !evt.ForceManagedMode && asg.Mode != "managed" {
			return nil
		}

		if asg.SpotisizeErrorDesc != "" {
			return &model.ApplyConfigurationError{Code: "500", Err: errors.New(asg.SpotisizeErrorDesc)}
		}

		if asg.Error != "" {
			return &model.ApplyConfigurationError{Code: "500", Err: errors.New(asg.Error)}
		}

		return &model.ApplyConfigurationError{Code: "500", Err: errors.New("could not update asg")}
	}

	return nil
}

func isExecutionDone(exec map[string]RegionExecutionDetails) bool {
	// loop on all account_region
	for _, dtl := range exec {
		if dtl.Started != dtl.Ended {
			return false
		}
	}

	return true
}
