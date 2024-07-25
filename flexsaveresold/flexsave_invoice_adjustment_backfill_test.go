package flexsaveresold

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/bigquery/mocks"
	"github.com/doitintl/errors"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

func TestService_deleteExistingRows(t *testing.T) {
	type fields struct {
		queryHandler mocks.QueryHandler
		jobHandler   mocks.JobHandler
	}

	var (
		ctx          = &gin.Context{}
		customerID   = "abcdefg"
		job          = bigquery.Job{}
		status       = bigquery.JobStatus{}
		firstOfMonth = time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC)
		lastOfMonth  = firstOfMonth.AddDate(0, 6, -1)

		someErr = errors.New("something went wrong")
		runErr  = errors.Wrapf(someErr, "Run() customer %s", customerID)
		waitErr = errors.Wrapf(someErr, "Wait() customer %s", customerID)
	)

	queryMatch := mock.MatchedBy(func(arg *bigquery.Query) bool {
		idCheck := arg.Parameters[0].Value == customerID

		firstDayStr, ok := arg.Parameters[1].Value.(string)
		if !ok {
			return false
		}

		firstDay, err := time.Parse(times.YearMonthDayLayout, firstDayStr)
		if err != nil {
			return false
		}

		dayOneCheck := firstDay.Day() == 1

		lastDayStr, ok := arg.Parameters[2].Value.(string)
		if !ok {
			return false
		}

		lastDay, err := time.Parse(times.YearMonthDayLayout, lastDayStr)
		if err != nil {
			return false
		}

		daysInMonth := time.Date(lastDay.Year(), lastDay.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()

		lastDayCheck := lastDay.Day() == daysInMonth

		return idCheck && dayOneCheck && lastDayCheck
	})

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.queryHandler.On("Run", ctx, queryMatch).Return(&job, nil)
				f.jobHandler.On("Wait", ctx, &job).Return(&status, nil)
			},
		},
		{
			name: "failure Run",
			on: func(f *fields) {
				f.queryHandler.On("Run", ctx, queryMatch).Return(&job, someErr)
			},
			wantErr: runErr,
		},
		{
			name: "failure Wait",
			on: func(f *fields) {
				f.queryHandler.On("Run", ctx, queryMatch).Return(&job, nil)
				f.jobHandler.On("Wait", ctx, &job).Return(&status, someErr)
			},
			wantErr: waitErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &Service{
				queryHandler: &fields.queryHandler,
				jobHandler:   &fields.jobHandler,
			}

			err := s.deleteExistingRows(ctx, customerID, firstOfMonth, lastOfMonth)

			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_validateEntities(t *testing.T) {
	var (
		ctx                  = &gin.Context{}
		ids                  = []string{"xxx", "yyy"}
		totalSavings float64 = -50
	)

	logging, err := logger.NewLogging(ctx)
	if err != nil {
		assert.NoError(t, err)
	}

	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		assert.NoError(t, err)
	}

	var refs []*firestore.DocumentRef

	fs := conn.Firestore(ctx)

	for _, id := range ids {
		refs = append(refs, fs.Doc("entities/"+id))
	}

	c := &common.Customer{Entities: refs, Snapshot: &firestore.DocumentSnapshot{
		Ref: refs[0],
	}}

	noEntityErr := fmt.Errorf("customer has %v entities, please provide all in the request body with adjusted amount. Required format: %v", len(c.Entities), `{"`+strings.Join(ids, `": 0, "`)+`": 0}`)
	numEntityErr := fmt.Errorf("customer %s expected %v entities in request and received %v, customer entities are: %v", c.Snapshot.Ref.ID, len(c.Entities), 1, strings.Join(ids, ","))
	matchEntityErr := fmt.Errorf("entity values do not match with existing for customer, customer has: %v, entities received in request :%v", strings.Join(ids, ","), strings.Join([]string{"xxx", "zzz"}, ","))
	savingsMathErr := fmt.Errorf("customer %s, total savings (%v) and sum of entities savings received (%v) do not match : %v", c.Snapshot.Ref.ID, totalSavings, -10, map[string]float64{"xxx": -10, "yyy": 0})
	positiveSavingsErr := fmt.Errorf("only negative savings allowed, request contained positive numbers %+v", map[string]float64{"xxx": 10, "yyy": 0})

	type args struct {
		customer        *common.Customer
		requestEntities map[string]float64
		totalSavings    float64
		monthDate       time.Time
	}

	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "no entities in request",
			args: args{
				customer:        c,
				requestEntities: nil,
				totalSavings:    totalSavings,
				monthDate:       time.Time{},
			},
			wantErr: noEntityErr,
		},
		{
			name: "number of entities not matching",
			args: args{
				customer:        c,
				requestEntities: map[string]float64{"xxx": 0},
				totalSavings:    totalSavings,
				monthDate:       time.Time{},
			},
			wantErr: numEntityErr,
		},
		{
			name: "passed entities are not within customer entities",
			args: args{
				customer: &common.Customer{Entities: refs, Snapshot: &firestore.DocumentSnapshot{
					Ref: refs[0],
				}},
				requestEntities: map[string]float64{"xxx": 0, "zzz": 0},
				totalSavings:    totalSavings,
				monthDate:       time.Time{},
			},
			wantErr: matchEntityErr,
		},
		{
			name: "total savings not equal summed entities savings",
			args: args{
				customer: &common.Customer{Entities: refs, Snapshot: &firestore.DocumentSnapshot{
					Ref: refs[0],
				}},
				requestEntities: map[string]float64{"xxx": -10, "yyy": 0},
				totalSavings:    totalSavings,
				monthDate:       time.Time{},
			},
			wantErr: savingsMathErr,
		},
		{
			name: "positive savings in request",
			args: args{
				customer: &common.Customer{Entities: refs, Snapshot: &firestore.DocumentSnapshot{
					Ref: refs[0],
				}},
				requestEntities: map[string]float64{"xxx": 10, "yyy": 0},
				totalSavings:    totalSavings,
				monthDate:       time.Time{},
			},
			wantErr: positiveSavingsErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEntities(tt.args.customer, tt.args.requestEntities, tt.args.totalSavings)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}
