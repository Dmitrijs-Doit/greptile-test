package recommendations

import "time"

type RecommendationForDedicatedPayerResponse struct {
	HourlyCommitmentToPurchase *float64
	EstimatedSavingsAmount     *float64
	AverageSavingsRate         *float64
	TotalWaste                 *float64
	TimeframeHours             int
}

type payerRecommendationV2Input struct {
	StartDateTime string `json:"startDateTime"`
	EndDateTime   string `json:"endDateTime"`
}

type artifactIndex struct {
	UnitIndex       float64 `json:"unit_index"`
	DoiTRevenue     float64 `json:"doit_revenue"`
	CustomerSavings float64 `json:"customer_savings"`
	SavingsRate     float64 `json:"savings_rate"`
	Waste           float64
	OnDemandCost    float64 `json:"ondemand_cost"`
}

type payerRecommendationV2Output struct {
	PayerID                string          `json:"payerID" firestore:"payer_id"`
	PayerType              string          `json:"payerType" firestore:"payer_type"`
	StartDateTime          string          `json:"startDateTime" firestore:"start_date_time"`
	UnionThresholdDateTime string          `json:"unionThresholdDateTime" firestore:"union_threshold_date_time"`
	EndDateTime            string          `json:"endDateTime" firestore:"end_date_time"`
	Recommendation         float64         `json:"recommendation" firestore:"recommendation"`
	ArtifactOptimumDetails []artifactIndex `json:"artifactOptimumDetails" firestore:"artifact_optimum_details"`
	MinCoverage            float64         `json:"minCoverage" firestore:"min_coverage"`
	Coverage               float64         `json:"coverage" firestore:"coverage"`
	StableOnDemandCost     float64         `json:"stableOnDemandCost" firestore:"stable_ondemand_cost"`
}

type sagemakerPayload struct {
	StartDateTime time.Time `json:"startDateTime"`
	EndDateTime   time.Time `json:"endDateTime"`
}

type ArtifactData struct {
	UnitID          float64 `json:"unitID"`
	DoitRevenue     float64 `json:"doitRevenue"`
	CustomerSavings float64 `json:"customerSavings"`
	Waste           float64 `json:"waste"`
	SavingsRate     float64 `json:"savingsRate"`
	OnDemandCost    float64 `json:"onDemandCost"`
}

type SageMakerRecommendation struct {
	PayerID          string         `json:"payerID"`
	StartDateTime    time.Time      `json:"startDateTime"`
	EndDateTime      time.Time      `json:"endDateTime"`
	Recommendation   float64        `json:"recommendation"`
	OptimalArtifacts []ArtifactData `json:"optimalArtifacts"`
	MinCoverage      float64        `json:"minCoverage"`
	Coverage         float64        `json:"coverage"`
}

type SageMakerRecommendationPayerResponse struct {
	HourlyCommitmentToPurchase float64
	EstimatedSavingsAmount     float64
	TimeframeHours             int
}
