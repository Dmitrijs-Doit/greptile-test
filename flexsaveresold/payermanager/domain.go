package payermanager

import "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"

type FormEntry struct {
	Status                      string           `json:"status" binding:"required"`
	Managed                     string           `json:"managed" binding:"required"`
	MinSpend                    *float64         `json:"minSpend"`
	MaxSpend                    *float64         `json:"maxSpend"`
	Discount                    []types.Discount `json:"discount"`
	Seasonal                    *bool            `json:"seasonal"`
	RDSStatus                   *string          `json:"rdsStatus"`
	SagemakerStatus             *string          `json:"sagemakerStatus"`
	TargetPercentage            *float64         `json:"targetPercentage"`
	StatusChangeReason          *string          `json:"statusChangeReason"`
	KeepActiveEvenWhenOnCredits bool             `json:"keepActiveEvenWhenOnCredits"`
	RDSTargetPercentage         *float64         `json:"rdsTargetPercentage"`
}
