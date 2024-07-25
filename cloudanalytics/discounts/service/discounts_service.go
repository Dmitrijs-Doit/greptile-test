package service

import (
	"context"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/bqutils"
	domainDiscounts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/discounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

type DiscountsService struct {
	conn *connection.Connection
}

func NewDiscountsService(
	conn *connection.Connection,
) *DiscountsService {
	return &DiscountsService{
		conn,
	}
}

func (s *DiscountsService) UpdateDiscounts(ctx context.Context, data *domainDiscounts.DiscountsTableUpdateData) error {
	if data.GetDiscountRowFunc == nil {
		return ErrMissingDiscountAPI
	}

	schema := bigquery.Schema{
		{Name: "customer", Required: true, Type: bigquery.StringFieldType},
		{Name: "entity", Required: true, Type: bigquery.StringFieldType},
		{Name: "contract", Required: true, Type: bigquery.StringFieldType},
		{Name: "is_active", Required: true, Type: bigquery.BooleanFieldType},
		{Name: "is_commitment", Required: true, Type: bigquery.BooleanFieldType},
		{Name: "contract_start_date", Required: true, Type: bigquery.DateFieldType},
		{Name: "contract_end_date", Required: false, Type: bigquery.DateFieldType},
		{Name: "start_date", Required: true, Type: bigquery.DateFieldType},
		{Name: "end_date", Required: false, Type: bigquery.DateFieldType},
		{Name: "discount", Required: true, Type: bigquery.FloatFieldType},
	}
	schema = append(schema, data.AdditionalSchemaFields...)

	discounts, err := s.getDiscounts(ctx, data)
	if err != nil {
		return err
	}

	requestData := bqutils.BigQueryTableLoaderRequest{
		DestinationProjectID:   data.DestinationProjectID,
		DestinationDatasetID:   data.DestinationDatasetID,
		DestinationTableName:   data.DiscountsTableName,
		ObjectDir:              data.AssetPrefix + "_discounts",
		ConfigJobID:            data.AssetPrefix + "_billing_discounts",
		RequirePartitionFilter: true,
		WriteDisposition:       bigquery.WriteTruncate,
		Clustering:             &data.Clustering,
	}

	bqClient := s.conn.Bigquery(ctx)

	loaderAttributes := bqutils.BigQueryTableLoaderParams{
		Client: bqClient,
		Schema: &schema,
		Rows:   discounts,
		Data:   &requestData,
	}

	if discounts != nil {
		if err := bqutils.BigQueryTableLoader(ctx, loaderAttributes); err != nil {
			return err
		}
	}

	return nil
}

// returns discounts data stored in firestore in a form of array of structured data rows
// to be stored in a bigquery table
// depending on the asset type, a single bigquery discount row has different schema
func (s *DiscountsService) getDiscounts(ctx context.Context, data *domainDiscounts.DiscountsTableUpdateData) ([]interface{}, error) {
	fs := s.conn.Firestore(ctx)

	contractSnaps, err := fs.Collection("contracts").
		Where("type", "==", data.AssetType).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	allDiscounts := make([]interface{}, 0)

	for _, contractSnap := range contractSnaps {
		discounts, err := s.getContractDiscounts(ctx, contractSnap, data)
		if err != nil {
			return nil, err
		}

		allDiscounts = append(allDiscounts, discounts...)
	}

	return allDiscounts, nil
}

func (s *DiscountsService) getContractDiscounts(
	ctx context.Context,
	contractSnap *firestore.DocumentSnapshot,
	data *domainDiscounts.DiscountsTableUpdateData,
) ([]interface{}, error) {
	var contract common.Contract
	if err := contractSnap.DataTo(&contract); err != nil {
		return nil, err
	}

	if contract.Customer == nil || contract.Entity == nil {
		return nil, nil
	}

	assetIDs, err := s.getContractAssetIDs(ctx, &contract, data.AssetType)
	if err != nil {
		return nil, err
	}

	if len(assetIDs) <= 0 {
		return nil, nil
	}

	return s.getContractDiscountRows(ctx, &contract, assetIDs, contractSnap.Ref.ID, data), nil
}

func (s *DiscountsService) getContractAssetIDs(
	ctx context.Context,
	contract *common.Contract,
	assetType string,
) ([]string, error) {
	assetPrefixLen := len(assetType) + 1
	assetIDs := make([]string, 0)
	fs := s.conn.Firestore(ctx)

	if len(contract.Assets) > 0 {
		for _, assetRef := range contract.Assets {
			assetIDs = append(assetIDs, assetRef.ID[assetPrefixLen:])
		}
	} else {
		docSnaps, err := fs.Collection("assets").
			Where("type", "==", assetType).
			Where("customer", "==", contract.Customer).
			Where("entity", "==", contract.Entity).
			Select().Documents(ctx).GetAll()
		if err != nil {
			return nil, err
		}

		for _, docSnap := range docSnaps {
			assetIDs = append(assetIDs, docSnap.Ref.ID[assetPrefixLen:])
		}
	}

	return assetIDs, nil
}

func (s *DiscountsService) getContractDiscountRows(
	ctx context.Context,
	contract *common.Contract,
	assetIDs []string,
	contractRefID string,
	data *domainDiscounts.DiscountsTableUpdateData,
) []interface{} {
	var discounts []interface{}

	discount := contract.Discount
	discountPreemptible, _ := contract.GetBoolProperty("discountPreemptible", false)
	rebaseModifier, _ := contract.GetFloatProperty("rebaseModifier", 0)
	contractStartDate := civil.DateOf(contract.StartDate)
	contractEndDate := bigquery.NullDate{Valid: false}

	if contract.IsCommitment {
		if !contract.EndDate.IsZero() {
			contractEndDate = bigquery.NullDate{
				Date:  civil.DateOf(contract.EndDate),
				Valid: true,
			}
		}
	} else if contract.DiscountEndDate != nil {
		contractEndDate = bigquery.NullDate{
			Date:  civil.DateOf(*contract.DiscountEndDate),
			Valid: true,
		}
	}

	for _, assetID := range assetIDs {
		dr := domainDiscounts.DiscountRow{
			Customer:          contract.Customer.ID,
			Entity:            contract.Entity.ID,
			Contract:          contractRefID,
			IsActive:          contract.Active,
			IsCommitment:      contract.IsCommitment,
			ContractStartDate: contractStartDate,
			ContractEndDate:   contractEndDate,
			StartDate:         contractStartDate,
			EndDate:           contractEndDate,
			AllowPreemptible:  discountPreemptible,
			Discount:          toProportion(discount),
			RebaseModifier:    toProportion(rebaseModifier),
		}

		if contract.ShouldUseCommitmentPeriodDiscounts() {
			for _, cp := range contract.CommitmentPeriods {
				dr.Discount = toProportion(cp.Discount)
				dr.StartDate = civil.DateOf(cp.StartDate)
				dr.EndDate = bigquery.NullDate{
					Date:  civil.DateOf(cp.EndDate),
					Valid: true,
				}

				discountRow := data.GetDiscountRowFunc(&dr, assetID)
				discounts = append(discounts, discountRow)
			}
		} else {
			discountRow := data.GetDiscountRowFunc(&dr, assetID)
			discounts = append(discounts, discountRow)
		}
	}

	return discounts
}

func toProportion(discount float64) float64 {
	return 1 - discount*0.01
}
