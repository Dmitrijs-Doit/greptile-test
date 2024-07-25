package domain

import (
	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
)

type GetAssetTypeDiscountRowFn func(*DiscountRow, string) interface{}

type DiscountsTableUpdateData struct {
	AssetType              string
	AssetPrefix            string
	GetDiscountRowFunc     GetAssetTypeDiscountRowFn
	DestinationProjectID   string
	DestinationDatasetID   string
	DiscountsTableName     string
	AdditionalSchemaFields []*bigquery.FieldSchema
	Clustering             []string
}

type DiscountRow struct {
	Customer          string
	Entity            string
	Contract          string
	IsActive          bool
	IsCommitment      bool
	ContractStartDate civil.Date
	ContractEndDate   bigquery.NullDate
	StartDate         civil.Date
	EndDate           bigquery.NullDate
	AllowPreemptible  bool
	Discount          float64
	RebaseModifier    float64
}
