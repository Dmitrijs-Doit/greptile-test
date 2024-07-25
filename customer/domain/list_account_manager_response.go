package domain

import "github.com/doitintl/hello/scheduled-tasks/common"

type AccountManagerListItem struct {
	ID           string                    `json:"id" firestore:"ref"`
	Email        string                    `json:"email"`
	Name         string                    `json:"name"`
	Role         common.AccountManagerRole `json:"role"`
	CalendlyLink string                    `json:"calendlyLink" firestore:"calendlyLink"`
}

type AccountManagerListAPI struct {
	AccountManagers []AccountManagerListItem `json:"accountManagers"`
}
