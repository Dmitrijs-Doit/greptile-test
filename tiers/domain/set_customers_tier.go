package domain

import "github.com/doitintl/firestore/pkg"

// Set multiple customers tier
type SetCustomersTierRequest struct {
	CustomersTiers []CustomerTierMapping `json:"customersTiers"`
}

// Set single customer tiers
type SetCustomerTiersRequest struct {
	NavigatorTierID string `json:"navigatorTierId"`
	SolveTierID     string `json:"solveTierId"`
}

type CustomerTierMapping struct {
	CustomerID string              `json:"customerId"`
	Tier       string              `json:"tier"`
	TierType   pkg.PackageTierType `json:"tierType"`
}
