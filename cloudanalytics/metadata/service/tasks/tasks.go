package service

import (
	"context"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

func CreateUpdateGCPAccountMetadataTask(ctx context.Context, conn *connection.Connection, body metadata.MetadataUpdateInput, accountID string, scheduleTime *time.Time) error {
	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   "/tasks/analytics/" + common.Assets.GoogleCloud + "/accounts/" + accountID + "/metadata",
		Queue:  common.TaskQueueCloudAnalyticsMetadataGCP,
	}

	if scheduleTime != nil && !scheduleTime.IsZero() {
		config.ScheduleTime = timestamppb.New(*scheduleTime)
	}

	if _, err := conn.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(body)); err != nil {
		return err
	}

	return nil
}
