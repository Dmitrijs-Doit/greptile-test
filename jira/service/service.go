package service

import (
	"context"
	"errors"

	"github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/jira/dal"
	"github.com/doitintl/jira/dal/iface"
)

type JiraService struct {
	loggerProvider logger.Provider
	instancesDAL   iface.InstancesDAL
}

func NewJiraService(ctx context.Context, loggerProvider logger.Provider, conn *connection.Connection) *JiraService {
	return &JiraService{
		loggerProvider: loggerProvider,
		instancesDAL:   dal.NewInstancesDAL(conn.Firestore(ctx), firestore.DocumentHandler{}),
	}
}

func (s *JiraService) CreateInstance(ctx context.Context, customerID string, url string) error {
	instanceExists, err := s.instancesDAL.CustomerInstanceExists(ctx, customerID)
	if err != nil {
		return err
	}

	if instanceExists {
		return errors.New("only one instance allowed")
	}

	l := s.loggerProvider(ctx)

	err = s.instancesDAL.CreatePendingInstance(ctx, customerID, url)
	if err != nil {
		l.Errorf("error creating jira instance: %v", err)
		return err
	}

	l.Info("successfully created jira instance")

	return nil
}
