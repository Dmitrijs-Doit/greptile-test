package dal

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	cloudtasks "github.com/doitintl/cloudtasks/iface"
	backfill "github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/domain"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/handlers"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type taskConfigKey string

const (
	BackfillSchedulerTaskConfig taskConfigKey = "bq-lens-backfill-scheduler"
	BackfillTaskConfig          taskConfigKey = "bq-lens-backfill-job"
	TableDiscoveryTaskConfig    taskConfigKey = "table-discovery"

	BackfillSchedulerTaskQueueName        = "bq-lens-backfill"
	BackfillSchedulerCloudFunctionURLDev  = "https://scheduled-tasks-dot-doitintl-cmp-dev.appspot.com/tasks/bq-lens/backfill-scheduler"
	BackfillSchedulerCloudFunctionURLProd = "https://scheduled-tasks-dot-me-doit-intl-com.appspot.com/tasks/bq-lens/backfill-scheduler"

	BackfillTaskQueueName        = "bq-lens-backfill"
	BackfillCloudFunctionURLDev  = "https://scheduled-tasks-dot-doitintl-cmp-dev.appspot.com/tasks/bq-lens/backfill"
	BackfillCloudFunctionURLProd = "https://scheduled-tasks-dot-me-doit-intl-com.appspot.com/tasks/bq-lens/backfill"

	TableDiscoveryTaskQueueName        = "bq-lens-tables-discovery"
	TableDiscoveryCloudFunctionURLDev  = "https://scheduled-tasks-dot-doitintl-cmp-dev.appspot.com/tasks/bq-lens/discovery"
	TableDiscoveryCloudFunctionURLProd = "https://scheduled-tasks-dot-me-doit-intl-com.appspot.com/tasks/bq-lens/discovery"
)

type CloudTaskDal struct {
	cloudTaskClient cloudtasks.CloudTaskClient
	tasksConfigs    map[taskConfigKey]common.CloudTaskConfig
}

func NewCloudTaskDal(cloudTaskClient cloudtasks.CloudTaskClient) *CloudTaskDal {
	d := &CloudTaskDal{
		cloudTaskClient: cloudTaskClient,
		tasksConfigs: map[taskConfigKey]common.CloudTaskConfig{
			BackfillSchedulerTaskConfig: {
				Method: cloudtaskspb.HttpMethod_POST,
				Queue:  BackfillSchedulerTaskQueueName,
				URL:    BackfillSchedulerCloudFunctionURLDev,
			},
			BackfillTaskConfig: {
				Method: cloudtaskspb.HttpMethod_POST,
				Queue:  BackfillTaskQueueName,
				URL:    BackfillCloudFunctionURLDev,
			},
			TableDiscoveryTaskConfig: {
				Method: cloudtaskspb.HttpMethod_POST,
				Queue:  TableDiscoveryTaskQueueName,
				URL:    TableDiscoveryCloudFunctionURLDev,
			},
		},
	}

	if common.Production {
		d.tasksConfigs = map[taskConfigKey]common.CloudTaskConfig{
			BackfillSchedulerTaskConfig: {
				Method: cloudtaskspb.HttpMethod_POST,
				Queue:  BackfillSchedulerTaskQueueName,
				URL:    BackfillSchedulerCloudFunctionURLProd,
			},
			BackfillTaskConfig: {
				Method: cloudtaskspb.HttpMethod_POST,
				Queue:  BackfillTaskQueueName,
				URL:    BackfillCloudFunctionURLProd,
			},
			TableDiscoveryTaskConfig: {
				Method: cloudtaskspb.HttpMethod_POST,
				Queue:  TableDiscoveryTaskQueueName,
				URL:    TableDiscoveryCloudFunctionURLProd,
			},
		}
	}

	return d
}

func (d *CloudTaskDal) CreateBackfillScheduleTask(ctx context.Context, sinkID string) error {
	taskcfg, ok := d.tasksConfigs[BackfillSchedulerTaskConfig]
	if !ok {
		return fmt.Errorf("task config not found for %s", BackfillSchedulerTaskConfig)
	}

	payload := struct {
		HandleSpecificSink string `json:"handleSpecificSink"`
		IsTestMode         bool   `json:"isTestMode"`
	}{
		HandleSpecificSink: sinkID,
		IsTestMode:         false,
	}

	if _, err := d.cloudTaskClient.CreateTask(ctx, taskcfg.Config(payload)); err != nil {
		return err
	}

	return nil
}

func (d *CloudTaskDal) CreateBackfillTask(ctx context.Context,
	dateBackfillInfo backfill.DateBackfillInfo,
	backfillDate time.Time,
	backfillProject string,
	customerID string,
	sinkID string,
) error {
	taskcfg, ok := d.tasksConfigs[BackfillTaskConfig]
	if !ok {
		return fmt.Errorf("task config not found for %s", BackfillTaskConfig)
	}

	payload := handlers.BackfillRequest{
		DateBackfillInfo: handlers.BackfillRequestDateBackfillInfo{
			BackfillMinCreationTime: dateBackfillInfo.BackfillMinCreationTime.Format(time.RFC3339),
			BackfillMaxCreationTime: dateBackfillInfo.BackfillMaxCreationTime.Format(time.RFC3339),
		},
		BackfillDate:    backfillDate.Format("2006-01-02"),
		BackfillProject: backfillProject,
		CustomerID:      customerID,
		DocID:           sinkID,
	}

	if _, err := d.cloudTaskClient.CreateTask(ctx, taskcfg.Config(payload)); err != nil {
		return err
	}

	return nil
}

func (d *CloudTaskDal) CreateTableDiscoveryTask(ctx context.Context, customerID string) error {
	taskcfg, ok := d.tasksConfigs[TableDiscoveryTaskConfig]
	if !ok {
		return fmt.Errorf("task config not found for %s", TableDiscoveryTaskConfig)
	}

	taskcfg.URL = fmt.Sprintf("%s/%s", taskcfg.URL, customerID)

	if _, err := d.cloudTaskClient.CreateTask(ctx, taskcfg.Config(nil)); err != nil {
		return err
	}

	return nil
}
