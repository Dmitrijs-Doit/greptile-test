package payermanager

import "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"

func getPointerOrDefault[T any](existing T, entry *T) T {
	if entry == nil {
		return existing
	}

	return *entry
}

func mergeDiscounts(existing []types.Discount, entry []types.Discount) []types.Discount {
	if len(entry) > 0 {
		merged := append([]types.Discount(nil), existing...)
		return append(merged, entry...)
	}

	return append([]types.Discount(nil), existing...)
}
