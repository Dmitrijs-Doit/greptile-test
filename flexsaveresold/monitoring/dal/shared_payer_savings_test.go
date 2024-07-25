package dal

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/doitintl/bigquery/iface"
	"github.com/doitintl/bigquery/mocks"
	"github.com/doitintl/errors"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/monitoring/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func Test_sharedPayerSavings_DetectSharedPayerSavingsDiscrepancies(t *testing.T) {
	type fields struct {
		queryHandler   mocks.QueryHandler
		loggerProvider logger.Provider
	}

	type args struct {
		ctx         context.Context
		currentDate string
	}

	discrepancyResults := domain.SharedPayerSavingsDiscrepancies{
		{CustomerID: "customer123", LastMonthSavings: 500.00},
		{CustomerID: "customer456", LastMonthSavings: 300.00},
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		args    args
		on      func(*fields)
		want    domain.SharedPayerSavingsDiscrepancies
		wantErr bool
	}{
		{
			name: "successful query execution",
			args: args{
				ctx:         ctx,
				currentDate: "2024-05-08",
			},
			on: func(f *fields) {
				f.queryHandler.On("Read",
					ctx,
					mock.MatchedBy(func(query *bigquery.Query) bool {
						return strings.Contains(query.QueryConfig.Q, "Discrepancies") &&
							strings.Contains(query.QueryConfig.Q, sharedPayerFlexsaveBillingTableView)
					})).
					Return(func() iface.RowIterator {
						q := &mocks.RowIterator{}
						q.On("Next", mock.Anything).Return(func(dest interface{}) error {
							switch d := dest.(type) {
							case *domain.SharedPayerSavingsDiscrepancy:
								*d = domain.SharedPayerSavingsDiscrepancy{CustomerID: "customer123", LastMonthSavings: 500.00}
							default:
								return errors.New("incorrect type provided for scan")
							}
							return nil
						}).Once()
						q.On("Next", mock.Anything).Return(func(dest interface{}) error {
							switch d := dest.(type) {
							case *domain.SharedPayerSavingsDiscrepancy:
								*d = domain.SharedPayerSavingsDiscrepancy{CustomerID: "customer456", LastMonthSavings: 300.00}
							default:
								return errors.New("incorrect type provided for scan")
							}
							return nil
						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil)
			},
			wantErr: false,
			want:    discrepancyResults,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{}

			if tt.on != nil {
				tt.on(f)
			}

			client, err := bigquery.NewClient(ctx, "flextest", option.WithoutAuthentication(), option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			if err != nil {
				panic(err)
			}

			s := sharedPayerSavings{
				client:         client,
				queryHandler:   &f.queryHandler,
				loggerProvider: f.loggerProvider,
			}

			got, err := s.DetectSharedPayerSavingsDiscrepancies(tt.args.ctx, tt.args.currentDate)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectSharedPayerSavingsDiscrepancies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DetectSharedPayerSavingsDiscrepancies() got = %v, want %v", got, tt.want)
			}
		})
	}
}
