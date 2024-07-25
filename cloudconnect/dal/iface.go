//go:generate mockery --output=./mocks --all
package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/logging/v2"

	bq "github.com/doitintl/bigquery"
	cl "github.com/doitintl/cloudlogging"
	crm "github.com/doitintl/cloudresourcemanager"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/pkg"
	su "github.com/doitintl/serviceusage"
)

// IGcpConnect is used to interact with GCP Connect
type IGcpConnect interface {
	GetCredentials(ctx context.Context, customerID string) ([]*common.GoogleCloudCredential, error)
	GetBigQueryLensCredentials(ctx context.Context, customerID string) ([]*common.GoogleCloudCredential, error)
	GetCredentialByOrg(ctx context.Context, cloudConectDoc *firestore.DocumentRef) (*common.GoogleCloudCredential, error)
	GetClientOption(ctx context.Context, cloudConectDoc *firestore.DocumentRef) (*pkg.GcpClientOption, error)
	GetConnectDetails(ctx context.Context, cloudConectDoc *firestore.DocumentRef) (*common.GCPConnectOrganization, string, error)
	SaveSinkDestination(ctx context.Context, sinkDestination string, cloudConectDoc *firestore.DocumentRef) error
	SaveSinkMetadata(ctx context.Context, data *pkg.SinkMetadata, cloudConectDoc *firestore.DocumentRef) (*firestore.WriteResult, error)
	GetSinkParams(sinkDestination string) *logging.LogSink
	GetSinkDestination(projectID string) string
	GetBQLensCustomersDocs(ctx context.Context) ([]*firestore.DocumentSnapshot, error)

	NewCloudResourceManager(ctx context.Context, options *pkg.GcpClientOption) (*crm.Service, error)
	NewBigQuery(ctx context.Context, options *pkg.GcpClientOption) (*bq.Service, error)
	NewCloudLogging(ctx context.Context, options *pkg.GcpClientOption) (*cl.Service, error)
	NewServiceUsage(ctx context.Context, options *pkg.GcpClientOption) (*su.Service, error)
}

type IAwsConnect interface {
	GetSpot0CustomerFlags(ctx context.Context, customerID string) (*Spot0CustomerFlags, error)
	SetSpot0CustomerFlags(ctx context.Context, customerID string) error
	GetCustomer(ctx context.Context, customerID string) (*common.Customer, error)
	GetCustomerAccountManagers(ctx context.Context, customer *common.Customer, company common.AccountManagerCompany) ([]*common.AccountManager, error)
	GetCustomerAdmins(ctx context.Context, customerID string) ([]common.User, error)

	SendMail(ctx context.Context, mailRecipients []MailRecipient, bccs []string, companyName string) error
}
