package googlecloud

import (
	"context"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"

	domainDiscounts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/discounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type DiscountRow struct {
	Customer          string            `json:"customer"`
	Entity            string            `json:"entity"`
	Contract          string            `json:"contract"`
	BillingAccountID  string            `json:"billing_account_id"`
	IsActive          bool              `json:"is_active"`
	IsCommitment      bool              `json:"is_commitment"`
	ContractStartDate civil.Date        `json:"contract_start_date"`
	ContractEndDate   bigquery.NullDate `json:"contract_end_date"`
	StartDate         civil.Date        `json:"start_date"`
	EndDate           bigquery.NullDate `json:"end_date"`
	AllowPreemptible  bool              `json:"allow_preemptible"`
	Discount          float64           `json:"discount"`
	RebaseModifier    float64           `json:"rebase_modifier"`
}

func GetDiscountsTableName() string {
	if common.Production {
		return "gcp_discounts_v1"
	}

	return "gcp_discounts_v1beta"
}

func GetGCPDiscountRowFunc(baseRow *domainDiscounts.DiscountRow, billingAccountID string) interface{} {
	return &DiscountRow{
		Customer:          baseRow.Customer,
		Entity:            baseRow.Entity,
		Contract:          baseRow.Contract,
		IsActive:          baseRow.IsActive,
		IsCommitment:      baseRow.IsCommitment,
		ContractStartDate: baseRow.ContractStartDate,
		ContractEndDate:   baseRow.ContractEndDate,
		StartDate:         baseRow.StartDate,
		EndDate:           baseRow.EndDate,
		AllowPreemptible:  baseRow.AllowPreemptible,
		Discount:          baseRow.Discount,
		RebaseModifier:    baseRow.RebaseModifier,
		BillingAccountID:  billingAccountID,
	}
}

func (s *BillingTableManagementService) UpdateDiscounts(ctx context.Context) error {
	schema := bigquery.Schema{
		{Name: "billing_account_id", Required: true, Type: bigquery.StringFieldType},
		{Name: "rebase_modifier", Required: true, Type: bigquery.FloatFieldType},
		{Name: "allow_preemptible", Required: true, Type: bigquery.BooleanFieldType},
	}

	return s.discountsService.UpdateDiscounts(ctx, &domainDiscounts.DiscountsTableUpdateData{
		AssetType:              common.Assets.GoogleCloud,
		AssetPrefix:            "gcp",
		GetDiscountRowFunc:     GetGCPDiscountRowFunc,
		DestinationProjectID:   domain.GetBillingProject(),
		DestinationDatasetID:   domain.BillingDataset,
		DiscountsTableName:     GetDiscountsTableName(),
		AdditionalSchemaFields: schema,
		Clustering:             []string{"billing_account_id"},
	})
}
