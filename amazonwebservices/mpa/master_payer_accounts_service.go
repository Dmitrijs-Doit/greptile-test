package mpa

import (
	"context"
	"strings"

	fsdal "github.com/doitintl/firestore"
	accountsDAL "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"

	"github.com/doitintl/googleadmin"

	"github.com/doitintl/cloudtasks/iface"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/http"
)

type MasterPayerAccountService struct {
	loggerProvider logger.Provider
	*connection.Connection
	awsClient                                        dal.IAWSDal
	mpaDAL                                           accountsDAL.MasterPayerAccounts
	cloudConnectDAL                                  fsdal.CloudConnect
	customersDAL                                     customerDal.Customers
	googleAdmin                                      googleadmin.GoogleAdmin
	cloudTaskClient                                  iface.CloudTaskClient
	sauronClient                                     http.IClient
	clientKeys                                       *clientKeys
	doitPolicyTemplateFile                           *string
	saasDoitRoleTemplateFile                         *string
	saasDoitPolicyTemplateFile                       *string
	doitPolicyOptionalPermissionsMaskFile            *string
	saasDoitPolicyConditionalPermissionsTemplateFile *string
}

func NewMasterPayerAccountService(log logger.Provider, conn *connection.Connection) (IMPAService, error) {
	ctx := context.Background()

	googleAdmin, err := googleadmin.NewGoogleAdmin(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	sauronURL := strings.Replace(amazonwebservices.GetSauronURL(), "/cmp", "", -1)
	sauronConfig := http.Config{BaseURL: sauronURL}

	sauronClient, err := http.NewClient(ctx, &sauronConfig)
	if err != nil {
		return nil, err
	}

	sauronAPIKey, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretSauronAPIKey)
	if err != nil {
		return nil, err
	}

	clientKeys := clientKeys{SauronApiKey: string(sauronAPIKey)}

	return &MasterPayerAccountService{
		log,
		conn,
		dal.NewAWSDal(),
		accountsDAL.NewMasterPayerAccountDALWithClient(conn.Firestore(ctx)),
		fsdal.NewCloudConnectDALWithClient(conn.Firestore(ctx)),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		googleAdmin,
		conn.CloudTaskClient,
		sauronClient,
		&clientKeys,
		&doitPolicyTemplateFile,
		&saasDoitRoleTemplateFile,
		&saasDoitPolicyTemplateFile,
		&doitPolicyOptionalPermissionsMaskTemplateFile,
		&saasDoitPolicyConditionalPermissionsTemplateFile,
	}, nil
}
