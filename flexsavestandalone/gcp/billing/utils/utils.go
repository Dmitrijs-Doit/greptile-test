package utils

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	automationConsts "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/consts"

	"github.com/doitintl/hello/scheduled-tasks/logger"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
)

func GetIsDebugValue() bool {
	debugStr := os.Getenv("DEBUG")
	if debugStr != "" {
		isDebug, err := strconv.ParseBool(debugStr)
		if err == nil {
			return isDebug
		}

		return false
	}

	return false
}

func SetupContext(ctx context.Context, logger logger.ILogger, ctxName string) (newCtx context.Context, ctxCancel context.CancelFunc, err error) {
	newCtx = context.WithValue(ctx, consts.IsDebugKey, GetIsDebugValue())
	ctxExpirationTime := time.Now().Add(consts.InternalManagerMaxDuration)
	newCtx = context.WithValue(newCtx, consts.ContextExpirationTimeKey, ctxExpirationTime)
	newCtx, ctxCancel = context.WithCancel(newCtx)

	go func(ctx context.Context, ctxCancel context.CancelFunc, name string) {
		select {
		case <-time.After(consts.InternalManagerMaxDuration):
			logger.Errorf("timeout reached for context %s. canceling the context.", name)
			ctxCancel()
		case <-ctx.Done():
			logger.Infof("context %s done. Closing canceling func", name)
			return
		}
	}(newCtx, ctxCancel, ctxName)

	return newCtx, ctxCancel, nil
}

func GetProjectName() string {
	if common.Production {
		return consts.BillingProjectProd
	}

	if common.ProjectID == consts.BillingProjectMeDoitIntlCom {
		return consts.BillingProjectProd
	}

	return common.ProjectID
}

func GetUnifiedTempTableName(iteration int64) string {
	return fmt.Sprintf("%s_%d", consts.UnifiedRawBillingTablePrefix, iteration)
}

func GetUnifiedTempTableFullName(iteration int64) string {
	return fmt.Sprintf("%s.%s.%s", GetProjectName(), consts.UnifiedGCPBillingDataset, GetUnifiedTempTableName(iteration))
}

func GetLocalCopyAccountTableName(billingAccountID string) string {
	return fmt.Sprintf("%s_%s", consts.LocalBillingTablePrefix, strings.Replace(billingAccountID, "-", "_", -1))
}
func GetAlternativeLocalCopyAccountTableName(billingAccountID string) string {
	return fmt.Sprintf("%s_%s", consts.AlternativeLocalBillingTablePrefix, strings.Replace(billingAccountID, "-", "_", -1))
}

func GetLocalTableByBillingAccount(billingAccountID string) *dataStructures.BillingTableInfo {
	return &dataStructures.BillingTableInfo{
		ProjectID: GetProjectName(),
		DatasetID: consts.LocalBillingDataset,
		TableID:   GetLocalCopyAccountTableName(billingAccountID),
	}
}

func IsDummy(remoteDataset string) bool {
	return automationConsts.DummyBQDatasetName == remoteDataset
}

func GetLocalCopyAccountTableFullName(billingAccountID string) string {
	return fmt.Sprintf("%s.%s.%s", GetProjectName(), consts.LocalBillingDataset, GetLocalCopyAccountTableName(billingAccountID))
}

func GetFullTableName(t *dataStructures.BillingTableInfo) (string, error) {
	if t == nil || t.TableID == "" || t.DatasetID == "" || t.ProjectID == "" {
		return "", fmt.Errorf("unable to set table name")
	}

	return fmt.Sprintf("%s.%s.%s", t.ProjectID, t.DatasetID, t.TableID), nil
}

func GetCopyToUnifiedTableJobPrefix(iteration int64) string {
	return fmt.Sprintf(consts.CopyToUnifiedTableJobPrefixTemplate, iteration)
}

func GetFromLocalTableToTmpTableJobPrefix(billingAccount string, iteration int64) string {
	return fmt.Sprintf(consts.FromLocalTableToTmpTableJobPrefixTemplate, billingAccount, iteration)
}

func GetMarkVerifiedTmpTableJobPrefix(billingAccount string, iteration int64) string {
	return fmt.Sprintf(consts.MarkVerifiedTmpTableJobPrefixTemplate, billingAccount, iteration)
}

func GetDeleteRowsOfBA(billingAccount string) string {
	return fmt.Sprintf(consts.DeleteRowsOfBATemplate, billingAccount)
}

func GetWaitForExternalJobToFinishMaxDuration(onboarding bool) time.Duration {
	if onboarding {
		return consts.WaitForExternalJobToFinishOnBoarding
	}

	return consts.WaitForExternalJobToFinish
}

func GetToBucketJobPrefix(billingAccount string, iteration int64) string {
	return fmt.Sprintf(consts.ToBucketJobPrefixTemplate, billingAccount, iteration)
}

func GetToBucketJobPrefixCheck(billingAccount string, iteration int64) string {
	return fmt.Sprintf(consts.ToBucketJobPrefixTemplate+"-", billingAccount, iteration)
}

func GetFromBucketJobPrefix(billingAccount string, iteration int64) string {
	return fmt.Sprintf(consts.FromBucketJobPrefixTemplate, billingAccount, iteration)
}

func GetFromBucketJobPrefixCheck(billingAccount string, iteration int64) string {
	return fmt.Sprintf(consts.FromBucketJobPrefixTemplate+"-", billingAccount, iteration)
}

func CheckExternalTaskJobsForNilPointers(etm *dataStructures.ExternalTaskMetadata) error {
	if etm.ExternalTaskJobs == nil {
		return fmt.Errorf("nil pointer for ExternalTaskJobs")
	}

	err := CheckToBucketJobForNilPointers(etm.ExternalTaskJobs)
	if err != nil {
		return err
	}

	err = CheckFromBucketJobForNilPointers(etm.ExternalTaskJobs)
	if err != nil {
		return err
	}

	return nil
}

func CheckToBucketJobForNilPointers(etm *dataStructures.ExternalTaskJobs) error {
	err := CheckJobForNilPointers(etm.ToBucketJob)
	if err != nil {
		return err
	}

	return nil
}

func CheckFromBucketJobForNilPointers(etm *dataStructures.ExternalTaskJobs) error {
	err := CheckJobForNilPointers(etm.FromBucketJob)
	if err != nil {
		return err
	}

	return nil
}

func CheckJobForNilPointers(job *dataStructures.Job) error {
	if job == nil {
		return fmt.Errorf("nil pointer for job")
	}

	err := CheckTimeForNilPointers(job.WaitToFinishTimeout)
	if err != nil {
		return err
	}

	err = CheckTimeForNilPointers(job.WaitToStartTimeout)
	if err != nil {
		return err
	}

	return nil
}

func CheckTimeForNilPointers(time *time.Time) error {
	if time == nil {
		return fmt.Errorf("nil pointer for time")
	}

	return nil
}
