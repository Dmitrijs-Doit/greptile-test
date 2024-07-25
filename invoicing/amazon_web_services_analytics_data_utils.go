package invoicing

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
)

const (
	indexYear = iota
	indexMonth
	indexDay
	indexLabel
	indexAccount
	indexCostType
	indexMarketplaceSD
	indexPayer
	indexIsMarketplace
	indexCost
	indexUsage
	indexSavings
)

type BillingDataQuery interface {
	GetBillingDataQuery(ctx context.Context, invoiceMonth time.Time, accounts []string, provider string) (*cloudanalytics.QueryRequest, error)
	GetBillingQueryFilters(provider string, costTypeFilters []string, costTypeInverse bool, systemLabelKey string, systemLabelFilters []string, systemLabelInverse bool) ([]*domainQuery.QueryRequestX, error)
	GetTimeSettings(month time.Time) *cloudanalytics.QueryRequestTimeSettings
}

type BillingDataQueryBuilder struct{}

func (builder *BillingDataQueryBuilder) GetBillingDataQuery(ctx context.Context, invoiceMonth time.Time, accounts []string, provider string) (*cloudanalytics.QueryRequest, error) {
	filters, err := builder.GetBillingQueryFilters(provider, []string{utils.BillingFlexsaveCostType}, true, DoitDataSource, []string{CustomerBackfill}, true)
	if err != nil {
		return nil, err
	}

	rows, err := builder.getBillingQueryRequestRows()
	if err != nil {
		return nil, err
	}

	cols, err := builder.getBillingQueryRequestCols()
	if err != nil {
		return nil, err
	}

	attributions, err := builder.generateComputeSp3yrNoUpfrontSkuAttribution()
	if err != nil {
		return nil, err
	}

	attributionGroups1, err := builder.generateFlexsaveNegationAttributionGroup()
	if err != nil {
		return nil, err
	}

	attributionGroups2, err := builder.generateIsMarketplaceAttributionGroup()
	if err != nil {
		return nil, err
	}

	attributionGroups := append(attributionGroups1, attributionGroups2...)

	qr := cloudanalytics.QueryRequest{
		Origin:         domainOrigin.QueryOriginFromContext(ctx),
		Type:           "report",
		CloudProviders: &[]string{provider},
		Accounts:       accounts,
		TimeSettings:   builder.GetTimeSettings(invoiceMonth),
		Rows:           rows,
		Cols:           cols,
		Filters:        filters,
		MetricFiltres: []*domainQuery.QueryRequestMetricFilter{
			{
				Metric:   report.MetricCost,
				Operator: report.MetricFilterNotBetween,
				Values:   []float64{-0.0001, 0.0001},
			},
		},
		Currency:          fixer.USD,
		LogScale:          false,
		IsPreset:          false,
		Organization:      nil,
		NoAggregate:       true,
		Attributions:      attributions,
		AttributionGroups: attributionGroups, //append(attributionGroups1, attributionGroups2...),
	}

	return &qr, nil
}

func (builder *BillingDataQueryBuilder) generateComputeSp3yrNoUpfrontSkuAttribution() ([]*domainQuery.QueryRequestX, error) {
	computeSpSkuFilter, err := domainQuery.NewFilter(domainQuery.FieldSKUDescription, domainQuery.WithValues([]string{SkuComputeSp3yrNoUpfront}))
	if err != nil {
		return nil, err
	}

	return []*domainQuery.QueryRequestX{
		{
			ID:              fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, SkuComputeSp3yrNoUpfrontFmt),
			Type:            metadata.MetadataFieldTypeAttribution,
			Key:             SkuComputeSp3yrNoUpfrontFmt,
			IncludeInFilter: false,
			Composite:       []*domainQuery.QueryRequestX{computeSpSkuFilter},
			Formula:         "A",
		},
	}, nil
}

func (builder *BillingDataQueryBuilder) generateFlexsaveNegationAttributionGroup() ([]*domainQuery.AttributionGroupQueryRequest, error) {
	creditCostTypeFilter, err := domainQuery.NewFilter(domainQuery.FieldCostType, domainQuery.WithValues([]string{CostTypeCredit}))
	if err != nil {
		return nil, err
	}

	flexsaveNegationCostTypeFilter, err := domainQuery.NewFilter(domainQuery.FieldCostType, domainQuery.WithValues([]string{CostTypeFlexsaveNegation}))
	if err != nil {
		return nil, err
	}

	flexsaveAdjustmentCostTypeFilter, err := domainQuery.NewFilter(domainQuery.FieldCostType, domainQuery.WithValues([]string{CostTypeFlexsaveAdjustment}))
	if err != nil {
		return nil, err
	}

	flexsaveManagementFeeCostTypeFilter, err := domainQuery.NewFilter(domainQuery.FieldCostType, domainQuery.WithValues([]string{CostTypeFlexsaveManagementFee}))
	if err != nil {
		return nil, err
	}

	flexsaveRIFeeCostTypeFilter, err := domainQuery.NewFilter(domainQuery.FieldCostType, domainQuery.WithValues([]string{CostTypeFlexsaveRIFee}))
	if err != nil {
		return nil, err
	}

	flexsaveRefundCostTypeFilter, err := domainQuery.NewFilter(domainQuery.FieldCostType, domainQuery.WithValues([]string{CostTypeFlexsaveRefund}))
	if err != nil {
		return nil, err
	}

	amazonSageMakerServiceFilter, err := domainQuery.NewFilter(domainQuery.FieldServiceID, domainQuery.WithValues([]string{ServiceIdAmazonSageMaker}))
	if err != nil {
		return nil, err
	}

	amazonRDSServiceFilter, err := domainQuery.NewFilter(domainQuery.FieldServiceID, domainQuery.WithValues([]string{ServiceIdAmazonRDS}))
	if err != nil {
		return nil, err
	}

	return []*domainQuery.AttributionGroupQueryRequest{
		{
			QueryRequestX: domainQuery.QueryRequestX{
				ID:   fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttributionGroup, CustomCostType),
				Type: metadata.MetadataFieldTypeAttributionGroup,
				Key:  CustomCostType,
			},
			Attributions: []*domainQuery.QueryRequestX{
				{
					ID:              fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, CostTypeCredit),
					Type:            metadata.MetadataFieldTypeAttribution,
					Key:             CostTypeCredit,
					IncludeInFilter: false,
					Composite:       []*domainQuery.QueryRequestX{creditCostTypeFilter},
					Formula:         "A",
				},
				{
					ID:              fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, CostTypeFlexsaveSagemakerNegation),
					Type:            metadata.MetadataFieldTypeAttribution,
					Key:             CostTypeFlexsaveSagemakerNegation,
					IncludeInFilter: false,
					Composite:       []*domainQuery.QueryRequestX{flexsaveNegationCostTypeFilter, amazonSageMakerServiceFilter},
					Formula:         "A AND B",
				},
				{
					ID:              fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, CostTypeFlexsaveRDSManagementFee),
					Type:            metadata.MetadataFieldTypeAttribution,
					Key:             CostTypeFlexsaveRDSManagementFee,
					IncludeInFilter: false,
					Composite:       []*domainQuery.QueryRequestX{flexsaveManagementFeeCostTypeFilter, amazonRDSServiceFilter},
					Formula:         "A AND B",
				},
				{
					ID:              fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, CostTypeFlexsaveRDSCharges),
					Type:            metadata.MetadataFieldTypeAttribution,
					Key:             CostTypeFlexsaveRDSCharges,
					IncludeInFilter: false,
					Composite:       []*domainQuery.QueryRequestX{flexsaveRIFeeCostTypeFilter, flexsaveRefundCostTypeFilter},
					Formula:         "A OR B",
				},
				{
					ID:              fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, CostTypeFlexsaveComputeNegation),
					Type:            metadata.MetadataFieldTypeAttribution,
					Key:             CostTypeFlexsaveComputeNegation,
					IncludeInFilter: false,
					Composite:       []*domainQuery.QueryRequestX{flexsaveNegationCostTypeFilter},
					Formula:         "A",
				},
				{
					ID:              fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, CostTypeFlexsaveAdjustment),
					Type:            metadata.MetadataFieldTypeAttribution,
					Key:             CostTypeFlexsaveAdjustment,
					IncludeInFilter: false,
					Composite:       []*domainQuery.QueryRequestX{flexsaveAdjustmentCostTypeFilter},
					Formula:         "A",
				},
			},
		},
	}, nil
}

//	 CASE
//		  WHEN (T.is_marketplace IN UNNEST(ARRAY(SELECT CAST(value AS BOOL) FROM UNNEST(['true']) value))) THEN T.service_description
//	 ELSE NULL
//	 END AS service_description,
func (builder *BillingDataQueryBuilder) generateIsMarketplaceAttributionGroup() ([]*domainQuery.AttributionGroupQueryRequest, error) {
	amazonMarketPlaceFilter, err := domainQuery.NewFilter(domainQuery.FieldIsMarketplace, domainQuery.WithValues([]string{"true"}))
	if err != nil {
		return nil, err
	}

	return []*domainQuery.AttributionGroupQueryRequest{
		{
			QueryRequestX: domainQuery.QueryRequestX{
				ID:   fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttributionGroup, MarketplaceServiceDescription),
				Type: metadata.MetadataFieldTypeAttributionGroup,
				Key:  MarketplaceServiceDescription,
			},
			Attributions: []*domainQuery.QueryRequestX{
				{
					ID:              fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, MarketplaceServiceDescription),
					Type:            metadata.MetadataFieldTypeAttribution,
					Key:             MarketplaceServiceDescription,
					AbsoluteKey:     metadata.MetadataFieldKeyServiceDescription,
					IncludeInFilter: false,
					Composite:       []*domainQuery.QueryRequestX{amazonMarketPlaceFilter},
					Formula:         "A",
				},
			},
		},
	}, nil
}

func (builder *BillingDataQueryBuilder) GetBillingQueryFilters(provider string, costTypeFilterValues []string, costTypeInverse bool, systemLabelKey string, systemLabelFilterValues []string, systemLabelInverse bool) ([]*domainQuery.QueryRequestX, error) {
	awsProviderFilter, err := domainQuery.NewFilter(domainQuery.FieldCloudProvider, domainQuery.WithValues([]string{provider}))
	if err != nil {
		return nil, err
	}

	costTypeOptions := []domainQuery.QueryRequestXOption{domainQuery.WithValues(costTypeFilterValues)}
	if costTypeInverse {
		costTypeOptions = append(costTypeOptions, domainQuery.WithInverse())
	}

	costTypeFilter, err := domainQuery.NewFilter(domainQuery.FieldCostType, costTypeOptions...)
	if err != nil {
		return nil, err
	}

	filters := []*domainQuery.QueryRequestX{awsProviderFilter, costTypeFilter}

	if systemLabelKey != "" && len(systemLabelFilterValues) > 0 {
		systemLabelOptions := []domainQuery.QueryRequestXOption{domainQuery.WithKey(systemLabelKey), domainQuery.WithValues(systemLabelFilterValues)}
		if systemLabelInverse {
			systemLabelOptions = append(systemLabelOptions, domainQuery.WithInverse())
		}

		systemLabelFilter, err := domainQuery.NewFilter(domainQuery.FieldSystemLabels, systemLabelOptions...)
		if err != nil {
			return nil, err
		}

		filters = append(filters, systemLabelFilter)
	}

	return filters, nil
}

func (builder *BillingDataQueryBuilder) getBillingQueryRequestRows() ([]*domainQuery.QueryRequestX, error) {
	year, err := domainQuery.NewRow("year")
	if err != nil {
		return nil, err
	}

	month, err := domainQuery.NewRow("month")
	if err != nil {
		return nil, err
	}

	day, err := domainQuery.NewRow("day")
	if err != nil {
		return nil, err
	}

	attrLabel, err := domainQuery.NewRow("attribution")
	if err != nil {
		return nil, err
	}

	return []*domainQuery.QueryRequestX{year, month, day, attrLabel}, nil
}

func (builder *BillingDataQueryBuilder) getBillingQueryRequestCols() ([]*domainQuery.QueryRequestX, error) {
	projectID, err := domainQuery.NewCol("project_id")
	if err != nil {
		return nil, err
	}

	attributionGroup1, err := domainQuery.NewAttributionGroupField(string(metadata.MetadataFieldTypeAttributionGroup), CustomCostType, domainQuery.QueryFieldPositionCol)
	if err != nil {
		return nil, err
	}

	attributionGroup2, err := domainQuery.NewAttributionGroupField(string(metadata.MetadataFieldTypeAttributionGroup), MarketplaceServiceDescription, domainQuery.QueryFieldPositionCol)
	if err != nil {
		return nil, err
	}

	payerAccountID, err := domainQuery.NewColConstituentField("system_labels", "aws/payer_account_id")
	if err != nil {
		return nil, err
	}

	isMarketPlace, err := domainQuery.NewCol("is_marketplace")
	if err != nil {
		return nil, err
	}

	return []*domainQuery.QueryRequestX{projectID, attributionGroup1, attributionGroup2, payerAccountID, isMarketPlace}, nil
}

func (builder *BillingDataQueryBuilder) GetTimeSettings(month time.Time) *cloudanalytics.QueryRequestTimeSettings {
	from := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, -1)

	return &cloudanalytics.QueryRequestTimeSettings{
		Interval: "day",
		From:     &from,
		To:       &to,
	}
}

type billingDataResult interface {
	TransformToDaysToAccountsToCostAndAccountIDs(rows [][]bigquery.Value) (map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem, []string, error)
}
type queryResultTransformer struct {
}

// TransformToDaysToAccountsToCostAndAccountIDs expects row:
// 0: "2022"
// 1: "01"
// 2: "01"
// 3: nil OR "ComputeSP_3yrNoUpfront"
// 4: "001404415847"
// 5: "usage"
// 6: nil OR "marketplace mangoDB"
// 7: "030555939678" // payer-account-id
// 8: true/false // isMarketplace
// 9: 0.04450553890000001 // cost
// 10: 32590.071414319
// 9: 0 // savings

func (data *queryResultTransformer) TransformToDaysToAccountsToCostAndAccountIDs(rows [][]bigquery.Value) (map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem, []string, error) {
	daysToAccountsToCost := map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{}
	accountIDs := make([]string, 0)
	uniqueAccountsMap := map[string]bool{}

	for _, row := range rows {
		day, err := data.getRowDate(row)
		if err != nil {
			return nil, nil, err
		}

		label := ""
		if row[indexLabel] != nil {
			label, err = query.BigqueryValueToString(row[indexLabel])
			if err != nil {
				return nil, nil, err
			}
		}

		accountID, err := query.BigqueryValueToString(row[indexAccount])
		if err != nil {
			return nil, nil, err
		}

		costType := ""
		if row[indexCostType] != nil {
			costType, err = query.BigqueryValueToString(row[indexCostType])
			if err != nil {
				return nil, nil, err
			}
		}

		payerAccountID := ""
		if row[indexPayer] != nil {
			payerAccountID, err = query.BigqueryValueToString(row[indexPayer])
			if err != nil {
				return nil, nil, err
			}
		}

		var isMarketplace bool
		switch value := row[indexIsMarketplace].(type) {
		case bool:
			isMarketplace = value
		default:
			return nil, nil, fmt.Errorf("unexpected type marketplace in row[%d], expected bool, actual %v", indexIsMarketplace, value)
		}

		marketplaceServiceDescription := ""
		if row[indexMarketplaceSD] != nil {
			marketplaceServiceDescription, err = query.BigqueryValueToString(row[indexMarketplaceSD])
			if err != nil {
				return nil, nil, err
			}
		}

		var cost float64
		switch value := row[indexCost].(type) {
		case float64:
			cost = value
		default:
			return nil, nil, fmt.Errorf("unexpected type cost in row[%d], expected float64, actual %v", indexCost, value)
		}

		var savings float64
		switch value := row[indexSavings].(type) {
		case float64:
			savings = value
		default:
			return nil, nil, fmt.Errorf("unexpected type savings in row[%d], expected float64, actual %v", indexSavings, value)
		}

		var flexsaveComputeNegation, flexsaveSagemakerNegation, flexsaveRDSNegation, flexsaveAdjustment, flexsaveRDSCharges float64
		if costType == CostTypeFlexsaveComputeNegation {
			flexsaveComputeNegation = cost
		} else if costType == CostTypeFlexsaveSagemakerNegation {
			flexsaveSagemakerNegation = cost
		} else if costType == CostTypeFlexsaveRDSManagementFee {
			flexsaveRDSNegation = -savings
		} else if costType == CostTypeFlexsaveAdjustment {
			flexsaveAdjustment = cost
		} else if costType == CostTypeFlexsaveRDSCharges {
			flexsaveRDSCharges = cost
		}
		dailyAccountToCost := daysToAccountsToCost[*day]
		if dailyAccountToCost == nil {
			dailyAccountToCost = map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{}
			daysToAccountsToCost[*day] = dailyAccountToCost
		}

		mapKey := pkg.CostAndSavingsAwsLineItemKey{
			AccountID:      accountID,
			PayerAccountID: payerAccountID,
			CostType:       costType,
			Label:          label,
			MarketplaceSD:  marketplaceServiceDescription,
			IsMarketplace:  isMarketplace,
		}

		if dailyAccountCost, ok := dailyAccountToCost[mapKey]; ok {
			dailyAccountCost.Costs += cost
			dailyAccountCost.Savings += savings
			dailyAccountCost.FlexsaveComputeNegations += flexsaveComputeNegation
			dailyAccountCost.FlexsaveSagemakerNegations += flexsaveSagemakerNegation
			dailyAccountCost.FlexsaveRDSNegations += flexsaveRDSNegation
			dailyAccountCost.FlexsaveAdjustments += flexsaveAdjustment
			dailyAccountCost.FlexsaveRDSCharges += flexsaveRDSCharges
		} else {
			dailyAccountToCost[mapKey] = &pkg.CostAndSavingsAwsLineItem{
				Costs:                      cost,
				Savings:                    savings,
				FlexsaveComputeNegations:   flexsaveComputeNegation,
				FlexsaveSagemakerNegations: flexsaveSagemakerNegation,
				FlexsaveRDSNegations:       flexsaveRDSNegation,
				FlexsaveAdjustments:        flexsaveAdjustment,
				FlexsaveRDSCharges:         flexsaveRDSCharges,
			}
		}

		if _, ok := uniqueAccountsMap[accountID]; !ok {
			uniqueAccountsMap[accountID] = true

			accountIDs = append(accountIDs, accountID)
		}
	}

	return daysToAccountsToCost, accountIDs, nil
}

func (data *queryResultTransformer) getRowDate(row []bigquery.Value) (*time.Time, error) {
	year, err := data.BigQueryValueToInt(row[indexYear])
	if err != nil {
		return nil, err
	}

	month, err := data.BigQueryValueToInt(row[indexMonth])
	if err != nil {
		return nil, err
	}

	day, err := data.BigQueryValueToInt(row[indexDay])
	if err != nil {
		return nil, err
	}

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	return &date, nil
}

func (data *queryResultTransformer) BigQueryValueToInt(row bigquery.Value) (int, error) {
	asString, err := query.BigqueryValueToString(row)
	if err != nil {
		return 0, err
	}

	asInt, err := strconv.Atoi(asString)
	if err != nil {
		return 0, err
	}

	return asInt, nil
}
