package service

import (
	"context"
	"fmt"
	"strings"

	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	pkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/billing-explainer/dal"
	billingExplainerDalIface "github.com/doitintl/hello/scheduled-tasks/billing-explainer/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/billing-explainer/domain"
	"github.com/doitintl/hello/scheduled-tasks/billing-explainer/utils"
	bucketsDal "github.com/doitintl/hello/scheduled-tasks/buckets/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	entityDal "github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type BillingExplainerService struct {
	loggerProvider logger.Provider
	bigQueryDal    billingExplainerDalIface.BigQueryDAL
	firestoreDal   billingExplainerDalIface.FirestoreDAL
	bucketsDal     bucketsDal.Buckets
	assetsDal      assetsDal.Assets
	entityDal      entityDal.Entites
}

const (
	BillingExplainerProjectProd = "doit-bq-billing-explainer"
	BillingExplainerProjectDev  = "doit-bq-billing-explainer-dev"
)

func NewBillingExplainerService(loggerProvider logger.Provider, conn *connection.Connection) *BillingExplainerService {
	ctx := context.Background()

	billingExplainerProject := BillingExplainerProjectDev

	if common.Production {
		billingExplainerProject = BillingExplainerProjectProd
	}

	return &BillingExplainerService{
		loggerProvider,
		dal.NewBigQueryDAL(ctx, billingExplainerProject),
		dal.NewFirestoreDAL(ctx, common.ProjectID),
		bucketsDal.NewBucketsFirestoreWithClient(conn.Firestore),
		assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
		entityDal.NewEntitiesFirestoreWithClient(conn.Firestore),
	}
}

func isSharedPayer(payerID string) bool {
	var SharedPayers = []string{"561602220360", "017920819041", "279843869311"}

	for _, value := range SharedPayers {
		if value == payerID {
			return true
		}
	}

	return false
}

func (s *BillingExplainerService) GetPayerInfoFromCustID(ctx context.Context, customerID string, startOfMonth string) ([]domain.PayerAccountInfoStruct, error) {
	log := s.loggerProvider(ctx)

	var payerAccountInfoStructList []domain.PayerAccountInfoStruct

	payerAccounts, err := s.bigQueryDal.GetPayerIDFromAccountsHistory(ctx, startOfMonth, customerID)
	if err != nil {
		return payerAccountInfoStructList, err
	}

	for _, payerAccount := range payerAccounts {
		var payerAccountInfoStruct domain.PayerAccountInfoStruct

		payerAccountInfoStruct.PayerID = payerAccount.PayerID

		mpaDoc, err := s.firestoreDal.GetPayerAccountDoc(ctx, payerAccount.PayerID)
		if err != nil {
			return payerAccountInfoStructList, err
		}

		if mpaDoc == nil {
			// Payer does not exist in firestore
			log.Infof("Payer doc does not exist in Firestore for PayerID %s", payerAccount.PayerID)
			continue
		}

		if mpaFriendlyName, ok := mpaDoc["friendlyName"].(string); ok {
			payerAccountInfoStruct.FriendlyName = mpaFriendlyName
		}

		payerAccountInfoStructList = append(payerAccountInfoStructList, payerAccountInfoStruct)
	}

	return payerAccountInfoStructList, nil
}

func (s *BillingExplainerService) GetFlexsaveCondition(PayerID string, isDefaultBucket bool) string {
	isSharedPayer := isSharedPayer(PayerID)

	var flexsaveCondition string

	if !isSharedPayer {
		flexsaveAccountsTableID := fmt.Sprintf("%s.measurement.flexsave_accounts", "me-doit-intl-com")

		cond := " OR (project_id IN (SELECT DISTINCT(aws_account_id) FROM %s) AND %v=true)"

		flexsaveCondition = fmt.Sprintf(cond, flexsaveAccountsTableID, isDefaultBucket)
	} else {
		flexsaveCondition = ""
	}

	return flexsaveCondition
}

func (s *BillingExplainerService) GetSummaryPageData(ctx context.Context, explainerParams domain.BillingExplainerParams, accountIDString string, payerTable string, PayerID string, isDefaultBucket bool) ([]domain.SummaryBQ, error) {
	log := s.loggerProvider(ctx)

	var data []domain.SummaryBQ

	flexsaveCondition := s.GetFlexsaveCondition(PayerID, isDefaultBucket)

	data, err := s.bigQueryDal.GetInvoiceSummary(ctx, explainerParams, payerTable, accountIDString, PayerID, flexsaveCondition)
	if err != nil {
		log.Errorf("Fail to get invoice summary for customerID %s and %s: %s", explainerParams.CustomerID, explainerParams.InvoiceMonth, err)
		return data, err
	}

	return data, nil
}

func (s *BillingExplainerService) GetDataByServiceBreakdown(ctx context.Context, explainerParams domain.BillingExplainerParams, accountIDString string, payerTable string, PayerID string, isDefaultBucket bool) ([]domain.ServiceRecord, error) {
	log := s.loggerProvider(ctx)

	var data []domain.ServiceRecord

	flexsaveCondition := s.GetFlexsaveCondition(PayerID, isDefaultBucket)

	data, err := s.bigQueryDal.GetServiceBreakdownData(ctx, explainerParams, payerTable, accountIDString, PayerID, flexsaveCondition)
	if err != nil {
		log.Errorf("Fail to get data by service breakdown for customerID %s and %s", explainerParams.CustomerID, explainerParams.InvoiceMonth)
		return data, err
	}

	return data, nil
}

func (s *BillingExplainerService) GetDataByAccountBreakdown(ctx context.Context, explainerParams domain.BillingExplainerParams, accountIDString string, payerTable string, PayerID string, isDefaultBucket bool) ([]domain.AccountRecord, error) {
	log := s.loggerProvider(ctx)

	var data []domain.AccountRecord

	flexsaveCondition := s.GetFlexsaveCondition(PayerID, isDefaultBucket)

	data, err := s.bigQueryDal.GetAccountBreakdownData(ctx, explainerParams, payerTable, accountIDString, PayerID, flexsaveCondition)
	if err != nil {
		log.Errorf("Fail to get data by account breakdown for customerID %s and %s", explainerParams.CustomerID, explainerParams.InvoiceMonth)
		return data, err
	}

	return data, nil
}

func (s *BillingExplainerService) ProcessAssetsInBucket(ctx context.Context, explainerParams domain.BillingExplainerParams, assets []*pkg.BaseAsset, payerTable, bucketName, PayerID string) ([]domain.SummaryBQ, []domain.ServiceRecord, []domain.AccountRecord, string, error) {
	var assetList []string

	var data []domain.SummaryBQ

	var serviceData []domain.ServiceRecord

	var accountData []domain.AccountRecord

	var err error

	for _, asset := range assets {
		if asset.AssetType == common.Assets.AmazonWebServices {
			parts := strings.Split(asset.ID, "-")

			if len(parts) > 0 {
				assetID := parts[len(parts)-1]
				assetList = append(assetList, assetID)
			}
		}
	}

	if len(assetList) > 0 {
		accountIDString := "'" + strings.Join(assetList, "','") + "'"

		// Check if the bucket is the default bucket. Set to true if no buckets are found (aka bucketName = "bucket") or the invoicing mode is not CUSTOM
		isDefaultBucket := bucketName == explainerParams.DefaultBucket || bucketName == "bucket" || explainerParams.InvoicingMode != "CUSTOM"

		data, err = s.GetSummaryPageData(ctx, explainerParams, accountIDString, payerTable, PayerID, isDefaultBucket)
		if err != nil {
			return data, serviceData, accountData, bucketName, err
		}

		serviceData, err = s.GetDataByServiceBreakdown(ctx, explainerParams, accountIDString, payerTable, PayerID, isDefaultBucket)
		if err != nil {
			return data, serviceData, accountData, bucketName, err
		}

		accountData, err = s.GetDataByAccountBreakdown(ctx, explainerParams, accountIDString, payerTable, PayerID, isDefaultBucket)
		if err != nil {
			return data, serviceData, accountData, bucketName, err
		}
	}

	return data, serviceData, accountData, bucketName, nil
}

func (s *BillingExplainerService) CreateBucketAssetsMap(ctx context.Context, entityBuckets []common.Bucket) (map[string][]*pkg.BaseAsset, error) {
	assetsBucketMap := make(map[string][]*pkg.BaseAsset)

	for _, bucket := range entityBuckets {
		if strings.Contains(bucket.Name, "GCP") {
			continue
		}

		assets, err := s.assetsDal.GetAssetsInBucket(ctx, bucket.Ref)
		if err != nil {
			return assetsBucketMap, err
		}

		if _, ok := assetsBucketMap[bucket.Name]; ok {
			assetsBucketMap[bucket.Name] = append(assetsBucketMap[bucket.Name], assets...)
		} else {
			assetsBucketMap[bucket.Name] = assets
		}
	}

	return assetsBucketMap, nil
}

func (s *BillingExplainerService) ProcessAssets(ctx context.Context, explainerParams domain.BillingExplainerParams, bucketAssetsMap map[string][]*pkg.BaseAsset, payerTable string, bucketMap map[string][]domain.SummaryBQ, serviceBucketMap map[string][]domain.ServiceRecord, accountBucketMap map[string][]domain.AccountRecord, PayerID string) error {
	for bucketName, assets := range bucketAssetsMap {
		data, serviceData, accountData, _, err := s.ProcessAssetsInBucket(ctx, explainerParams, assets, payerTable, bucketName, PayerID)
		if err != nil {
			return err
		}

		if _, ok := bucketMap[bucketName]; ok {
			// If the bucket name exists, append the new result to the existing slice
			bucketMap[bucketName] = append(bucketMap[bucketName], data...)
			serviceBucketMap[bucketName] = append(serviceBucketMap[bucketName], serviceData...)
			accountBucketMap[bucketName] = append(accountBucketMap[bucketName], accountData...)
		} else {
			// If the bucket name doesn't exist, create a new slice with the result
			bucketMap[bucketName] = data
			serviceBucketMap[bucketName] = serviceData
			accountBucketMap[bucketName] = accountData
		}
	}

	return nil
}

func (s *BillingExplainerService) ProcessAssetsForEntity(ctx context.Context, explainerParams domain.BillingExplainerParams, entityID, payerTable string, PayerID string) ([]domain.SummaryBQ, []domain.ServiceRecord, []domain.AccountRecord, string, error) {
	log := s.loggerProvider(ctx)

	entityRef := s.entityDal.GetRef(ctx, entityID)

	var finalData []domain.SummaryBQ

	var serviceData []domain.ServiceRecord

	var accountData []domain.AccountRecord

	assets, err := s.assetsDal.GetAssetsInEntity(ctx, entityRef)
	if err != nil {
		log.Errorf("Fail to get assets for entityID %s and customerID %s", entityID, explainerParams.CustomerID)
		return finalData, serviceData, accountData, "bucket", err
	}

	return s.ProcessAssetsInBucket(ctx, explainerParams, assets, payerTable, "bucket", PayerID)
}

func (s *BillingExplainerService) GetBillingExplainerSummaryAndStoreInFS(ctx context.Context, customerID string, billingMonth string, entityID string, isBackfill bool) error {
	log := s.loggerProvider(ctx)

	startOfMonth, endOfMonth, err := utils.GetMonthDateRange(billingMonth)
	if err != nil {
		return err
	}

	log.Infof("Running GetBillingExplainerSummary for customerID %s and billingMonth %s ", customerID, billingMonth)

	invoiceMonth, err := utils.FormatYearMonth(billingMonth)
	if err != nil {
		return err
	}

	var bqProject = "doitintl-cmp-aws-data"

	customerTable := fmt.Sprintf("%s.aws_billing_%s.doitintl_billing_export_v1_%s_%s", bqProject, customerID, customerID, billingMonth)
	explainerParams := domain.BillingExplainerParams{CustomerID: customerID, StartOfMonth: startOfMonth, EndOfMonth: endOfMonth, InvoiceMonth: invoiceMonth, CustomerTable: customerTable}

	payerAccountInfo, err := s.GetPayerInfoFromCustID(ctx, customerID, startOfMonth)
	if err != nil {
		return err
	}

	if len(payerAccountInfo) == 0 {
		return fmt.Errorf("No payer account info found for customerID %s", customerID)
	}

	bucketMap := make(map[string][]domain.SummaryBQ)

	serviceBucketMap := make(map[string][]domain.ServiceRecord)

	accountBucketMap := make(map[string][]domain.AccountRecord)

	entityBuckets, err := s.bucketsDal.GetBuckets(ctx, entityID)
	if err != nil {
		return err
	}

	bucketAssetsMap, err := s.CreateBucketAssetsMap(ctx, entityBuckets)
	if err != nil {
		return err
	}

	if len(entityBuckets) > 0 {
		// Get default bucket
		entity, err := s.entityDal.GetEntity(ctx, entityID)
		if err != nil {
			return err
		}

		explainerParams.InvoicingMode = entity.Invoicing.Mode

		if entity.Invoicing.Default != nil {
			defaultBucketID := entity.Invoicing.Default.ID
			for _, bucket := range entityBuckets {
				if bucket.Ref.ID == defaultBucketID {
					explainerParams.DefaultBucket = bucket.Name
					break
				}
			}
		}
	}

	for _, payerInfo := range payerAccountInfo {
		var resellerNumber string

		log.Infof("friendlyname for customerID %s and payerID %s  %s ", customerID, payerInfo.PayerID, payerInfo.FriendlyName)

		part := strings.Split(payerInfo.FriendlyName, "#")

		if len(part) > 1 {
			resellerNumber = part[1]
		}

		log.Infof("Reseller number for customerID  PayerID %v  %v %v ", customerID, payerInfo.PayerID, resellerNumber)

		payerTable := fmt.Sprintf("doitintl-cmp-aws-data.payer_accounts.payer_account_doit_reseller_account_n%s_%s", resellerNumber, payerInfo.PayerID)

		var data []domain.SummaryBQ

		var serviceData []domain.ServiceRecord

		var accountData []domain.AccountRecord

		var bucketName string

		if len(entityBuckets) > 0 {
			// Assets spread across buckets
			err = s.ProcessAssets(ctx, explainerParams, bucketAssetsMap, payerTable, bucketMap, serviceBucketMap, accountBucketMap, payerInfo.PayerID)
			if err != nil {
				return err
			}
		} else {
			// Assets in just single bucket
			data, serviceData, accountData, bucketName, err = s.ProcessAssetsForEntity(ctx, explainerParams, entityID, payerTable, payerInfo.PayerID)
			if err != nil {
				return err
			}
		}

		if _, ok := bucketMap[bucketName]; ok {
			bucketMap[bucketName] = append(bucketMap[bucketName], data...)
			serviceBucketMap[bucketName] = append(serviceBucketMap[bucketName], serviceData...)
			accountBucketMap[bucketName] = append(accountBucketMap[bucketName], accountData...)
		} else {
			bucketMap[bucketName] = data
			serviceBucketMap[bucketName] = serviceData
			accountBucketMap[bucketName] = accountData
		}
	}

	for bucket, results := range bucketMap {
		if len(results) > 0 {
			err = s.firestoreDal.UpdateEntityFirestoreDoc(ctx, isBackfill, invoiceMonth, entityID, explainerParams.InvoicingMode, results, bucket, serviceBucketMap[bucket], accountBucketMap[bucket])
		}
	}

	if err != nil {
		log.Errorf("Fail to update explainer summary in billing entity doc for entityUD %s and billingMonth %s", entityID, invoiceMonth)
		return err
	}

	return err
}
