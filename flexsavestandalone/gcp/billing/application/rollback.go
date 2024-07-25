package application

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/retry"
)

func (o *Onboarding) rollback(ctx context.Context, requestParams *dataStructures.OnboardingRequestBody, step onboardingStep, originalError error) {
	logger := o.loggerProvider(ctx)

	// Early return if we completed successfully.
	if step == onboardingStepCompleted {
		return
	}

	logger.Infof("rolling back onboarding for BA %s at step %q. Onboarding failed with error %v", requestParams.BillingAccountID, step, originalError)

	switch step {
	case onboardingStepCreateMetadataForNewBillingID, onboardingStepFindOrCreateBucket:
		o.rollbackDeleteMetadataForNewBillingID(ctx, requestParams, step)
	case onboardingStepCreateLocalTable, onboardingStepNotifyStarted:
		o.rollbackDeleteLocalTable(ctx, requestParams, step)
		o.rollbackDeleteMetadataForNewBillingID(ctx, requestParams, step)
	default:
		logger.Infof("rollback for BA %s at step %q performed no action", requestParams.BillingAccountID, step)
	}
}

func (o *Onboarding) rollbackDeleteLocalTable(ctx context.Context, requestParams *dataStructures.OnboardingRequestBody, step onboardingStep) {
	logger := o.loggerProvider(ctx)

	err := o.table.DeleteLocalTable(ctx, requestParams.BillingAccountID)
	if err != nil {
		logger.Errorf("failed to roll back %s for id %v: %v", step, requestParams.BillingAccountID, err)
	}
}

func (o *Onboarding) rollbackDeleteMetadataForNewBillingID(ctx context.Context, requestParams *dataStructures.OnboardingRequestBody, step onboardingStep) {
	logger := o.loggerProvider(ctx)

	err := retry.BackOffDelay(
		func() error {
			err := o.metadata.DeleteInternalTaskMetadata(ctx, requestParams.BillingAccountID)
			if err != nil {
				return err
			}

			return nil
		},
		consts.MetadataOperationMaxRetries,
		consts.MetadataOperationFirstRetryDelay,
	)
	if err != nil {
		logger.Errorf("failed to roll back %s for id %v: %v", step, requestParams.BillingAccountID, err)
	}
}
