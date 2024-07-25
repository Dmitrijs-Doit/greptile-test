package fanout

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"

	"github.com/doitintl/cloudtasks/iface"
	"github.com/doitintl/errors"
	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/assets"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Service interface {
	CreateCacheForAllCustomers(ctx context.Context) error
	CreateSavingsPlansCacheForAllCustomers(ctx context.Context) error
}

type service struct {
	loggerProvider   logger.Provider
	customersDAL     customerDal.Customers
	cloudTaskService iface.CloudTaskClient
	assetsDal        assetsDal.Assets
	integrationsDAL  fsdal.Integrations
}

func NewFanoutService(log logger.Provider, conn *connection.Connection) Service {
	assetsDAL := assetsDal.NewAssetsFirestoreWithClient(conn.Firestore)
	integrationsDAL := fsdal.NewIntegrationsDALWithClient(conn.Firestore(context.Background()))

	return service{
		log,
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		conn.CloudTaskClient,
		assetsDAL,
		integrationsDAL,
	}
}

func (s service) CreateSavingsPlansCacheForAllCustomers(ctx context.Context) error {
	customerIDs, err := s.integrationsDAL.GetAWSEligibleCustomerIDs(ctx)
	if err != nil {
		return err
	}

	log := s.loggerProvider(ctx)

	log.Infof("running savings plan cache job for %d customers", len(customerIDs))

	var failedCustomers []string

	for _, ID := range customerIDs {
		path := fmt.Sprintf("/tasks/flex-ri/savings-plans-cache/%s", ID)

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_GET,
			Path:   path,
			Queue:  common.TaskQueueFlexsaveAWSSavingsPlansCache,
		}

		conf := config.Config(nil)

		if _, err := s.cloudTaskService.CreateTask(ctx, conf); err != nil {
			log.Errorf("unable to create savings plan cache generation task for customer: %s", ID)
			failedCustomers = append(failedCustomers, ID)
		}
	}

	if len(failedCustomers) == 0 {
		return nil
	}

	return fmt.Errorf("failed generating jobs for %v", strings.Join(failedCustomers, ", "))
}

func (s service) CreateCacheForAllCustomers(ctx context.Context) error {
	log := s.loggerProvider(ctx)

	snaps, err := s.customersDAL.GetAWSCustomers(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get aws customers for cache fan-out")
	}

	var customerRefs []*firestore.DocumentRef

	for _, s := range snaps {
		customerRefs = append(customerRefs, s.Ref)
	}

	s.runComputeCache(ctx, log, customerRefs)

	s.runSageMakerCache(ctx, log, customerRefs)

	s.runRDSCache(ctx, log, customerRefs)

	return nil
}

func (s service) runComputeCache(ctx context.Context, log logger.ILogger, customerRefs []*firestore.DocumentRef) {
	log.Infof("running compute cache job for %d customers", len(customerRefs))

	var failedCustomers []string

	for _, ref := range customerRefs {
		customerID := ref.ID

		configInfo, err := s.integrationsDAL.GetFlexsaveConfigurationCustomer(ctx, customerID)
		if err != nil && !errors.Is(err, fsdal.ErrNotFound) {
			log.Error(err)
			continue
		}

		if configInfo != nil && configInfo.AWS.TimeDisabled != nil {
			continue
		}

		standaloneAssets, err := s.assetsDal.GetAWSStandaloneAssets(ctx, ref)
		if err != nil {
			log.Error(err)
			continue
		}

		if assets.HasAWSStandaloneFlexsave(standaloneAssets) {
			continue
		}

		path := fmt.Sprintf("/tasks/flex-ri/cache/%s", customerID)

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_GET,
			Path:   path,
			Queue:  common.TaskQueueFlexsaveAWSCache,
		}

		conf := config.Config(nil)

		if _, err := s.cloudTaskService.CreateTask(ctx, conf); err != nil {
			log.Errorf("unable to create cache generation job for customer %s", customerID)
			failedCustomers = append(failedCustomers, customerID)
		}
	}

	if len(failedCustomers) == 0 {
		return
	}

	log.Errorf("failed generating compute cache jobs for %v", strings.Join(failedCustomers, ", "))
}

func (s service) runSageMakerCache(ctx context.Context, log logger.ILogger, customerRefs []*firestore.DocumentRef) {
	log.Infof("running sagemaker cache for %d customers", len(customerRefs))

	var failedCustomers []string

	for _, ref := range customerRefs {
		customerID := ref.ID

		path := fmt.Sprintf("/tasks/flexsave-sagemaker/run-cache/%s", customerID)

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_GET,
			Path:   path,
			Queue:  common.TaskQueueFlexsaveAWSCache,
		}

		conf := config.Config(nil)

		if _, err := s.cloudTaskService.CreateTask(ctx, conf); err != nil {
			log.Errorf("sagemaker cache failed for: %s", customerID)
			failedCustomers = append(failedCustomers, customerID)
		}
	}

	if len(failedCustomers) == 0 {
		return
	}

	log.Errorf("failed generating sagemaker cache jobs for %v", strings.Join(failedCustomers, ", "))
}

func (s service) runRDSCache(ctx context.Context, log logger.ILogger, customerRefs []*firestore.DocumentRef) {
	log.Infof("running RDS cache for %d customers", len(customerRefs))

	var failedCustomers []string

	for _, ref := range customerRefs {
		customerID := ref.ID

		path := fmt.Sprintf("/tasks/flexsave-rds/run-cache/%s", customerID)

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_GET,
			Path:   path,
			Queue:  common.TaskQueueFlexsaveAWSCache,
		}

		conf := config.Config(nil)

		if _, err := s.cloudTaskService.CreateTask(ctx, conf); err != nil {
			log.Errorf("RDS cache failed for: %s", customerID)
			failedCustomers = append(failedCustomers, customerID)
		}
	}

	if len(failedCustomers) == 0 {
		return
	}

	log.Errorf("failed generating rds cache jobs for %v", strings.Join(failedCustomers, ", "))
}
