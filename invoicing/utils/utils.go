package utils

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

const (
	BillingFlexsaveCostType           = "Flexsave"
	InvoiceFlexsaveSavingsDescription = "Flexsave Savings"
	InvoiceFlexsaveSavingsDetails     = "DoiT Flexsave Savings"
	LookerType                        = "looker"
	NavigatorType                     = "navigator"
	SolveType                         = "solve"
	SolveAcceleratorType              = "solve-accelerator"
	// BizOps(external) invoice types, only used in export spreadsheet
	NavigatorInvoiceType        = "doit-navigator"
	SolveInvoiceType            = "doit-solve"
	SolveAcceleratorInvoiceType = "doit-solve-accelerator"
)

func GetDiscount(ctx context.Context, contractRef *firestore.DocumentRef, cache map[string][2]float64) ([2]float64, string) {
	l := logger.FromContext(ctx)

	if contractRef == nil {
		return [2]float64{}, ""
	}

	if cache != nil {
		if v, prs := cache[contractRef.ID]; prs {
			return v, contractRef.ID
		}
	}

	docSnap, err := contractRef.Get(ctx)
	if err != nil {
		l.Errorf("failed to get contract %s with error: %s", contractRef.ID, err)
		return [2]float64{}, ""
	}

	var contract common.Contract

	if err := docSnap.DataTo(&contract); err != nil {
		l.Errorf("failed to populate contract %s to struct with error: %s", contractRef.ID, err)
		return [2]float64{}, ""
	}

	var result [2]float64
	result[0] = contract.Discount

	if contract.Type == common.Assets.GSuite {
		if v, prs := contract.Properties["specialDiscount"]; prs {
			switch t := v.(type) {
			case int64:
				result[1] = float64(t)
			case float64:
				result[1] = float64(t)
			default:
			}
		}
	}

	if cache != nil {
		cache[contractRef.ID] = result
	}

	return result, contractRef.ID
}

// ToProportion changes a discount value of "25" to decimal value 0.75
func ToProportion(d float64) float64 {
	return 1 - d*0.01
}

func GetQuantityAndValue(quantity int64, value float64) (int64, float64) {
	if value >= 0 {
		return quantity, value
	}

	return -quantity, -value
}

func IsIssuableAssetType(assetType string) bool {
	return slice.Contains([]string{common.Assets.AmazonWebServices, NavigatorType, SolveType, SolveAcceleratorType}, assetType)
}
