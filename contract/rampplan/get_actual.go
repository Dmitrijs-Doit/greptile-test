package rampplan

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	attributionGroupsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"google.golang.org/api/iterator"
)

type Spend struct {
	Services map[string]float64
	Total    float64
}

// the percent from the total commitment value of marketplace spend that is covered
var marketplaceCoveredPercent = map[string]float64{
	common.Assets.GoogleCloud:       0.5,
	common.Assets.AmazonWebServices: 0.25,
}

func (s *Service) GetActualMonthlySpend(ctx context.Context, rp pkg.RampPlan) ([]map[pkg.YearMonth]Spend, error) {
	periodsSpends := make([]map[pkg.YearMonth]Spend, 0)
	logger := s.Logger(ctx)

	if rp.AttributionGroup == nil {
		return nil, fmt.Errorf("ramp plan %s has no attribution group", rp.Ref.ID)
	}

	ag, ats, err := getAttributionGroupWithAttributions(ctx, s.conn.Firestore, rp.AttributionGroup.ID)
	if err != nil {
		logger.Errorf("Failed to get attribution group with attributions %s: %v", rp.AttributionGroup.ID, err)
		return nil, err
	}

	accounts, err := getAccounts(ctx, s.conn.Firestore, rp.ContractID, rp.ContractEntity)
	if err != nil {
		return nil, err
	}

	marketplaceCoveredPercent, ok := marketplaceCoveredPercent[rp.Platform]
	if !ok {
		logger.Errorf("Failed to get marketplace covered percent for platform %s", rp.Platform)
	}

	commitmentValue, err := getCommitmentValue(ctx, s.conn.Firestore, rp.ContractID)
	if err != nil {
		logger.Errorf("Failed to get commitment value for contract %s: %v", rp.ContractID, err)
		return nil, err
	}

	maxMarketplaceCover := commitmentValue * marketplaceCoveredPercent

	attributionNames := make(map[string]bool)
	for _, attribution := range ats {
		attributionNames[attribution.Name] = true
	}

	periods := rp.CommitmentPeriods

	var currMarketplaceTotal float64

	now := time.Now()
	for _, period := range periods {
		if period.StartDate.After(now) {
			periodsSpends = append(periodsSpends, make(map[pkg.YearMonth]Spend))
			continue
		}

		periodEndDate := period.EndDate.AddDate(0, 0, -1) // end date is exclusive
		qr := rampPlanQueryRequest(ctx, ag, ats, accounts, rp.Platform, &period.StartDate, &periodEndDate)

		billingData, err := s.cloudAnalytics.GetQueryResult(ctx, &qr, rp.Customer.ID, "")
		if err != nil || billingData.Error != nil {
			periodsSpends = append(periodsSpends, make(map[pkg.YearMonth]Spend))
			continue
		}

		spends, updatedMarketplaceTotal := actualMontlySpend(billingData, maxMarketplaceCover, currMarketplaceTotal, attributionNames)
		currMarketplaceTotal = updatedMarketplaceTotal

		periodsSpends = append(periodsSpends, spends)
	}

	return periodsSpends, nil
}

func getCommitmentValue(ctx context.Context, firestore connection.FirestoreFromContextFun, contractID string) (float64, error) {
	fsClient := firestore(ctx)

	contractSnap, err := fsClient.Collection("contracts").Doc(contractID).Get(ctx)
	if err != nil {
		return 0, err
	}

	contract := &pkg.Contract{}
	if err := contractSnap.DataTo(contract); err != nil {
		return 0, err
	}

	if len(contract.CommitmentPeriods) == 0 {
		return contract.EstimatedValue, nil
	}

	var commitment float64
	for _, commitmentPeriod := range contract.CommitmentPeriods {
		commitment += commitmentPeriod.Value
	}

	return commitment, nil
}

func getAccounts(ctx context.Context, firestore connection.FirestoreFromContextFun, contractID string, entity *firestore.DocumentRef) ([]string, error) {
	fsClient := firestore(ctx)
	contractDocRef := fsClient.Collection("contracts").Doc(contractID)
	assets := fsClient.Collection("assets").Where("contract", "==", contractDocRef).Where("entity", "==", entity).Documents(ctx)
	accounts := make(map[string]bool)

	for {
		doc, err := assets.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		var accountField string

		contractType, err := doc.DataAt("type")
		if err != nil {
			return nil, fmt.Errorf("failed to get type from %s", doc.Ref.ID)
		}

		if contractType == common.Assets.GoogleCloud || contractType == common.Assets.GoogleCloudProject {
			accountField = "properties.billingAccountId"
		} else if contractType == common.Assets.AmazonWebServices {
			accountField = "properties.accountId"
		}

		accountID, err := doc.DataAt(accountField)
		if err != nil {
			return nil, fmt.Errorf("failed to get accountId from %s", doc.Ref.ID)
		}

		accountIDStr, ok := accountID.(string)
		if !ok {
			return nil, fmt.Errorf("accountId is not a string in %s", doc.Ref.ID)
		}

		if _, ok := accounts[accountIDStr]; ok {
			continue
		}

		if accountIDStr == "" {
			continue
		}

		accounts[accountIDStr] = true
	}

	accountsSlice := make([]string, 0, len(accounts))
	for account := range accounts {
		accountsSlice = append(accountsSlice, account)
	}

	return accountsSlice, nil
}

func getAttributionGroupWithAttributions(ctx context.Context, firestore connection.FirestoreFromContextFun, ID string) (*attributiongroups.AttributionGroup, []*attribution.Attribution, error) {
	agDAL := attributionGroupsDAL.NewAttributionGroupsFirestoreWithClient(firestore)
	atDAL := attributionDAL.NewAttributionsFirestoreWithClient(firestore)

	ag, err := agDAL.Get(ctx, ID)
	if err != nil {
		return nil, nil, err
	}

	attributions := make([]*attribution.Attribution, 0)

	for _, attributionRef := range ag.Attributions {
		at, err := atDAL.GetAttribution(ctx, attributionRef.ID)
		if err != nil {
			return nil, nil, err
		}

		attributions = append(attributions, at)
	}

	return ag, attributions, nil
}

func rampPlanQueryRequest(
	ctx context.Context,
	ag *attributiongroups.AttributionGroup,
	ats []*attribution.Attribution,
	accounts []string,
	cloudProvider string,
	startDate *time.Time,
	endDate *time.Time,
) cloudanalytics.QueryRequest {
	attributions := make([]*domain.QueryRequestX, 0)
	for _, at := range ats {
		attributions = append(attributions, queryRequestXFromAttribution(at))
	}

	attributionGroups := domain.AttributionGroupQueryRequest{
		QueryRequestX: domain.QueryRequestX{
			ID:              ag.ID,
			Type:            metadata.MetadataFieldTypeAttributionGroup,
			Key:             ag.Name,
			IncludeInFilter: true,
		},
		Attributions: attributions,
	}

	from := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	to := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC)
	timeSettings := &cloudanalytics.QueryRequestTimeSettings{
		Interval: "month",
		From:     &from,
		To:       &to,
	}

	monthCol, err := domain.NewCol("month")
	if err != nil {
		panic(err)
	}

	yearCol, err := domain.NewCol("year")
	if err != nil {
		panic(err)
	}

	serviceCol, err := domain.NewCol("service_description")
	if err != nil {
		panic(err)
	}

	isMarketplaceCol, err := domain.NewCol("is_marketplace")
	if err != nil {
		panic(err)
	}

	attributionGroupCol := queryRequestXFromAttributionGroup(ag)

	costTypeCol, err := domain.NewCol("cost_type")
	if err != nil {
		panic(err)
	}

	cloudProviders := []string{cloudProvider}
	cloudProviderfilter := &domain.QueryRequestX{
		IncludeInFilter: true,
		ID:              "fixed:cloud_provider",
		Key:             "cloud_provider",
		Field:           "T.cloud_provider",
		Type:            "fixed",
		Position:        domain.QueryFieldPositionUnused,
		Values:          &cloudProviders,
	}

	return cloudanalytics.QueryRequest{
		Accounts:          accounts,
		CloudProviders:    &cloudProviders,
		Filters:           []*domain.QueryRequestX{cloudProviderfilter},
		AttributionGroups: []*domain.AttributionGroupQueryRequest{&attributionGroups},
		TimeSettings:      timeSettings,
		Cols:              []*domain.QueryRequestX{yearCol, monthCol, serviceCol, isMarketplaceCol, attributionGroupCol, costTypeCol},
		Origin:            domainOrigin.QueryOriginRampPlan,
		IncludeCredits:    true,
	}
}

func queryRequestXFromAttributionGroup(ag *attributiongroups.AttributionGroup) *domain.QueryRequestX {
	return &domain.QueryRequestX{
		ID:       ag.ID,
		Type:     metadata.MetadataFieldTypeAttributionGroup,
		Key:      ag.Name,
		Position: domain.QueryFieldPositionCol,
	}
}

func queryRequestXFromAttribution(a *attribution.Attribution) *domain.QueryRequestX {
	composite := make([]*domain.QueryRequestX, 0)
	for _, at := range a.Filters {
		composite = append(composite, queryRequestXFromFilters(at))
	}

	return &domain.QueryRequestX{
		ID:              a.ID,
		Type:            metadata.MetadataFieldTypeAttribution,
		Key:             a.Name,
		IncludeInFilter: true,
		Formula:         a.Formula,
		Composite:       composite,
	}
}

func queryRequestXFromFilters(filter report.BaseConfigFilter) *domain.QueryRequestX {
	return &domain.QueryRequestX{
		ID:        filter.ID,
		Type:      filter.Type,
		Key:       filter.Key,
		Inverse:   filter.Inverse,
		Regexp:    filter.Regexp,
		Values:    filter.Values,
		AllowNull: filter.AllowNull,
		Field:     filter.Field,
	}
}

func actualMontlySpend(billing cloudanalytics.QueryResult, marketplaceMaxCover float64, currMarketplaceTotal float64, atsNames map[string]bool) (map[pkg.YearMonth]Spend, float64) {
	var spends = make(map[pkg.YearMonth]Spend)

	for _, row := range billing.Rows {
		year, err := strconv.Atoi(row[0].(string))
		if err != nil {
			continue
		}

		month, err := strconv.Atoi(row[1].(string))
		if err != nil {
			continue
		}

		service, ok := row[2].(string)
		if !ok || service == "" {
			service = "N/A"
		}

		isMarketplace := row[3]

		attribution, attributionOk := row[4].(string)
		matchesAttribution := attributionOk && atsNames[attribution]

		costType, costTypeOk := row[5].(string)
		isCredit := costTypeOk && (costType == "Credit" || costType == "Credit Adjustment")

		// skip rows that don't match the attributions, or are credits
		if !matchesAttribution && !isCredit {
			continue
		}

		if isCredit {
			service = "Credits"
		}

		spend, ok := row[6].(float64)
		if !ok {
			continue
		}

		// marketplace is covered up to a certain percent of the total contract commitment
		if isMarketplace != nil && isMarketplace.(bool) {
			// Doit Cloud Cost Anomaly Detection is a marketplace service that is counted towards the commitment
			// but the spend in the tables is 0, so we add it here, only for 2023
			if service == "DoiT Cloud Cost Anomaly Detection" && year == 2023 {
				usage, ok := row[7].(float64)
				if !ok {
					continue
				}

				spend = usage * 0.01
			}

			// all marketplace services should be summed under "marketplace"
			service = "Marketplace"

			if currMarketplaceTotal+spend > marketplaceMaxCover {
				spend = marketplaceMaxCover - currMarketplaceTotal
			}

			currMarketplaceTotal += spend
		}

		key := pkg.YearMonth{Year: year, Month: month}

		mapItem, ok := spends[key]
		if !ok {
			mapItem = Spend{
				Services: make(map[string]float64),
			}
		}

		mapItem.Services[service] += spend
		mapItem.Total += spend
		spends[key] = mapItem
	}

	return spends, currMarketplaceTotal
}

func addActualSpendsToPeriods(periods []pkg.CommitmentPeriod, periodsSpends []map[pkg.YearMonth]Spend) ([]pkg.CommitmentPeriod, error) {
	if len(periods) != len(periodsSpends) {
		return nil, fmt.Errorf("periods and periodsSpends must be of the same length")
	}

	for i, period := range periods {
		curMonth := time.Date(period.StartDate.Year(), period.StartDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		period.Dates = make([]pkg.YearMonth, 0)
		period.Actuals = make([]float64, 0)
		breakdowns := make([]map[string]float64, 0)

		periodSpend := periodsSpends[i]

		for curMonth.Before(period.EndDate) {
			period.Dates = append(period.Dates, pkg.YearMonth{Year: curMonth.Year(), Month: int(curMonth.Month())})
			spend, ok := periodSpend[pkg.YearMonth{Year: curMonth.Year(), Month: int(curMonth.Month())}]
			curMonth = curMonth.AddDate(0, 1, 0)

			// if there is no spend for the month, add 0 to the actuals and an empty breakdown
			if !ok {
				period.Actuals = append(period.Actuals, 0)
				breakdowns = append(breakdowns, make(map[string]float64))

				continue
			}

			period.Actuals = append(period.Actuals, spend.Total)
			breakdowns = append(breakdowns, spend.Services)
		}

		// sum up the breakdowns for each month into a single breakdown for the period
		periodBreakdown := make(map[string]float64)

		for _, breakdown := range breakdowns {
			for service, spend := range breakdown {
				if _, ok := periodBreakdown[service]; !ok {
					periodBreakdown[service] = spend
				} else {
					periodBreakdown[service] += spend
				}
			}
		}

		periodBreakdownTop := topServicesBySpend(periodBreakdown, 10)
		period.PeriodActualsBreakdown = periodBreakdownTop
		periods[i] = period
	}

	return periods, nil
}

// topServicesBySpend returns the top n services by spend and "other" which is the sum of the rest of the services, and the credits if any
func topServicesBySpend(services map[string]float64, n int) map[string]float64 {
	topServices := make(map[string]float64)

	type kv struct {
		k string
		v float64
	}

	// move credits
	credits, ok := services["Credits"]
	if ok {
		topServices["Credits"] = credits

		delete(services, "Credits")
	}

	// convert the map to a slice
	arr := make([]kv, 0, len(services))
	for k, v := range services {
		arr = append(arr, kv{k, v})
	}

	// sort by value, descending
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].v > arr[j].v
	})

	// sum up the services after the n top ones into "other"
	other := 0.0
	for i := n; i < len(arr); i++ {
		other += arr[i].v
	}

	// return the top n services and "other"
	i := 0

	for {
		if i >= len(arr) || i >= n {
			break
		}

		topServices[arr[i].k] = arr[i].v
		i++
	}

	if other != 0 {
		topServices["Other"] = other
	}

	return topServices
}
