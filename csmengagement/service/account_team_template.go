package service

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/customer/domain"
)

func GetAccountTeamHTMLTemplate(accountTeam []domain.AccountManagerListItem) (string, string) {
	var (
		li           string
		calendlyLink string
	)

	for _, accountManager := range accountTeam {
		switch accountManager.Role {
		case common.AccountManagerRoleFSR:
			li += getLiItem(accountManager.Name, "Account Manager", accountManager.Email)
		case common.AccountManagerRoleSAM:
			li += getLiItem(accountManager.Name, "Strategic Account Manager", accountManager.Email)
		case common.AccountManagerRoleTAM:
			li += getLiItem(accountManager.Name, "Technical Account Manager", accountManager.Email)
		case common.AccountManagerRoleCSM:
			li += getLiItem(accountManager.Name, "Customer Success Manager", accountManager.Email)
			calendlyLink = accountManager.CalendlyLink
		}
	}

	return fmt.Sprintf("<UL>%s</UL>", li), calendlyLink
}
