package domain

import "strings"

func ExtractResourceID(resourceName string) string {
	sl := strings.Split(resourceName, "/")

	return sl[len(sl)-1]
}
