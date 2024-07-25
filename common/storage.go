package common

import "fmt"

func GetStaticEphemeralBucket() string {
	if Production {
		return "cmp-static-ephemeral"
	}

	return fmt.Sprintf("%s-static-ephemeral", ProjectID)
}
