package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/priority/dal/iface"
	httpClient "github.com/doitintl/http"
)

type dal struct {
	loggerProvider          logger.Provider
	priorityFirestore       iface.PriorityFirestore
	priorityClient          httpClient.IClient
	avalaraClient           httpClient.IClient
	priorityProcedureClient httpClient.IClient

	priorityUserName string
	priorityPassword string
}

func NewPriorityDAL(
	loggerProvider logger.Provider,
	priorityFirestore iface.PriorityFirestore,
	priorityClient httpClient.IClient,
	avalaraClient httpClient.IClient,
	priorityProcedureClient httpClient.IClient,
	opts ...Option,
) (iface.ReaderWriter, error) {
	d := &dal{
		loggerProvider:          loggerProvider,
		priorityFirestore:       priorityFirestore,
		priorityClient:          priorityClient,
		avalaraClient:           avalaraClient,
		priorityProcedureClient: priorityProcedureClient,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d, d.validate()
}

func (d *dal) validate() error {
	if d.loggerProvider == nil {
		return ErrLogIsNil
	}

	if d.priorityClient == nil {
		return ErrPriorityClientIsNil
	}

	if d.priorityProcedureClient == nil {
		return ErrPriorityProcedureClientIsNil
	}

	if d.priorityUserName == "" {
		return ErrPriorityUserNameIsEmpty
	}

	if d.priorityPassword == "" {
		return ErrPriorityPasswordIsEmpty
	}

	return nil
}

func (d *dal) withBasicAuth(ctx context.Context) context.Context {
	return httpClient.WithBasicAuth(ctx, &httpClient.BasicAuthContextData{
		User:     d.priorityUserName,
		Password: d.priorityPassword,
	})
}
