package bigquery

import (
	"errors"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type GetJobByIDParams struct {
	JobID      string
	CustomerID string
	Project    string
	Location   string
}

type Service struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
}

func NewService(loggerProvider logger.Provider, conn *connection.Connection) *Service {
	return &Service{
		loggerProvider,
		conn,
	}
}

func (s *Service) GetQueryFromJob(ctx *gin.Context, job *bigquery.Job) (string, error) {
	if job == nil {
		return "", errors.New("query job is nil")
	}

	jc, err := job.Config()
	if err != nil {
		return "", err
	}

	queryConfig, ok := jc.(*bigquery.QueryConfig)
	if !ok {
		return "", errors.New("job is not a query")
	}

	return queryConfig.Q, nil
}

func (s *Service) GetCustomerJob(ctx *gin.Context, params GetJobByIDParams) (*bigquery.Job, error) {
	fs := s.conn.Firestore(ctx)
	l := s.loggerProvider(ctx)

	cloudConnectCreds, err := getCustomerBQLensCloudConnectCred(ctx, fs, params.CustomerID)
	if err != nil {
		return nil, err
	}

	for _, cloudConnect := range cloudConnectCreds {
		customerCreds, err := common.NewGcpCustomerAuthService(cloudConnect).GetClientOption()
		if err != nil {
			return nil, err
		}

		bq, err := bigquery.NewClient(ctx, params.Project, customerCreds)
		if err != nil {
			return nil, err
		}
		defer bq.Close()

		job, err := bq.JobFromIDLocation(ctx, params.JobID, params.Location)
		if err != nil {
			return nil, err
		}

		if job == nil {
			l.Infof("failed to get job %s for customer %s using %s", params.JobID, params.CustomerID, cloudConnect.ClientID)
			continue
		}

		return job, nil
	}

	return nil, fmt.Errorf("job %s not found for customer %s", params.JobID, params.CustomerID)
}
