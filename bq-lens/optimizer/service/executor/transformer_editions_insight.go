package executor

import (
	_ "embed"
	"sort"

	"github.com/go-playground/validator/v10"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	pricebookDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/domain"
	insightsSDK "github.com/doitintl/insights/sdk"
)

//go:embed descriptions/switch_to_editions.mdx
var descriptionInsightSwitchToEditions string

const (
	msInHour   = 3.6e+6
	bytesInTib = 1e+12

	// Recommend switching if they can save at least $100 a year.
	minimumSavingThreshold float64 = 100.0 / 365.0

	// Based on the 90-day lookback window.
	dayScalingFactor float64 = 1 / 90.0

	switchToEditionsTitle            = "BigQuery Lens On Demand Savings"
	switchToEditionsShortDescription = "Switch to Editions"
	switchToEditionsInsightKey       = "switch-to-editions"

	supportCategory = "gcp/bigquery"
)

var (
	switchToEditionsDimensions = []string{"Project", "Edition"}

	otherTags = []string{"FinOps"}
)

func TransformSwitchToEditions(
	customerID string,
	projectsAssignedToReservations map[string]struct{},
	aggregatedJobsStatistics []bqmodels.AggregatedJobStatistic,
	editionsPricebooks pricebookDomain.PriceBooksByEdition,
	onDemandPricebook map[string]float64,
) (insightsSDK.InsightResponse, error) {
	results := &insightsSDK.InsightResults{
		IsRelevant: false,
	}

	response := insightsSDK.InsightResponse{
		Key:                    switchToEditionsInsightKey,
		CustomerID:             customerID,
		Title:                  switchToEditionsTitle,
		ShortDescription:       switchToEditionsShortDescription,
		DetailedDescriptionMdx: descriptionInsightSwitchToEditions,
		Status:                 insightsSDK.StatusSuccess,
		CloudTags:              []insightsSDK.CloudTag{insightsSDK.CloudTagGCP},
		OtherTags:              otherTags,
		IsInternal:             false,
		SupportCategory:        supportCategory,
		Results:                results,
	}

	estimatedCosts := onDemandSavingsSimulation(aggregatedJobsStatistics, editionsPricebooks, onDemandPricebook)
	projectsCosts := aggregateProjectCosts(estimatedCosts)
	filteredCosts := filterProjectCosts(projectsCosts, projectsAssignedToReservations)

	// Early return if there's no projects with pure on-demand workloads.
	if len(filteredCosts) == 0 {
		return response, nil
	}

	normalisedProjectCosts := normaliseProjectCosts(filteredCosts, dayScalingFactor)
	potentialProjectSavings, isOptimized := findPotentialSavings(normalisedProjectCosts)

	potentialDailySavings := &insightsSDK.PotentialDailySavings{
		IsOptimized: isOptimized,
	}

	insightResults := &insightsSDK.InsightResults{
		IsRelevant:            true,
		PotentialDailySavings: potentialDailySavings,
	}

	response.Results = insightResults

	if isOptimized {
		return response, nil
	}

	breakdownData := make([]insightsSDK.BreakdownData[float64], 0)

	for projectID, editionSavings := range potentialProjectSavings {
		saving := findOptimalSaving(editionSavings)

		dimensionValues := []string{projectID, saving.Edition.String()}
		breakdownData = append(breakdownData, insightsSDK.BreakdownData[float64]{
			Value:           saving.DailySavings,
			BaseValue:       saving.BaseCost,
			DimensionValues: dimensionValues,
		})
	}

	if len(breakdownData) == 0 {
		response.Results.PotentialDailySavings.IsOptimized = true
		return response, nil
	}

	sort.Slice(breakdownData, func(i, j int) bool { return breakdownData[i].Value > breakdownData[j].Value })

	response.Results.PotentialDailySavings.Breakdown = &insightsSDK.Breakdown[float64]{
		Dimensions: switchToEditionsDimensions,
		Data:       breakdownData,
	}

	return response, response.Validate(validator.New(validator.WithRequiredStructEnabled()))
}

func findOptimalSaving(editionSavings []domain.EditionSavings) domain.EditionSavings {
	// Find a good balance between commitment and slotHour cost.
	preferred := []domain.EditionModel{
		domain.EditionModelStandard,
		domain.EditionModelEnterprise1y,
		domain.EditionModelEnterprise,
		domain.EditionModelEnterprise3y,
	}

	savings := make(map[domain.EditionModel]domain.EditionSavings)

	for _, editionSaving := range editionSavings {
		savings[editionSaving.Edition] = editionSaving
	}

	for _, p := range preferred {
		if saving, ok := savings[p]; ok {
			return saving
		}
	}

	// return the highest saving found if we didn't find any of the preferred ones.
	return editionSavings[0]
}

func findPotentialSavings(normalisedProjectCosts domain.ProjectsCosts) (domain.ProjectsSavings, bool) {
	isOptimized := true

	potentialSavings := make(domain.ProjectsSavings)

	for projectID, costs := range normalisedProjectCosts {
		onDemandCost := costs[domain.EditionModelOnDemand]

		editionSavings := []domain.EditionSavings{}

		for edition, cost := range costs {
			if edition == domain.EditionModelOnDemand {
				continue
			}

			// We only recommend switching if the customer can save at least $100 a year.
			if cost < onDemandCost && (onDemandCost-cost) >= minimumSavingThreshold {
				isOptimized = false

				editionSavings = append(editionSavings, domain.EditionSavings{
					Edition:      edition,
					DailySavings: onDemandCost - cost,
					BaseCost:     onDemandCost,
				})
			}
		}

		sort.Slice(editionSavings, func(i, j int) bool {
			return editionSavings[i].DailySavings > editionSavings[j].DailySavings
		})

		if len(editionSavings) > 0 {
			potentialSavings[projectID] = editionSavings
		}
	}

	return potentialSavings, isOptimized
}

func normaliseProjectCosts(filteredProjectsCosts domain.ProjectsCosts, scale float64) domain.ProjectsCosts {
	normalisedCosts := make(domain.ProjectsCosts)

	for projectID, costs := range filteredProjectsCosts {
		if _, ok := normalisedCosts[projectID]; !ok {
			normalisedCosts[projectID] = make(map[domain.EditionModel]float64)
		}

		for edition, cost := range costs {
			normalisedCosts[projectID][edition] = cost * scale
		}
	}

	return normalisedCosts
}

func filterProjectCosts(projectsCosts domain.ProjectsCosts, projectsAssignedToReservations map[string]struct{}) domain.ProjectsCosts {
	filteredProjectsCosts := make(domain.ProjectsCosts)

	for projectID, costs := range projectsCosts {
		if _, ok := projectsAssignedToReservations[projectID]; ok {
			continue
		}

		filteredProjectsCosts[projectID] = costs
	}

	return filteredProjectsCosts
}

func aggregateProjectCosts(estimatedCosts domain.EstimatedCosts) domain.ProjectsCosts {
	projectsCosts := make(domain.ProjectsCosts)

	for _, projectsCost := range estimatedCosts {
		for projectID, costs := range projectsCost {
			if _, ok := projectsCosts[projectID]; !ok {
				projectsCosts[projectID] = make(map[domain.EditionModel]float64)
			}

			for edition, cost := range costs {
				projectsCosts[projectID][edition] += cost
			}
		}
	}

	return projectsCosts
}

func onDemandSavingsSimulation(
	aggregatedJobsStatistics []bqmodels.AggregatedJobStatistic,
	editionsPricebooks pricebookDomain.PriceBooksByEdition,
	onDemandPricebook map[string]float64,
) domain.EstimatedCosts {
	estimatedCosts := make(domain.EstimatedCosts)

	aggregatedCounters := aggregatedStatisticsToPricingUnits(aggregatedJobsStatistics)

	for region, projectStats := range aggregatedCounters {
		ondemandCost := onDemandPricebook[region]
		editionsCosts := editionsCostsForRegion(editionsPricebooks, region)

		estimatedCosts[region] = make(domain.ProjectsCosts)

		for projectID, stats := range projectStats {
			if _, ok := estimatedCosts[region][projectID]; !ok {
				estimatedCosts[region][projectID] = make(map[domain.EditionModel]float64)
			}

			estimatedCosts[region][projectID][domain.EditionModelOnDemand] = stats.ScanTB * ondemandCost

			for edition, editionCost := range editionsCosts {
				estimatedCosts[region][projectID][edition] = stats.SlotHours * editionCost
			}
		}
	}

	return estimatedCosts
}

func aggregatedStatisticsToPricingUnits(aggregatedJobsStatistics []bqmodels.AggregatedJobStatistic) domain.AggregatedOnDemandCounters {
	aggregatedOnDemandCounters := make(domain.AggregatedOnDemandCounters)

	for _, jobStat := range aggregatedJobsStatistics {
		location := jobStat.Location
		projectID := jobStat.ProjectID
		slotHours := float64(jobStat.TotalSlotsMS) / msInHour
		scanTB := float64(jobStat.TotalBilledBytes) / bytesInTib

		if _, ok := aggregatedOnDemandCounters[location]; !ok {
			aggregatedOnDemandCounters[domain.Region(location)] = make(map[domain.ProjectID]domain.AggregatedJobStatisticCounters)
		}

		aggregatedOnDemandCounters[domain.Region(location)][domain.ProjectID(projectID)] =
			domain.AggregatedJobStatisticCounters{
				SlotHours: slotHours,
				ScanTB:    scanTB,
			}
	}

	return aggregatedOnDemandCounters
}

func editionsCostsForRegion(pricebooks pricebookDomain.PriceBooksByEdition, region string) map[domain.EditionModel]float64 {
	standard := *pricebooks[pricebookDomain.Standard]
	enterprise := *pricebooks[pricebookDomain.Enterprise]

	costsForRegion := map[domain.EditionModel]float64{
		domain.EditionModelStandard:     standard[string(pricebookDomain.OnDemand)][region],
		domain.EditionModelEnterprise:   enterprise[string(pricebookDomain.OnDemand)][region],
		domain.EditionModelEnterprise1y: enterprise[string(pricebookDomain.Commit1Yr)][region],
		domain.EditionModelEnterprise3y: enterprise[string(pricebookDomain.Commit3Yr)][region],
	}

	return costsForRegion
}
