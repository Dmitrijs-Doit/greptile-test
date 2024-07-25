package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/costandusagereportservice"
	"github.com/doitintl/firestore/pkg"
	assets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/saasservice/utils"
	"github.com/doitintl/retry"
)

/**
Standalone onboarding documents are being store in a "StandaloneID" convention
`amazon-web-services-CUSTOMER_ID.accounts.ACCOUNT_ID`
which construct with:
* firestore document ID: `amazon-web-services-CUSTOMER_ID`
* accounts field: `accounts`
* aws account ID: `ACCOUNT_ID

Single document on firestore (/integrations/billing-standalone/standaloneOnboarding)
may include single or many aws accounts

** In order to generate "Standalone ID" use server/shared/firestore/pkg.ComposeStandaloneID(). Call it easily by s.composeStandaloneID(customerID, accountID)
** In order to take crucial parts out of standaloneID use server/shared/firestore/pkg.ExtractStandaloneID()
*/

// InitOnboarding creates onboarding document for the 1st time
func (s *AWSSaaSConsoleOnboardService) InitOnboarding(ctx context.Context, customerID string, req *AWSSaaSConsoleInitRequest) *saasconsole.OnboardingResponse {
	logger := s.getLogger(ctx, customerID)
	step := pkg.OnboardingStepInit

	documentID := s.getDocumentID(customerID)
	standaloneID := s.composeStandaloneID(customerID, req.AccountID)

	if err := saasconsole.InitOnboarding(ctx, customerID, documentID, s.customersDAL, s.saasConsoleDAL); err != nil {
		if err == pkg.ErrorAlreadyExist {
			if _, err := s.saasConsoleDAL.GetAWSOnboarding(ctx, standaloneID); err != nil {
				if err != pkg.ErrorNoAccount {
					return s.updateAndSlackError(ctx, customerID, req.AccountID, step, err)
				}
			} else {
				return saasconsole.Success
			}
		} else {
			return s.updateAndSlackError(ctx, customerID, req.AccountID, step, err)
		}
	}

	document := &pkg.AWSStandaloneOnboarding{
		AccountID: req.AccountID,
	}

	if err := s.saasConsoleDAL.UpdateOnboarding(ctx, standaloneID, document); err != nil {
		return s.updateAndSlackError(ctx, customerID, req.AccountID, step, err)
	}

	logger.Info(saasconsole.StepMessageInitOnboardingCompleted)

	return saasconsole.Success
}

// AddContract creates contract for a given customer
func (s *AWSSaaSConsoleOnboardService) AddContract(ctx context.Context, req *saasconsole.StandaloneContractRequest) *saasconsole.OnboardingResponse {
	step := pkg.OnboardingStepContract

	if err := saasconsole.AddContract(
		ctx,
		req,
		saasconsole.StandaloneContractTypeAWS,
		0,
		s.contractsDAL.Add,
		s.entitiesDAL,
		s.customersDAL,
		s.accountManagersDAL,
		s.saasConsoleDAL,
	); err != nil {
		return s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	return saasconsole.Success
}

// CURDiscovery wraps and invokes curDiscovery with timeout, after the server safely returns 200
func (s *AWSSaaSConsoleOnboardService) CURDiscovery(ctx context.Context, req *AWSSaaSConsoleCURDiscoveryRequest) *saasconsole.OnboardingResponse {
	step := pkg.OnboardingStepCURDiscovery
	logger := s.getLogger(ctx, req.CustomerID)

	logger.Infof("%s - customer %s account %s", stepMessageCURDiscoveryStarted, req.CustomerID, req.AccountID)

	logger.Debugf("request: %+v", req)

	if err := s.curDiscovery(ctx, req.CustomerID, req.AccountID, req.S3Bucket); err != nil {
		return s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	standaloneID := s.composeStandaloneID(req.CustomerID, req.AccountID)

	if err := s.saasConsoleDAL.SetAWSArn(ctx, standaloneID, req.Arn); err != nil {
		return s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	logger.Info(stepMessageCURDiscoveryCompleted)

	return saasconsole.Success
}

func (s *AWSSaaSConsoleOnboardService) CURRefresh(ctx context.Context, req *AWSSaaSConsoleCURRefreshRequest) *saasconsole.OnboardingResponse {
	step := pkg.OnboardingStepCURRefresh
	logger := s.getLogger(ctx, req.CustomerID)

	logger.Info(stepMessageCURRefreshStarted)

	standaloneID := s.composeStandaloneID(req.CustomerID, req.AccountID)

	path, err := s.saasConsoleDAL.GetAWSCURPath(ctx, standaloneID, 0)
	if err != nil {
		return s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	if err := s.curDiscovery(ctx, req.CustomerID, req.AccountID, path.Bucket); err != nil {
		return s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	logger.Info(stepMessageCURRefreshCompleted)

	return saasconsole.Success
}

// UpdateBillingWrapper wraps and invokes UpdateBilling with timeout, after the server safely returns 200
func (s *AWSSaaSConsoleOnboardService) UpdateBilling(ctx context.Context, req *AWSSaaSConsoleActivateRequest) *saasconsole.OnboardingResponse {
	step := pkg.OnboardingStepActivation
	logger := s.getLogger(ctx, req.CustomerID)

	logger.Info(stepMessageUpdateBillingStarted)

	if err := s.updateBilling(ctx, req); err != nil {
		return s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	logger.Info(stepMessageUpdateBillingCompleted)

	return saasconsole.Success
}

// StackDeletion add log pointing for stack deletion initiated by AWS console
func (s *AWSSaaSConsoleOnboardService) StackDeletion(ctx context.Context, customerID, accountID string) *saasconsole.OnboardingResponse {
	logger := s.getLogger(ctx, customerID)

	logger.Infof("%s for customer %s, account id: %s", stackDeletion, customerID, accountID)

	customerRef := s.customersDAL.GetRef(ctx, customerID)
	cloudConnectAssetType := utils.GetCloudConnectAssetType(pkg.AWS)

	status, err := s.cloudConnectDAL.GetAWSCloudConnectConnectionStatus(ctx, customerRef, cloudConnectAssetType, accountID)
	if err != nil {
		logger.Errorf("couldn't get cloud connect status for customer %s, account %s, error %s", customerID, accountID, err)

		return saasconsole.Failure
	}

	if status == nil {
		status = &pkg.AWSCloudConnectStatus{
			Status: pkg.AWSCloudConnectStatusValid,
		}
	}

	if status.Status == pkg.AWSCloudConnectStatusValid {
		status.Status = pkg.AWSCloudConnectStatusCritical
		status.InvalidInfo = &pkg.AWSCloudConnectInvalidInfo{
			RoleError:   stackDeletedRoleError,
			PolicyError: stackDeletedPolicyError,
		}

		if err := saasconsole.PublishPermissionsAlertSlackNotification(ctx, pkg.AWS, s.customersDAL, customerID, accountID, status, []string{}); err != nil {
			logger.Errorf("couldn't send slack deletion slack notification for customer %s, account %s, error %s", customerID, accountID, err)
		} else {
			status.InvalidInfo.AlertSentAt = time.Now().UTC()
		}

		if err := s.cloudConnectDAL.SetAWSCloudConnectConnectionStatus(ctx, customerRef, cloudConnectAssetType, accountID, status); err != nil {
			logger.Errorf("couldn't update cloud connect status for customer %s, account %s, error %s", customerID, accountID, err)
		}
	}

	return saasconsole.Failure
}

func (s *AWSSaaSConsoleOnboardService) curDiscovery(ctx context.Context, customerID, accountID, bucket string) error {
	if err := s.validateCustomerAccountID(ctx, customerID, accountID); err != nil {
		return err
	}

	if customerCurInfo, ok := saasconsole.AWSCustomCustomerCurMap[customerID]; ok {
		if curInfo, ok := customerCurInfo[accountID]; ok {
			if curInfo.Bucket != bucket {
				return errorCUR
			}

			curInfo.AccountID = accountID

			return s.handleCustomCurAccount(ctx, customerID, curInfo)
		}
	}

	var reports []*costandusagereportservice.ReportDefinition

	var retryErr error

	_ = retry.BackOffDelay(
		func() error {
			var doNotRetry bool

			reports, doNotRetry, retryErr = s.getReportsDefinitions(accountID)
			if doNotRetry {
				return nil
			}

			return retryErr
		},
		5,
		time.Second*30,
	)

	if retryErr != nil {
		return retryErr
	}

	s.getLogger(ctx, customerID).Debugf("account: %s, found CUR report definitions: %+v", accountID, reports)

	validCURs := []pkg.AWSSaaSConsoleCURPath{}
	invalidCURs := []pkg.AWSSaaSConsoleCURPath{}

	for _, report := range reports {
		valid, path := s.validateSingleCUR(report, bucket)
		if path == nil {
			continue
		}

		if valid {
			validCURs = append(validCURs, *path)
		} else {
			invalidCURs = append(invalidCURs, *path)
		}
	}

	err := retry.BackOffDelay(
		func() error {
			return s.updateCURs(ctx, customerID, accountID, bucket, &validCURs, &invalidCURs)
		},
		5,
		time.Second,
	)

	return err
}

func (s *AWSSaaSConsoleOnboardService) updateCURs(ctx context.Context, customerID, accountID, bucket string, validCURs, invalidCURs *[]pkg.AWSSaaSConsoleCURPath) error {
	standaloneID := s.composeStandaloneID(customerID, accountID)

	if len(*validCURs) > 0 {
		s.setDefaultReport(validCURs)

		if err := s.saasConsoleDAL.SetAWSCURPaths(ctx, standaloneID, &pkg.AWSSaaSConsoleCURPaths{
			State: pkg.SaaSConsoleValidCUR,
			Paths: *validCURs,
		}); err != nil {
			return err
		}

		return nil
	} else if len(*invalidCURs) > 0 {
		s.setDefaultReport(invalidCURs)

		if strings.Contains((*invalidCURs)[0].ReportName, doitSubString) {
			if err := s.saasConsoleDAL.SetAWSCURPaths(ctx, standaloneID, &pkg.AWSSaaSConsoleCURPaths{
				State: pkg.SaaSConsoleInvalidCUR,
				Paths: []pkg.AWSSaaSConsoleCURPath{
					(*invalidCURs)[0],
				},
			}); err != nil {
				return err
			}

			return nil
		}
	}

	if err := s.saasConsoleDAL.SetAWSCURPaths(ctx, standaloneID, &pkg.AWSSaaSConsoleCURPaths{
		State: pkg.SaaSConsoleNoneCURs,
		Paths: []pkg.AWSSaaSConsoleCURPath{
			{
				Bucket: bucket,
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

func (s *AWSSaaSConsoleOnboardService) setDefaultReport(curs *[]pkg.AWSSaaSConsoleCURPath) *[]pkg.AWSSaaSConsoleCURPath {
	for i, cur := range *curs {
		if strings.Contains(cur.ReportName, doitSubString) {
			(*curs)[0], (*curs)[i] = (*curs)[i], (*curs)[0]
			break
		}
	}

	return curs
}

// UpdateBilling updates billing for the customer, final onboarding step
func (s *AWSSaaSConsoleOnboardService) updateBilling(ctx context.Context, req *AWSSaaSConsoleActivateRequest) error {
	account, err := s.saasConsoleDAL.GetAWSAccount(ctx, req.CustomerID, req.AccountID)
	if err != nil {
		return err
	}

	if len(account.CURPaths.Paths) < req.CURPathIndex+1 {
		return errorInvalidCURIndex
	}

	if account.CURPaths.State == pkg.SaaSConsoleInvalidCUR {
		return errorCUR
	}

	if err := s.createAsset(ctx, req.CustomerID, req.AccountID); err != nil {
		return err
	}

	path := account.CURPaths.Paths[req.CURPathIndex]

	if err := s.createCloudConnect(ctx, req.CustomerID, req.AccountID, account.Arn, path.Bucket, path.PathPrefix, path.ReportName); err != nil {
		return err
	}

	if err := s.enableCustomer(ctx, req.CustomerID); err != nil {
		return err
	}

	if err := s.saasConsoleDAL.CompleteAWSOnboarding(ctx, s.composeStandaloneID(req.CustomerID, req.AccountID)); err != nil {
		return err
	}

	standaloneID := s.composeStandaloneID(req.CustomerID, req.AccountID)
	if err := s.saasConsoleDAL.SetAWSCURPaths(ctx, standaloneID, &pkg.AWSSaaSConsoleCURPaths{
		State: pkg.SaaSConsoleValidCUR,
		Paths: []pkg.AWSSaaSConsoleCURPath{path},
	}); err != nil {
		s.getLogger(ctx, req.CustomerID).Errorf("couldn't set selected CUR path for customer %s, account %s, error %s", req.CustomerID, req.AccountID, err)
	}

	_ = saasconsole.PublishOnboardSuccessSlackNotification(ctx, pkg.AWS, s.customersDAL, req.CustomerID, req.AccountID)

	return nil
}

// CreateAsset creates AWS Asset on fs given AWS account properties
func (s *AWSSaaSConsoleOnboardService) createAsset(ctx context.Context, customerID, accountID string) error {
	assetRef := s.assetsDAL.GetRef(ctx, s.getAssetID(accountID))
	customerRef := s.customersDAL.GetRef(ctx, customerID)

	// contractRef, err := s.contractsDAL.GetCustomerContractRef(ctx, customerRef, string(saasconsole.StandaloneContractTypeAWS))
	// if err != nil {
	// 	return err
	// }

	properties, err := s.getAssetProperties(accountID)
	if err != nil {
		return err
	}

	asset := assets.AWSAsset{
		BaseAsset: assets.BaseAsset{
			AssetType: common.Assets.AmazonWebServicesStandalone,
			Bucket:    nil,
			Contract:  nil, // contract should be added after trial ends
			Entity:    nil,
			Customer:  customerRef,
		},
		Properties: properties,
	}

	if _, err := assetRef.Set(ctx, asset); err != nil {
		return err
	}

	// if err := saasconsole.UpdateContractAssets(ctx, s.contractsDAL, contractRef, assetRef); err != nil {
	// 	return err
	// }

	return nil
}

// CreateCloudConnect creates AWS CloudConnect doc on fs if valid Cost and Usage Report exist on AWS account
func (s *AWSSaaSConsoleOnboardService) createCloudConnect(ctx context.Context, customerID, accountID, role, s3Bucket, curPath, reportName string) error {
	curPath, _ = strings.CutSuffix(curPath, "/")

	fullPath := fmt.Sprintf("%s/%s", curPath, reportName)

	billingEtl := &pkg.BillingEtl{
		Settings: &pkg.BillingEtlSettings{
			Active:      true,
			Bucket:      s3Bucket,
			CurBasePath: fullPath,
			DoitArn:     pkg.StandaloneDoitArn,
		},
	}
	customerRef := s.customersDAL.GetRef(ctx, customerID)
	arn := getRoleArn(role, accountID)

	return s.cloudConnectDAL.CreateAWSCloudConnect(ctx, accountID, arn, customerRef, billingEtl)
}

func (s *AWSSaaSConsoleOnboardService) handleCustomCurAccount(ctx context.Context, customerID string, curInfo saasconsole.AWSCustomCurInfo) error {
	validCURs := []pkg.AWSSaaSConsoleCURPath{
		{
			Bucket:     curInfo.Bucket,
			ReportName: curInfo.ReportPath,
			PathPrefix: curInfo.PathPrefix,
		},
	}

	return retry.BackOffDelay(
		func() error {
			return s.updateCURs(ctx, customerID, curInfo.AccountID, curInfo.Bucket, &validCURs, nil)
		},
		5,
		time.Second,
	)
}
