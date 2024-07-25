package service

import (
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/stretchr/testify/assert"
)

func Test_mapBudgetToNotification(t *testing.T) {

	type args struct {
		b         *budget.Budget
		nt        budget.BudgetNotificationType
		alertDate *time.Time
	}

	customerRef := &firestore.DocumentRef{
		ID: "test",
	}

	now := time.Now()
	forcastTime := now.AddDate(0, 0, 3)

	tests := []struct {
		name string
		args args
		want *budget.BudgetNotification
	}{
		{
			name: "Test mapBudgetToNotification",
			args: args{
				alertDate: &now,
				b: &budget.Budget{
					ID:       "1",
					Customer: customerRef,
					Name:     "test",
					Config: &budget.BudgetConfig{
						Amount:   100,
						Currency: "USD",
						Alerts: [3]budget.BudgetAlert{
							{
								Percentage: 90,
							},
						},
					},
					Utilization: budget.BudgetUtilization{
						Current:                   91,
						ForecastedTotalAmountDate: &forcastTime,
					},
					Recipients: []string{"email@example.com"},
				},
				nt: budget.BudgetNotificationTypeThreshold,
			},
			want: &budget.BudgetNotification{
				Name:              "test",
				Type:              budget.BudgetNotificationTypeThreshold,
				BudgetID:          "1",
				Customer:          customerRef,
				AlertDate:         now,
				AlertAmount:       "90",
				AlertPercentage:   90,
				CurrencySymbol:    "$",
				CurrentAmount:     "91",
				CurrentPercentage: 91,
				ForcastedDate:     &forcastTime,
				ExpireBy:          now.AddDate(0, 0, 31),
				Recipients:        []string{"email@example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if got := mapBudgetToNotification(tt.args.b, tt.args.nt, *tt.args.alertDate); !assert.Equal(t, tt.want, got) {
				t.Errorf("mapBudgetToNotification() = %v, want %v", got, tt.want)
			}
		})
	}
}
