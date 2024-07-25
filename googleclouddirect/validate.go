package googleclouddirect

import (
	"regexp"
)

func MatchBillingAccount(v string) (bool, error) {
	// has to match the schema in the frontend
	// https://github.com/doitintl/cmp-main/blob/394563419b86cad4a19536978a1e9263453f52ce/client/src/Pages/Assets/Forms/utils.ts
	return regexp.MatchString("^(?:[A-F0-9]{6}-){2}[A-F0-9]{6}$", v)
}

func MatchProject(v string) (bool, error) {
	// has to match the schema in the frontend
	// https://github.com/doitintl/cmp-main/blob/394563419b86cad4a19536978a1e9263453f52ce/client/src/Pages/Assets/Forms/utils.ts
	return regexp.MatchString("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", v)
}

func MatchDataset(v string) (bool, error) {
	// has to match the schema in the frontend
	// https://github.com/doitintl/cmp-main/blob/394563419b86cad4a19536978a1e9263453f52ce/client/src/Pages/Assets/Forms/utils.ts
	return regexp.MatchString("^[a-zA-Z0-9_]{1,1000}[a-zA-Z0-9_]{0,24}$", v)
}
