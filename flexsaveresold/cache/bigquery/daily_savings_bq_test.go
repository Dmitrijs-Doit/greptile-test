package bq

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/iterator"

	"github.com/doitintl/bigquery/iface"
	"github.com/doitintl/bigquery/mocks"
	pkg "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/types"
)

func Test_bigQueryService_getDailyPayerOnDemand(t *testing.T) {
	type fields struct {
		QueryHandler mocks.QueryHandler
	}

	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	now := time.Now()

	twoDaysAgo := now.AddDate(0, 0, -2)
	oneDayAgo := now.AddDate(0, 0, -1)

	params := DailyBQParams{
		Context:    context.Background(),
		CustomerID: "ABCDEF",
		Start:      sevenDaysAgo,
		End:        now,
	}

	inputChannel := make(chan map[string]float64)
	expectedDaily := make(map[string]float64)
	expectedDaily[oneDayAgo.Format(dateFormat)] = 2
	expectedDaily[twoDaysAgo.Format(dateFormat)] = 1

	inputErr := make(chan error)
	errDefault := errors.New("something went wrong")

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.QueryHandler.On("Read",
					params.Context,
					mock.MatchedBy(func(query *bigquery.Query) bool {
						expected := []bigquery.QueryParameter{
							{Name: "start", Value: sevenDaysAgo.Format(dateFormat)},
							{Name: "end", Value: now.Format(dateFormat)},
						}

						return assert.Equalf(t, expected, query.Parameters, "")
					})).
					Return(func() iface.RowIterator {
						rowIterator := &mocks.RowIterator{}
						rowIterator.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*pkg.ItemType)
							arg.Cost = 1
							arg.Date = twoDaysAgo
						}).Once()
						rowIterator.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*pkg.ItemType)
							arg.Cost = 2
							arg.Date = oneDayAgo
						}).Once()
						rowIterator.On("Next", mock.Anything).Return(iterator.Done).Once()
						return rowIterator
					}(), nil)
			},
		},
		{
			name: "failed to process query",
			on: func(f *fields) {
				f.QueryHandler.On("Read",
					params.Context,
					mock.MatchedBy(func(query *bigquery.Query) bool {
						expected := []bigquery.QueryParameter{
							{Name: "start", Value: sevenDaysAgo.Format(dateFormat)},
							{Name: "end", Value: now.Format(dateFormat)},
						}

						return assert.Equalf(t, expected, query.Parameters, "")
					})).
					Return(func() iface.RowIterator {
						rowIterator := &mocks.RowIterator{}
						rowIterator.On("Next", mock.Anything).Return(iterator.Done).Once()
						return rowIterator
					}(), errDefault)
			},
			wantErr: errDefault,
		},
		{
			name: "failed during values iterator",
			on: func(f *fields) {
				f.QueryHandler.On("Read",
					params.Context,
					mock.MatchedBy(func(query *bigquery.Query) bool {
						expected := []bigquery.QueryParameter{
							{Name: "start", Value: sevenDaysAgo.Format(dateFormat)},
							{Name: "end", Value: now.Format(dateFormat)},
						}

						return assert.Equalf(t, expected, query.Parameters, "")
					})).
					Return(func() iface.RowIterator {
						rowIterator := &mocks.RowIterator{}
						rowIterator.On("Next", mock.Anything).Return(errDefault).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*pkg.ItemType)
							arg.Cost = 1
							arg.Date = twoDaysAgo
						}).Once()
						rowIterator.On("Next", mock.Anything).Return(iterator.Done).Once()
						return rowIterator
					}(), nil)
			},
			wantErr: errDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			go func() {
				s := &BigQueryService{
					QueryHandler: &fields.QueryHandler,
				}

				s.getDailyPayerOnDemand(params, inputChannel, inputErr)
			}()

			select {
			case daily := <-inputChannel:
				assert.EqualValues(t, expectedDaily, daily)
			case err := <-inputErr:
				if tt.wantErr != nil {
					assert.ErrorContains(t, err, tt.wantErr.Error())
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

func Test_bigQueryService_getDailyPayerSavings(t *testing.T) {
	type fields struct {
		QueryHandler mocks.QueryHandler
	}

	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	now := time.Now()

	twoDaysAgo := now.AddDate(0, 0, -2)
	oneDayAgo := now.AddDate(0, 0, -1)

	params := DailyBQParams{
		Context:    context.Background(),
		CustomerID: "ABCDEF",
		Start:      sevenDaysAgo,
		End:        now,
	}

	inputChannel := make(chan map[string]float64)
	expectedDaily := make(map[string]float64)
	expectedDaily[oneDayAgo.Format(dateFormat)] = -7
	expectedDaily[twoDaysAgo.Format(dateFormat)] = -5

	inputErr := make(chan error)
	errDefault := errors.New("something went wrong")

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.QueryHandler.On("Read",
					params.Context,
					mock.MatchedBy(func(query *bigquery.Query) bool {
						expected := []bigquery.QueryParameter{
							{Name: "start", Value: params.Start.Format(dateFormat)},
							{Name: "end", Value: params.End.Format(dateFormat)},
						}

						return assert.Equalf(t, expected, query.Parameters, "")
					})).
					Return(func() iface.RowIterator {
						rowIterator := &mocks.RowIterator{}
						rowIterator.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*pkg.ItemType)
							arg.Cost = 5
							arg.Date = twoDaysAgo
						}).Once()
						rowIterator.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*pkg.ItemType)
							arg.Cost = 7
							arg.Date = oneDayAgo
						}).Once()
						rowIterator.On("Next", mock.Anything).Return(iterator.Done).Once()
						return rowIterator
					}(), nil)
			},
		},
		{
			name: "failed to process query",
			on: func(f *fields) {
				f.QueryHandler.On("Read",
					params.Context,
					mock.MatchedBy(func(query *bigquery.Query) bool {
						expected := []bigquery.QueryParameter{
							{Name: "start", Value: params.Start.Format(dateFormat)},
							{Name: "end", Value: params.End.Format(dateFormat)},
						}

						return assert.Equalf(t, expected, query.Parameters, "")
					})).
					Return(func() iface.RowIterator {
						rowIterator := &mocks.RowIterator{}
						return rowIterator
					}(), errDefault)
			},
			wantErr: errDefault,
		},
		{
			name: "failed during iterator",
			on: func(f *fields) {
				f.QueryHandler.On("Read",
					params.Context,
					mock.MatchedBy(func(query *bigquery.Query) bool {
						expected := []bigquery.QueryParameter{
							{Name: "start", Value: params.Start.Format(dateFormat)},
							{Name: "end", Value: params.End.Format(dateFormat)},
						}

						return assert.Equalf(t, expected, query.Parameters, "")
					})).
					Return(func() iface.RowIterator {
						rowIterator := &mocks.RowIterator{}
						rowIterator.On("Next", mock.Anything).Return(errDefault)
						return rowIterator
					}(), nil)
			},
			wantErr: errDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			go func() {
				s := &BigQueryService{
					QueryHandler: &fields.QueryHandler,
				}

				s.getDailyPayerSavings(params, inputChannel, inputErr)
			}()

			select {
			case daily := <-inputChannel:
				assert.EqualValues(t, expectedDaily, daily)
			case err := <-inputErr:
				if tt.wantErr != nil {
					assert.ErrorContains(t, err, tt.wantErr.Error())
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}
