package dal

import "time"

type ListBudgetsArgs struct {
	CustomerID      string
	Email           string
	Filter          *BudgetListFilter
	MinCreationTime *time.Time
	MaxCreationTime *time.Time
	MaxResults      int // default: 50
	PageToken       string
	IsDoitEmployee  bool
}

type BudgetListFilter struct {
	Owners       []string
	TimeModified *time.Time
	OrderBy      string // default: "lastModified"
}
