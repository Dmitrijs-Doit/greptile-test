package domain

import "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/iface"

const (
	FlexsaveSageMakerReasonCantEnableNoBillingTable iface.FlexsaveSageMakerReasonCantEnable = "no_billing_table"
	FailedRecommendationProcess                     iface.FlexsaveSageMakerReasonCantEnable = "failed_recommendation_process"
	NoSpend                                         iface.FlexsaveSageMakerReasonCantEnable = "no_spend"
	LowSpend                                        iface.FlexsaveSageMakerReasonCantEnable = "low_spend"

	SageMakerMinHourlyCommitment = 1.0
)
