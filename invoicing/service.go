package invoicing

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/bigquery"
	bigqueryIface "github.com/doitintl/bigquery/iface"
	"github.com/doitintl/cloudtasks/iface"
	mpaDAL "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	reportDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	contractDal "github.com/doitintl/hello/scheduled-tasks/contract/dal"
	contractDalIface "github.com/doitintl/hello/scheduled-tasks/contract/dal/iface"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	customersDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/aws"
	invoicingDal "github.com/doitintl/hello/scheduled-tasks/invoicing/dal"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/doitproducts"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	flexsaveDal "github.com/doitintl/hello/scheduled-tasks/invoicing/flexsave/dal"
	lookerIface "github.com/doitintl/hello/scheduled-tasks/invoicing/looker/iface"
	looker "github.com/doitintl/hello/scheduled-tasks/invoicing/looker/service"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type commonAWSInvoicing interface {
	CalculateSpendAndCreditsData(invoiceMonthString string,
		accountID string, date time.Time, cost float64,
		entityRef *firestore.DocumentRef,
		assetRef *firestore.DocumentRef,
		credits []*aws.CustomerCreditAmazonWebServices,
		accountsData map[string]float64,
		creditsData map[string]map[string]float64)
	GetAmazonWebServicesCredits(ctx context.Context,
		invoiceMonth time.Time,
		customerRef *firestore.DocumentRef,
		accountsWithAssets []string) ([]*aws.CustomerCreditAmazonWebServices, error)
}

type InvoicingService struct {
	*logger.Logging
	*connection.Connection
	fixerService               *fixer.FixerService
	contractDAL                contractDalIface.ContractFirestore
	customersDAL               customersDal.Customers
	customerAssetInvoice       CustomerAssetInvoice
	lookerInvoicingService     lookerIface.InvoicingService
	customerDoITPackageInvoice CustomerDoITPackageInvoice
	awsAnalyticsService        AnalyticsAWSInvoicing
}

type IssuedInvoice interface {
	CancelIssuedInvoices(ctx context.Context, request *CancelIssuedInvoicesReq, email string) error
}

type CustomerAssetInvoice interface {
	GetAWSInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows)
	GetAWSStandaloneInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows)
	GetGCPStandaloneInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows)
	IsUseAnalyticsDataForInvoice(ctx context.Context, customerRef *firestore.DocumentRef) (bool, error)
}

type CustomerDoITPackageInvoice interface {
	GetDoITNavigatorInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows)
	GetDoITSolveInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows)
	GetDoITSolveAcceleratorInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows)
}

type CustomerAssetInvoiceWorker struct {
	*connection.Connection
	loggerProvider        logger.Provider
	customersDAL          customersDal.Customers
	monthlyBillingDataDAL invoicingDal.MonthlyBillingData
	flexsaveDAL           flexsaveDal.FlexsaveStandalone
	assetsDAL             assetsDal.Assets
}

type commonAWSInvoicingService struct {
	loggerProvider logger.Provider
}

type CloudHealthAWSInvoicingService struct {
	loggerProvider     logger.Provider
	common             commonAWSInvoicing
	invoiceMonthParser InvoiceMonthParser
	billingData        BillingData
	*connection.Connection
}

type AnalyticsAWSInvoicing interface {
	UpdateAmazonWebServicesInvoicingData(ctx context.Context, invoiceMonth, version string, validateWithOld, dry bool) error
	AmazonWebServicesInvoicingDataWorker(ginCtx *gin.Context, customerID, invoiceMonthInput string, dry bool) error
}

type AnalyticsAWSInvoicingService struct {
	loggerProvider        logger.Provider
	common                commonAWSInvoicing
	invoiceMonthParser    InvoiceMonthParser
	billingData           BillingData
	assetsDAL             assetsDal.Assets
	assetSettingsDAL      assetsDal.AssetSettings
	monthlyBillingDataDAL invoicingDal.MonthlyBillingData
	customers             customersDal.Customers
	cloudTaskClient       iface.CloudTaskClient
	flexsaveAPI           flexapi.FlexAPI
}

func NewAWSCustomerAssetInvoiceWorker(conn *connection.Connection) *CustomerAssetInvoiceWorker {
	return &CustomerAssetInvoiceWorker{
		conn,
		logger.FromContext,
		customersDal.NewCustomersFirestoreWithClient(conn.Firestore),
		invoicingDal.NewMonthlyBillingDataFirestoreWithClient(conn.Firestore),
		flexsaveDal.NewFlexsaveStandaloneFirestoreWithClient(conn.Firestore),
		assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
	}
}

func NewInvoicingService(log *logger.Logging, conn *connection.Connection) (*InvoicingService, error) {
	fixerService, err := fixer.NewFixerService(logger.FromContext, conn)
	if err != nil {
		return nil, err
	}

	lookerInvoicingService, err := looker.NewInvoicingService(logger.FromContext, conn)
	if err != nil {
		return nil, err
	}

	doitPackageInvoicingService, err := doitproducts.NewDoITPackageService(logger.FromContext, conn)
	if err != nil {
		return nil, err
	}

	flexapiService, err := flexapi.NewFlexAPIService()
	if err != nil {
		return nil, err
	}

	awsAnalyticsService, err := NewAnalyticsAWSInvoicingService(conn, conn.Firestore, conn.CloudTaskClient, conn.Bigquery, flexapiService)
	if err != nil {
		return nil, err
	}

	return &InvoicingService{
		log,
		conn,
		fixerService,
		contractDal.NewContractFirestoreWithClient(conn.Firestore),
		customersDal.NewCustomersFirestoreWithClient(conn.Firestore),
		NewAWSCustomerAssetInvoiceWorker(conn),
		lookerInvoicingService,
		doitPackageInvoicingService,
		awsAnalyticsService,
	}, nil
}

func NewCloudHealthAWSInvoicingService(conn *connection.Connection) (*CloudHealthAWSInvoicingService, error) {
	awsInvoicingService := commonAWSInvoicingService{logger.FromContext}

	customerDal := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)
	reportDal := reportDAL.NewReportsFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(logger.FromContext, conn, reportDal, customerDal)
	if err != nil {
		return nil, err
	}

	billingData := BillingDataService{
		logger.FromContext,
		&BillingDataQueryBuilder{},
		&queryResultTransformer{},
		cloudAnalytics,
		bigquery.QueryHandler{},
		conn.Bigquery,
		conn.Firestore,
		mpaDAL.NewMasterPayerAccountDALWithClient(conn.Firestore(context.Background())),
	}

	parser := DefaultInvoiceMonthParser{InvoicingDaySwitchOver: 10}

	return &CloudHealthAWSInvoicingService{
		logger.FromContext,
		&awsInvoicingService,
		&parser,
		&billingData,
		conn,
	}, nil
}

func NewAnalyticsAWSInvoicingService(conn *connection.Connection, firestoreFun connection.FirestoreFromContextFun, cloudTaskClient iface.CloudTaskClient, bigQueryFromContextFun connection.BigQueryFromContextFun, flexapiService flexapi.FlexAPI) (*AnalyticsAWSInvoicingService, error) {
	awsInvoicingService := commonAWSInvoicingService{logger.FromContext}

	customerDal := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)
	reportDal := reportDAL.NewReportsFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(logger.FromContext, conn, reportDal, customerDal)
	if err != nil {
		return nil, err
	}

	billingData := BillingDataService{
		logger.FromContext,
		&BillingDataQueryBuilder{},
		&queryResultTransformer{},
		cloudAnalytics,
		bigquery.QueryHandler{},
		bigQueryFromContextFun,
		firestoreFun,
		mpaDAL.NewMasterPayerAccountDALWithClient(firestoreFun(context.Background())),
	}

	parser := DefaultInvoiceMonthParser{InvoicingDaySwitchOver: 10}

	return &AnalyticsAWSInvoicingService{
		logger.FromContext,
		&awsInvoicingService,
		&parser,
		&billingData,
		assetsDal.NewAssetsFirestoreWithClient(firestoreFun),
		assetsDal.NewAssetSettingsFirestoreWithClient(firestoreFun),
		invoicingDal.NewMonthlyBillingDataFirestoreWithClient(firestoreFun),
		customersDal.NewCustomersFirestoreWithClient(firestoreFun),
		cloudTaskClient,
		flexapiService,
	}, nil
}

func NewBillingDataService(conn *connection.Connection) *BillingDataService {
	mpaDal := mpaDAL.NewMasterPayerAccountDALWithClient(conn.Firestore(context.Background()))

	customerDal := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)
	reportDal := reportDAL.NewReportsFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(logger.FromContext, conn, reportDal, customerDal)
	if err != nil {
		panic(err)
	}
	billingData := BillingDataService{
		logger.FromContext,
		&BillingDataQueryBuilder{},
		&queryResultTransformer{},
		cloudAnalytics,
		bigquery.QueryHandler{},
		conn.Bigquery,
		conn.Firestore,
		mpaDal,
	}

	return &billingData
}

//go:generate mockery --name BillingData --output ./mocks --outpkg mocks
type BillingData interface {
	GetBillableAssetIDs(ctx context.Context, invoiceMonth time.Time) ([]string, error)
	GetBillableCustomerIDs(ctx context.Context, invoiceMonth time.Time) ([]string, []string, []string, error)
	GetCloudhealthCustomerIDsFromFirestore(ctx context.Context) (map[string]string, error)
	GetStandaloneCustomerIDsFromFirestore(ctx context.Context) ([]string, error)
	GetCustomerBillingData(ctx *gin.Context, customerID string, invoiceMonth time.Time) (map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem, []string, error)
	GetCustomerInvoicingReadiness(ctx context.Context, customerID string, invoiceMonth time.Time, invoicingDaySwitchOver int) (bool, error)
	SnapshotCustomerBillingTable(ctx context.Context, customerID string, invoiceMonth time.Time) error
	HasCustomerInvoiceBeenIssued(ctx context.Context, customerID string, invoiceMonth time.Time) (bool, error)
	HasAnyInvoiceBeenIssued(ctx context.Context, invoiceMonth string) (bool, error)
	GetCustomerBillingSessionID(ctx context.Context, customerID string, invoiceMonth time.Time) string
	SaveCreditUtilizationToFS(ctx context.Context, invoiceMonth time.Time, credits []*aws.CustomerCreditAmazonWebServices) error
}

type BillingDataService struct {
	loggerProvider         logger.Provider
	billingDataQuery       BillingDataQuery
	billingDataTransformer billingDataResult
	cloudAnalytics         cloudanalytics.CloudAnalytics
	queryHandler           bigqueryIface.QueryHandler
	bigQueryClientFunc     connection.BigQueryFromContextFun
	firestoreClientFunc    connection.FirestoreFromContextFun
	mpaDAL                 mpaDAL.MasterPayerAccounts
}
