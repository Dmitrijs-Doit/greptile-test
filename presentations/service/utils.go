package service

func doesSliceHaveItem[T any](slice []T, f func(T) bool) bool {
	for _, s := range slice {
		if f(s) {
			return true
		}
	}

	return false
}
