package scripts

import (
	"encoding/json"
	"fmt"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/errors"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func createFlexsaveInvoiceAdjustment(ctx *gin.Context) []error {
	payload := struct {
		Month        string             `json:"month"`
		TotalSavings float64            `json:"totalSavings"`
		Entities     map[string]float64 `json:"entities"`
	}{
		Month:        "2023-11-01",
		TotalSavings: -528.95,
		Entities: map[string]float64{
			"4Bzh4kKq6lKi0oTc4c6j": -528.95,
			"zV5f1k0xdVURZuIeJCCn": 0,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return []error{errors.Wrapf(err, "failed to parse payload to json %+v", payload)}
	}

	customerID := "S3Rv1xgKtJOpwhKzDBni"

	task := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   fmt.Sprintf("/tasks/flex-ri/backfill-savings/customer/%s", customerID),
		Queue:  common.TaskQueueFlexsaveInvoiceAdjustment,
		Body:   body,
	}

	if _, err := common.CreateCloudTask(ctx, &task); err != nil {
		return []error{errors.Wrapf(err, "failed to create cloud task")}
	}

	return nil
}
