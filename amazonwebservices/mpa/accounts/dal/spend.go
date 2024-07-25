package dal

type Spend struct {
	Spend    float64 `bigquery:"spend"`
	CostType string  `bigquery:"cost_type"`
}

type Spends []Spend

func (ss Spends) Spend(costType string) (float64, bool) {
	for _, s := range ss {
		if s.CostType == costType {
			return s.Spend, true
		}
	}

	return 0, false
}
