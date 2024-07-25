// Code generated by mockery v2.42.3. DO NOT EDIT.

package mocks

import (
	context "context"

	bigquery "cloud.google.com/go/bigquery"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"

	domain "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// Bigquery is an autogenerated mock type for the Bigquery type
type Bigquery struct {
	mock.Mock
}

// GenerateStorageRecommendation provides a mock function with given fields: ctx, customerID, bq, discount, replacements, now, hasTableDiscovery
func (_m *Bigquery) GenerateStorageRecommendation(ctx context.Context, customerID string, bq *bigquery.Client, discount float64, replacements domain.Replacements, now time.Time, hasTableDiscovery bool) (domain.PeriodTotalPrice, dal.RecommendationSummary, error) {
	ret := _m.Called(ctx, customerID, bq, discount, replacements, now, hasTableDiscovery)

	if len(ret) == 0 {
		panic("no return value specified for GenerateStorageRecommendation")
	}

	var r0 domain.PeriodTotalPrice
	var r1 dal.RecommendationSummary
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *bigquery.Client, float64, domain.Replacements, time.Time, bool) (domain.PeriodTotalPrice, dal.RecommendationSummary, error)); ok {
		return rf(ctx, customerID, bq, discount, replacements, now, hasTableDiscovery)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *bigquery.Client, float64, domain.Replacements, time.Time, bool) domain.PeriodTotalPrice); ok {
		r0 = rf(ctx, customerID, bq, discount, replacements, now, hasTableDiscovery)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(domain.PeriodTotalPrice)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *bigquery.Client, float64, domain.Replacements, time.Time, bool) dal.RecommendationSummary); ok {
		r1 = rf(ctx, customerID, bq, discount, replacements, now, hasTableDiscovery)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(dal.RecommendationSummary)
		}
	}

	if rf, ok := ret.Get(2).(func(context.Context, string, *bigquery.Client, float64, domain.Replacements, time.Time, bool) error); ok {
		r2 = rf(ctx, customerID, bq, discount, replacements, now, hasTableDiscovery)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetAggregatedJobStatistics provides a mock function with given fields: ctx, bq, projectID, location
func (_m *Bigquery) GetAggregatedJobStatistics(ctx context.Context, bq *bigquery.Client, projectID string, location string) ([]bqmodels.AggregatedJobStatistic, error) {
	ret := _m.Called(ctx, bq, projectID, location)

	if len(ret) == 0 {
		panic("no return value specified for GetAggregatedJobStatistics")
	}

	var r0 []bqmodels.AggregatedJobStatistic
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, string, string) ([]bqmodels.AggregatedJobStatistic, error)); ok {
		return rf(ctx, bq, projectID, location)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, string, string) []bqmodels.AggregatedJobStatistic); ok {
		r0 = rf(ctx, bq, projectID, location)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]bqmodels.AggregatedJobStatistic)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *bigquery.Client, string, string) error); ok {
		r1 = rf(ctx, bq, projectID, location)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBillingProjectsWithEditions provides a mock function with given fields: ctx, bq
func (_m *Bigquery) GetBillingProjectsWithEditions(ctx context.Context, bq *bigquery.Client) (map[string][]domain.BillingProjectWithReservation, error) {
	ret := _m.Called(ctx, bq)

	if len(ret) == 0 {
		panic("no return value specified for GetBillingProjectsWithEditions")
	}

	var r0 map[string][]domain.BillingProjectWithReservation
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client) (map[string][]domain.BillingProjectWithReservation, error)); ok {
		return rf(ctx, bq)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client) map[string][]domain.BillingProjectWithReservation); ok {
		r0 = rf(ctx, bq)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string][]domain.BillingProjectWithReservation)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *bigquery.Client) error); ok {
		r1 = rf(ctx, bq)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBillingProjectsWithEditionsSingleCustomer provides a mock function with given fields: ctx, bq, customerID
func (_m *Bigquery) GetBillingProjectsWithEditionsSingleCustomer(ctx context.Context, bq *bigquery.Client, customerID string) ([]domain.BillingProjectWithReservation, error) {
	ret := _m.Called(ctx, bq, customerID)

	if len(ret) == 0 {
		panic("no return value specified for GetBillingProjectsWithEditionsSingleCustomer")
	}

	var r0 []domain.BillingProjectWithReservation
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, string) ([]domain.BillingProjectWithReservation, error)); ok {
		return rf(ctx, bq, customerID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, string) []domain.BillingProjectWithReservation); ok {
		r0 = rf(ctx, bq, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]domain.BillingProjectWithReservation)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *bigquery.Client, string) error); ok {
		r1 = rf(ctx, bq, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerDiscounts provides a mock function with given fields: ctx, bq
func (_m *Bigquery) GetCustomerDiscounts(ctx context.Context, bq *bigquery.Client) (map[string]float64, error) {
	ret := _m.Called(ctx, bq)

	if len(ret) == 0 {
		panic("no return value specified for GetCustomerDiscounts")
	}

	var r0 map[string]float64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client) (map[string]float64, error)); ok {
		return rf(ctx, bq)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client) map[string]float64); ok {
		r0 = rf(ctx, bq)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]float64)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *bigquery.Client) error); ok {
		r1 = rf(ctx, bq)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDatasetLocationAndProjectID provides a mock function with given fields: ctx, bq, datasetID
func (_m *Bigquery) GetDatasetLocationAndProjectID(ctx context.Context, bq *bigquery.Client, datasetID string) (string, string, error) {
	ret := _m.Called(ctx, bq, datasetID)

	if len(ret) == 0 {
		panic("no return value specified for GetDatasetLocationAndProjectID")
	}

	var r0 string
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, string) (string, string, error)); ok {
		return rf(ctx, bq, datasetID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, string) string); ok {
		r0 = rf(ctx, bq, datasetID)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *bigquery.Client, string) string); ok {
		r1 = rf(ctx, bq, datasetID)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(context.Context, *bigquery.Client, string) error); ok {
		r2 = rf(ctx, bq, datasetID)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetMinAndMaxDates provides a mock function with given fields: ctx, bq, projectID, location
func (_m *Bigquery) GetMinAndMaxDates(ctx context.Context, bq *bigquery.Client, projectID string, location string) (*bqmodels.CheckCompleteDaysResult, error) {
	ret := _m.Called(ctx, bq, projectID, location)

	if len(ret) == 0 {
		panic("no return value specified for GetMinAndMaxDates")
	}

	var r0 *bqmodels.CheckCompleteDaysResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, string, string) (*bqmodels.CheckCompleteDaysResult, error)); ok {
		return rf(ctx, bq, projectID, location)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, string, string) *bqmodels.CheckCompleteDaysResult); ok {
		r0 = rf(ctx, bq, projectID, location)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bqmodels.CheckCompleteDaysResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *bigquery.Client, string, string) error); ok {
		r1 = rf(ctx, bq, projectID, location)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTableDiscoveryMetadata provides a mock function with given fields: ctx, bq
func (_m *Bigquery) GetTableDiscoveryMetadata(ctx context.Context, bq *bigquery.Client) (*bigquery.TableMetadata, error) {
	ret := _m.Called(ctx, bq)

	if len(ret) == 0 {
		panic("no return value specified for GetTableDiscoveryMetadata")
	}

	var r0 *bigquery.TableMetadata
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client) (*bigquery.TableMetadata, error)); ok {
		return rf(ctx, bq)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client) *bigquery.TableMetadata); ok {
		r0 = rf(ctx, bq)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bigquery.TableMetadata)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *bigquery.Client) error); ok {
		r1 = rf(ctx, bq)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewBigquery creates a new instance of Bigquery. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBigquery(t interface {
	mock.TestingT
	Cleanup(func())
}) *Bigquery {
	mock := &Bigquery{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}