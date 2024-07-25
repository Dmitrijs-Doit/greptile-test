package utils

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/rippling/pkg"
	ripplingPkg "github.com/doitintl/rippling/pkg"
)

func GetRipplingLogger(ctx context.Context, loggerProvider logger.Provider, flow string) logger.ILogger {
	l := loggerProvider(ctx)
	l.SetLabels(map[string]string{
		"service": "rippling",
		"flow":    flow,
	})

	return l
}

func GetFullName(amRippling *ripplingPkg.Employee) string {
	firstName := amRippling.PreferredFirstName
	lastName := amRippling.PreferredLastName

	if firstName == "" {
		firstName = amRippling.FirstName
	}

	if lastName == "" {
		lastName = amRippling.LastName
	}

	return firstName + " " + lastName
}

func ToMap(employees []*ripplingPkg.Employee) pkg.AccountManagersMap {
	accountManagersMap := pkg.AccountManagersMap{}
	for _, employee := range employees {
		accountManagersMap[employee.ID] = employee
	}

	return accountManagersMap
}
