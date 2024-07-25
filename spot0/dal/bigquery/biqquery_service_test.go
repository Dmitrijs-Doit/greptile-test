package bq

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBigQueryService(t *testing.T) {
	ctx := context.Background()
	bqs, err := NewBigQueryService(ctx)
	assert.NoError(t, err)

	t.Run("query should return without start date and and date", func(t *testing.T) {
		query := bqs.buildAggregateDailySavingsQuery("", "", "")
		assert.NotContains(t, query, "set start_date")
		assert.NotContains(t, query, "set end_date")
		assert.NotContains(t, query, "AND project_id")
	})

	t.Run("query should return with start date and and date", func(t *testing.T) {
		query := bqs.buildAggregateDailySavingsQuery("2022-09-01", "2022-09-03", "")
		assert.Contains(t, query, `set start_date = "2022-09-01"`)
		assert.Contains(t, query, `set end_date = "2022-09-03"`)
	})

	t.Run("query should return with accountId", func(t *testing.T) {
		query := bqs.buildAggregateDailySavingsQuery("", "", "12345")
		assert.Contains(t, query, `AND project_id = "12345"`)
	})

	t.Run("query should return without billing_year and billing_month", func(t *testing.T) {
		query := bqs.buildMonthlyUsageQuery("", "", "")
		assert.NotContains(t, query, `set billing_year_filter`)
		assert.NotContains(t, query, `set billing_month_filter`)
		assert.NotContains(t, query, `AND account = "`)
	})

	t.Run("query should return with billing_year and billing_month", func(t *testing.T) {
		query := bqs.buildMonthlyUsageQuery("2022", "09", "12345")
		assert.Contains(t, query, `set billing_year_filter = "2022"`)
		assert.Contains(t, query, `set billing_month_filter = "09"`)
		assert.Contains(t, query, `AND account = "12345"`)
	})

	t.Run("nonBillingTagsAsgQuery query should return 4 fields", func(t *testing.T) {
		query := bqs.buildNonBillingTagsAsgQuery()
		assert.Contains(t, query, "a.primaryDomain AS primary_domain, a.account, a.region, a.name AS asg_name")
	})

	t.Run("buildNonBillingTagsDomainQuery", func(t *testing.T) {
		query := bqs.buildNonBillingTagsDomainQuery()
		assert.IsType(t, query, "string")
	})
}
