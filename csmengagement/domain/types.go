package domain

import customerDomain "github.com/doitintl/hello/scheduled-tasks/customer/domain"

type NoAttributionEmailParams struct {
	RecipientEmail string
	CustomerName   string
	RecipientName  string
	AccountTeam    []customerDomain.AccountManagerListItem
}
