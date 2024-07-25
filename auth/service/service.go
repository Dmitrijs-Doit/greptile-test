package service

import (
	"context"

	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AuthService struct {
	loggerProvider logger.Provider
	*connection.Connection
	customersDAL customerDal.Customers
}

func NewAuthService(log logger.Provider, conn *connection.Connection) *AuthService {
	return &AuthService{
		log,
		conn,
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
	}
}

func (s *AuthService) Validate(ctx context.Context, customerID string) (string, error) {
	c, e := s.customersDAL.GetCustomer(ctx, customerID)
	if e != nil {
		return "", e
	}

	return c.PrimaryDomain, nil
}
