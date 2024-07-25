package service

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	csmDal "github.com/doitintl/hello/scheduled-tasks/csmengagement/dal"
	"github.com/doitintl/hello/scheduled-tasks/csmengagement/domain"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"

	customerTypeDal "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	userDal "github.com/doitintl/hello/scheduled-tasks/user/dal"
	userDalIface "github.com/doitintl/hello/scheduled-tasks/user/dal/iface"
	notificationSender "github.com/doitintl/notificationcenter/pkg"
)

type Service interface {
	GetNoAttributionsEmails(ctx context.Context) ([]domain.NoAttributionEmailParams, error)
	SendNoAttributionsEmails(ctx context.Context, emailsToSend []domain.NoAttributionEmailParams) error
	SendNewAttributionEmails(ctx context.Context) error
	SendFirstInvoiceEmail(ctx context.Context, customerID string) error
	SendNoCustomerEngagementNotifications(ctx context.Context) error
}

type service struct {
	l                  logger.ILogger
	fs                 *firestore.Client
	notificationSender notificationSender.NotificationSender
	csmService         CSMEngagement
	csmEngagementDAL   csmDal.CSMEngagementDAL
	customerDAL        customerDal.Customers
	userDAL            userDalIface.IUserFirestoreDAL
	noAttrsDAL         csmDal.INoAttributionsEmail
	customerTypeDal    customerTypeDal.CustomerTypeIface
}

func NewService(ctx context.Context, fs *firestore.Client, l logger.ILogger) Service {
	log := l
	if log == nil {
		log = logger.FromContext(ctx)
	}

	client, err := notificationSender.NewClient(ctx, common.ProjectID)
	if err != nil {
		panic(err)
	}

	fsFunc := func(ctx context.Context) *firestore.Client {
		return fs
	}

	return &service{
		log,
		fs,
		client,
		NewCsmEngagement(fs),
		csmDal.NewCSMEngagementDAL(fsFunc),
		customerDal.NewCustomersFirestoreWithClient(fsFunc),
		userDal.NewUserFirestoreDALWithClient(fsFunc),
		csmDal.NewNoAttributionsEmail(fs),
		customerTypeDal.NewCustomerTypeDALWithClient(fs),
	}
}
