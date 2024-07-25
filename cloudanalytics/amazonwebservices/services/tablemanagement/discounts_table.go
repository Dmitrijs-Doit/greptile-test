package tablemanagement

import (
	"context"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/utils"
	domainDiscounts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/discounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type DiscountRow struct {
	Customer          string            `json:"customer"`
	Entity            string            `json:"entity"`
	Contract          string            `json:"contract"`
	ProjectID         string            `json:"project_id"`
	IsActive          bool              `json:"is_active"`
	IsCommitment      bool              `json:"is_commitment"`
	ContractStartDate civil.Date        `json:"contract_start_date"`
	ContractEndDate   bigquery.NullDate `json:"contract_end_date"`
	StartDate         civil.Date        `json:"start_date"`
	EndDate           bigquery.NullDate `json:"end_date"`
	Discount          float64           `json:"discount"`
}

func GetAWSDiscountRowFunc(baseRow *domainDiscounts.DiscountRow, billingAccountID string) interface{} {
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
		Discount:          baseRow.Discount,
		ProjectID:         billingAccountID,
	}
}

func (s *BillingTableManagementService) UpdateDiscounts(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	schema := bigquery.Schema{
		{Name: "project_id", Required: true, Type: bigquery.StringFieldType},
	}

	if err := s.discountsService.UpdateDiscounts(ctx, &domainDiscounts.DiscountsTableUpdateData{
		AssetType:              common.Assets.AmazonWebServices,
		AssetPrefix:            "aws",
		GetDiscountRowFunc:     GetAWSDiscountRowFunc,
		DestinationProjectID:   utils.GetBillingProject(),
		DestinationDatasetID:   utils.GetDiscountsDatasetName(),
		DiscountsTableName:     utils.GetDiscountsTableName(),
		AdditionalSchemaFields: schema,
		Clustering:             []string{"project_id"},
	}); err != nil {
		l.Errorf("Error updating discounts: %s\n", err)
		return err
	}

	return nil
}
