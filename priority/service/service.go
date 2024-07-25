package service

import (
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/priority/dal"
	dalIface "github.com/doitintl/hello/scheduled-tasks/priority/dal/iface"
	serviceIface "github.com/doitintl/hello/scheduled-tasks/priority/service/iface"
)

type service struct {
	loggerProvider       logger.Provider
	conn                 *connection.Connection
	priorityFirestore    dal.PriorityFirestore
	priorityReaderWriter dalIface.ReaderWriter
}

func NewService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	priorityFirestore dal.PriorityFirestore,
	priorityReaderWriter dalIface.ReaderWriter,
) (serviceIface.Service, error) {
	s := &service{
		loggerProvider:       loggerProvider,
		conn:                 conn,
		priorityFirestore:    priorityFirestore,
		priorityReaderWriter: priorityReaderWriter,
	}

	return s, s.validate()
}

func (s *service) validate() error {
	if s.loggerProvider == nil {
		return ErrLogIsNil
	}

	if s.priorityReaderWriter == nil {
		return ErrPriorityReaderWriterIsNil
	}

	return nil
}
