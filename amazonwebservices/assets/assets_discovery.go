package assets

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/organizations"
	"google.golang.org/api/iterator"

	doitFirestore "github.com/doitintl/firestore"
	batchIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/access"
	amazonwebservicesDomain "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

const batchSet = "batch.Set err: %v"

const (
	manualAssetDiscoverySuperBETMPA = "798642997822"
	manualAssetDiscoveryValue       = "manual"

	assetDiscoveryQuery = `WITH customer_data AS (
	SELECT
	project_id as account_id,
  FROM
	  doitintl-cmp-aws-data.payer_accounts.payer_account_doit_reseller_account_n1586_798642997822
  WHERE
	DATE(export_time) = @current_date
	AND cost_type != "Tax"
	AND NOT LOWER(description) LIKE "%regenerated invoice%"
  GROUP BY
	  1
  ),
  assets_data AS (
	SELECT
	  SPLIT(__key__.name, '-')[OFFSET(3)] as account_id
	FROM me-doit-intl-com.analytics.assets
	WHERE
	customer.name = "5IeRlmY8jo9CwcuDTXUq"
  ),
  filter_data AS (
	SELECT
	  A.account_id as data_id,
	  B.account_id as assets_id
	FROM
	  customer_data A
	LEFT JOIN
	  assets_data B
	USING(account_id)
  )

  SELECT * FROM filter_data WHERE assets_id IS NULL`
)

type mpaData struct {
	customer           *common.Customer
	flexsaveAccountIDs []string
	assetType          string
	masterPayerAccount *amazonwebservicesDomain.MasterPayerAccount
	customerRef        *firestore.DocumentRef
	entityRef          *firestore.DocumentRef
}

type assetData struct {
	assetSettings    *pkg.AWSAssetSettings
	asset            *amazonwebservices.Asset
	assetSettingsRef *firestore.DocumentRef
}

func (s *AWSAssetsService) UpdateAssetsAllMPA(ctx context.Context) error {
	fs := s.conn.Firestore(ctx)
	logger := s.loggerProvider(ctx)

	masterPayerAccounts, err := s.mpaDAL.GetMasterPayerAccounts(ctx, fs)
	if err != nil {
		return fmt.Errorf("failed to fetch master payer accounts with error: %s", err)
	}

	// Create cloud tasks
	for _, mpa := range masterPayerAccounts.Accounts {
		if mpa.Status != amazonwebservicesDomain.MasterPayerAccountStatusActive {
			continue
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf("/tasks/assets/%s/dedicated/%s", common.Assets.AmazonWebServices, mpa.AccountNumber),
			Queue:  common.TaskQueueAssetsAWS,
		}
		conf := config.Config(nil)

		if _, err = s.cloudTaskClient.CreateTask(ctx, conf); err != nil {
			logger.Errorf("failed to create aws asset discovery task for dedicated payer %s with error: %s", mpa.AccountNumber, err)
			continue
		}
	}

	return nil
}

func (s *AWSAssetsService) UpdateAssetsMPA(ctx context.Context, mpaID string) error {
	logger := s.loggerProvider(ctx)
	assetType := common.Assets.AmazonWebServices
	customerType := "dedicated"

	masterPayerAccount, err := s.mpaDAL.GetMasterPayerAccount(ctx, mpaID)

	if err != nil {
		return fmt.Errorf("failed to fetch master payer accounts with error: %s", err)
	}

	if masterPayerAccount.IsSharedPayer() {
		return nil
	}

	customer, err := s.customersDAL.GetCustomer(ctx, *masterPayerAccount.CustomerID)

	if err != nil {
		logger.Errorf("failed to get customer with error: %s", err)
		return nil
	}

	svc, err := s.getAwsOrganization(ctx, masterPayerAccount)

	if err != nil {
		logger.Errorf("failed to get aws organization with error: %s", err)
		return nil
	}

	return s.updateAssets(ctx, masterPayerAccount, customer, svc, assetType, customerType)
}

// Update manual assets for Superbet MPA
func (s *AWSAssetsService) UpdateManualAssetsMPA(ctx context.Context, mpaID string) error {
	logger := s.loggerProvider(ctx)
	assetType := common.Assets.AmazonWebServices
	currentDate := time.Now().Format("2006-01-02")

	masterPayerAccount, err := s.mpaDAL.GetMasterPayerAccount(ctx, mpaID)
	if err != nil {
		return fmt.Errorf("failed to fetch master payer accounts with error: %s", err)
	}

	if masterPayerAccount.IsSharedPayer() {
		return nil
	}

	customer, err := s.customersDAL.GetCustomer(ctx, *masterPayerAccount.CustomerID)
	if err != nil {
		logger.Errorf("failed to get customer with error: %s", err)
		return nil
	}

	svc, err := s.getAwsOrganization(ctx, masterPayerAccount)
	if err != nil {
		logger.Errorf("failed to get aws organization with error: %s", err)
		return nil
	}

	return s.updateManualAssets(ctx, masterPayerAccount, customer, svc, assetType, currentDate)
}

func (s *AWSAssetsService) UpdateStandaloneAssets(ctx context.Context, customerID, accountID string) error {
	l := s.loggerProvider(ctx)
	assetsType := "standalone"

	session, err := s.awsAccessService.GetAWSSession(accountID, access.FunctionListAccountsPages)
	if err != nil {
		l.Errorf("failed to create aws session for customer %s; %s", err)
		return err
	}

	svc := organizations.New(session)
	assetType := common.Assets.AmazonWebServicesStandalone

	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		l.Errorf("failed to get customer; %s", err)
		return err
	}

	mpa := &amazonwebservicesDomain.MasterPayerAccount{
		AccountNumber: accountID,
		CustomerID:    &customerID,
		FriendlyName:  s.getStandalonePayerAccountName(accountID),
	}

	return s.updateAssets(ctx, mpa, customer, svc, assetType, assetsType)
}

func (s *AWSAssetsService) getStandalonePayerAccountName(ID string) string {
	return fmt.Sprintf("standalone-payer-%s", ID)
}

func (s *AWSAssetsService) updateAssets(
	ctx context.Context,
	masterPayerAccount *amazonwebservicesDomain.MasterPayerAccount,
	customer *common.Customer,
	svc *organizations.Organizations,
	assetType string,
	assetsType string,
) error {
	fs := s.conn.Firestore(ctx)

	logger := s.loggerProvider(ctx)

	flexsaveAccountIDs, err := s.flexsaveAPI.ListFlexsaveAccountsWithCache(ctx, time.Minute*30)
	if err != nil {
		return err
	}

	batch := doitFirestore.NewBatchProviderWithClient(fs, 100).Provide(ctx)

	customerRef := customer.Snapshot.Ref

	logger.Infof("Asset Discovery - %v - processing:", assetsType)

	logger.Infof("- customer: %v", customerRef.ID)

	entityRef := s.getEntityRef(ctx, customer)
	if entityRef != nil {
		logger.Infof("- entity: %v", entityRef.ID)
	}

	logger.Infof("- MPA: %v", masterPayerAccount.AccountNumber)
	// hold fetched account IDs
	fetchedAccountIDs := make([]string, 0)

	if err := svc.ListAccountsPages(&organizations.ListAccountsInput{},
		func(page *organizations.ListAccountsOutput, lastPage bool) bool {
			mpaData := &mpaData{
				customer,
				flexsaveAccountIDs,
				assetType,
				masterPayerAccount,
				customerRef,
				entityRef,
			}
			// collect account IDs from AWS SDK
			for _, acc := range page.Accounts {
				fetchedAccountIDs = append(fetchedAccountIDs, *acc.Id)
			}
			// Build assets doc and assetSettings doc
			// The documents are rewritten every time.
			s.buildAccountsAssets(ctx, batch, page.Accounts, mpaData)
			if err := batch.Commit(ctx); err != nil {
				logger.Errorf("batch.Commit err: %v", err)
			}

			return !lastPage
		},
	); err != nil {
		logger.Error(err)
		return err
	}

	logger.Infof("Asset Discovery - %v - fetched accounts: %v", assetsType, fetchedAccountIDs)

	return nil
}

// Update assets for superbet
func (s *AWSAssetsService) updateManualAssets(
	ctx context.Context,
	masterPayerAccount *amazonwebservicesDomain.MasterPayerAccount,
	customer *common.Customer,
	svc *organizations.Organizations,
	assetType string,
	currentDate string,
) error {
	var accounts []string
	var addedAccounts []string
	totalAdded := 0

	fs := s.conn.Firestore(ctx)
	bq := s.conn.Bigquery(ctx)

	logger := s.loggerProvider(ctx)
	logger.Infof("Manual asset discovery for Superbet MPA, current date: %v", currentDate)

	flexsaveAccountIDs, err := s.flexsaveAPI.ListFlexsaveAccountsWithCache(ctx, time.Minute*30)
	if err != nil {
		logger.Error(err)
		return err
	}

	batch := doitFirestore.NewBatchProviderWithClient(fs, 100).Provide(ctx)
	// Customer reference document
	customerRef := customer.Snapshot.Ref
	// Entity reference document
	entityRef := s.getEntityRef(ctx, customer)
	// MPA document struct
	mpaData := &mpaData{
		customer:           customer,
		flexsaveAccountIDs: flexsaveAccountIDs,
		assetType:          assetType,
		masterPayerAccount: masterPayerAccount,
		customerRef:        customerRef,
		entityRef:          entityRef,
	}

	query := bq.Query(assetDiscoveryQuery)

	// Set query parameters
	query.Parameters = []bigquery.QueryParameter{
		{
			Name:  "current_date",
			Value: currentDate,
		},
	}
	// Query
	iter, err := query.Read(ctx)
	if err != nil {
		return err
	}

	for {
		var row []bigquery.Value

		err := iter.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return err
		}

		if accountID, ok := row[0].(string); ok {
			accounts = append(accounts, accountID)
		}
	}

	logger.Info("Asset Discovery - manual - processing:")
	logger.Infof("- customer: %v", customerRef.ID)
	logger.Infof("- MPA: %v", masterPayerAccount.AccountNumber)
	logger.Infof("Asset Discovery - manual - fetched accounts: %v", accounts)

	for _, accountID := range accounts {
		if slice.Contains(flexsaveAccountIDs, accountID) {
			continue
		}
		// assets and assetSettings document ID
		docID := fmt.Sprintf("%s-%s", assetType, accountID)
		accountOrg := &amazonwebservices.Account{
			ID:   accountID,
			Name: accountID,
			PayerAccount: &amazonwebservicesDomain.PayerAccount{
				AccountID:   masterPayerAccount.AccountNumber,
				DisplayName: masterPayerAccount.FriendlyName,
			},
		}
		assetRef := s.assetsDAL.GetRef(ctx, docID)
		assetSettingsRef := s.assetSettingsDAL.GetRef(ctx, docID)
		paths := []firestore.FieldPath{[]string{"type"}, []string{"properties"}, []string{"customer"}, []string{"discovery"}}
		hasSauronRole := false

		assetSettings, err := s.assetSettingsDAL.GetAWSAssetSettings(ctx, docID)
		if err != nil && err != doitFirestore.ErrNotFound {
			logger.Errorf("getAssetSetting err: %v", err)
			continue
		}
		// Create asset settings if not found
		if assetSettings == nil {
			assetSettings = &pkg.AWSAssetSettings{
				BaseAsset: pkg.BaseAsset{
					AssetType: mpaData.assetType,
				},
			}
		}

		var supportSettings *pkg.AWSSettingsSupport

		if assetSettings.Settings != nil {
			supportSettings = &assetSettings.Settings.Support
		}

		props := &pkg.AWSProperties{
			AccountID:    accountID,
			Name:         accountID,
			FriendlyName: accountID,
			SauronRole:   hasSauronRole,
			OrganizationInfo: &pkg.OrganizationInfo{
				PayerAccount: accountOrg.PayerAccount,
				Status:       accountOrg.Status,
				Email:        accountOrg.Email,
			},
			Support: supportSettings,
		}

		asset := amazonwebservices.Asset{
			AssetType:  assetType,
			Customer:   customerRef,
			Properties: props,
			Discovery:  manualAssetDiscoveryValue,
		}

		assetData := &assetData{
			assetSettings,
			&asset,
			assetSettingsRef,
		}
		// Build assetSettings doc and assets doc
		// Set assetSettings doc
		if err := s.buildPaths(ctx, batch, mpaData, assetData, &paths); err != nil {
			logger.Errorf("buildpaths", err)
			continue
		}

		// Set assets doc with the updates values from assetData.asset
		if err := batch.Set(ctx, assetRef, asset, firestore.Merge(paths...)); err != nil {
			logger.Errorf(batchSet, err)
			continue
		}

		if err := batch.Set(ctx, assetRef.Collection("assetMetadata").Doc("metadata"), map[string]interface{}{
			"lastUpdated": firestore.ServerTimestamp,
			"type":        asset.AssetType,
		}); err != nil {
			logger.Errorf(batchSet, err)
			continue
		} else {
			addedAccounts = append(addedAccounts, docID)
			totalAdded++
		}
	}

	if err := batch.Commit(ctx); err != nil {
		logger.Errorf("batch.Commit err: %v", err)
		return err
	}
	// Log the assets that were created
	if addedAccounts != nil {
		logger.Infof("Asset Discovery - manual - new assets created: %v", addedAccounts)
	}
	logger.Infof("Asset Discovery - manual - total new assets created: %v, total Flexsave accounts skipped: %v", totalAdded, len(accounts)-totalAdded)

	return nil
}

func (s *AWSAssetsService) buildAccountsAssets(ctx context.Context, batch batchIface.Batch, accounts []*organizations.Account, mpaData *mpaData) {
	logger := s.loggerProvider(ctx)

	for _, account := range accounts {
		if slice.Contains(mpaData.flexsaveAccountIDs, *account.Id) {
			continue
		}
		// Create document ID
		docID := fmt.Sprintf("%s-%s", mpaData.assetType, *account.Id)
		// Get account organization
		accountOrg := s.getAccount(account, mpaData.masterPayerAccount)
		// Get assets reference
		assetRef := s.assetsDAL.GetRef(ctx, docID)
		// Get assetSettings reference
		assetSettingsRef := s.assetSettingsDAL.GetRef(ctx, docID)
		// Paths to update
		paths := []firestore.FieldPath{[]string{"type"}, []string{"properties"}, []string{"customer"}}
		hasSauronRole := amazonwebservices.GetSauronRole(ctx, accountOrg, mpaData.customerRef)

		// Get assetSettings document
		assetSettings, err := s.assetSettingsDAL.GetAWSAssetSettings(ctx, docID)
		if err != nil && err != doitFirestore.ErrNotFound {
			logger.Errorf("getAssetSetting err: %v", err)
			continue
		}

		// Create assetSettings doc if not found
		if assetSettings == nil {
			assetSettings = &pkg.AWSAssetSettings{
				BaseAsset: pkg.BaseAsset{
					AssetType: mpaData.assetType,
				},
			}
		}

		var supportSettings *pkg.AWSSettingsSupport

		// Create support settings if not found
		if assetSettings.Settings != nil {
			supportSettings = &assetSettings.Settings.Support
		}

		// Build asset doc
		props := s.getAssetProperties(account, accountOrg, hasSauronRole, supportSettings)
		asset := amazonwebservices.Asset{
			AssetType:  mpaData.assetType,
			Customer:   mpaData.customerRef,
			Properties: props,
		}
		// Asset data struct with assetSettings and assets doc
		assetData := &assetData{
			assetSettings,
			&asset,
			assetSettingsRef,
		}
		// update assets doc data with assetSettings doc data (if exist)
		// and build and overwrite assetSettings document.
		if err := s.buildPaths(ctx, batch, mpaData, assetData, &paths); err != nil {
			continue
		}
		// set/overwrite assets doc
		if err := batch.Set(ctx, assetRef, asset, firestore.Merge(paths...)); err != nil {
			logger.Errorf(batchSet, err)
			continue
		}

		if err := batch.Set(ctx, assetRef.Collection("assetMetadata").Doc("metadata"), map[string]interface{}{
			"lastUpdated": firestore.ServerTimestamp,
			"type":        asset.AssetType,
		}); err != nil {
			logger.Errorf(batchSet, err)
			continue
		}
	}
}

func (s *AWSAssetsService) buildPaths(ctx context.Context, batch batchIface.Batch, mpaData *mpaData, assetData *assetData, paths *[]firestore.FieldPath) error {
	logger := s.loggerProvider(ctx)

	if mpaData.customerRef.ID != fb.Orphan.ID {
		if err := s.buildPathsCustomer(ctx, batch, mpaData, assetData, paths); err != nil {
			logger.Errorf("buildPathsCustomer err: %v", err)
			return err
		}
	} else {
		// Could not find customer, update settings to reference orphan customer and reset entity, contract
		if err := s.buildPathsNoCustomer(ctx, batch, mpaData, assetData, paths); err != nil {
			logger.Errorf("buildPathsNoCustomer err: %v", err)
			return err
		}
	}

	return nil
}

func (s *AWSAssetsService) getEntityRef(ctx context.Context, customer *common.Customer) *firestore.DocumentRef {
	logger := s.loggerProvider(ctx)

	var oneActiveEntity bool

	var entityRef *firestore.DocumentRef

	for _, entity := range customer.Entities {
		entity, err := s.entitiesDAL.GetEntity(ctx, entity.ID)
		if err != nil {
			logger.Errorf("failed to get entity with error: %s", err)
			continue
		}

		if entity.Active {
			if oneActiveEntity {
				entityRef = nil
				break
			}

			entityRef = entity.Snapshot.Ref
			oneActiveEntity = true
		}
	}

	return entityRef
}

func (s *AWSAssetsService) getAccount(account *organizations.Account, masterPayerAccount *amazonwebservicesDomain.MasterPayerAccount) *amazonwebservices.Account {
	return &amazonwebservices.Account{
		ID:              *account.Id,
		Name:            *account.Name,
		Status:          *account.Status,
		Arn:             *account.Arn,
		Email:           *account.Email,
		JoinedMethod:    *account.JoinedMethod,
		JoinedTimestamp: *account.JoinedTimestamp,
		PayerAccount: &amazonwebservicesDomain.PayerAccount{
			AccountID:   masterPayerAccount.AccountNumber,
			DisplayName: masterPayerAccount.FriendlyName,
		},
	}
}

func (s *AWSAssetsService) getAssetProperties(account *organizations.Account, accountOrg *amazonwebservices.Account, hasSauronRole bool, supportSettings *pkg.AWSSettingsSupport) *pkg.AWSProperties {
	return &pkg.AWSProperties{
		AccountID:    *account.Id,
		Name:         *account.Name,
		FriendlyName: *account.Name,
		SauronRole:   hasSauronRole,
		OrganizationInfo: &pkg.OrganizationInfo{
			PayerAccount: accountOrg.PayerAccount,
			Status:       accountOrg.Status,
			Email:        accountOrg.Email,
		},
		Support: supportSettings,
	}
}

// TODO make this a method in a new service related to Amazon
func (s *AWSAssetsService) getAwsOrganization(ctx context.Context, masterPayerAccount *amazonwebservicesDomain.MasterPayerAccount) (*organizations.Organizations, error) {
	creds, err := masterPayerAccount.NewCredentials("")
	if err != nil {
		return nil, err
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(endpoints.UsEast1RegionID),
		Credentials: creds,
	})
	if err != nil {
		return nil, err
	}

	return organizations.New(sess), nil
}

// Build assets doc and
// Build and set assetSettings documents.
// The documents are rewritten every time.
func (s *AWSAssetsService) buildPathsCustomer(ctx context.Context, batch batchIface.Batch, mpaData *mpaData, assetData *assetData, paths *[]firestore.FieldPath) error {
	fs := s.conn.Firestore(ctx)

	var bucketRef *firestore.DocumentRef

	assetSettingsUpdate := map[string]interface{}{}
	// If assetSettings has customer and entity, update the entity and bucket
	// Otherwise, update the customer, entity and asset type with mpaData
	if assetData.assetSettings.Customer != nil && assetData.assetSettings.Customer.ID == mpaData.customerRef.ID && assetData.assetSettings.Entity != nil {
		mpaData.entityRef = assetData.assetSettings.Entity
		bucketRef = assetData.assetSettings.Bucket
	} else {
		assetSettingsUpdate["type"] = mpaData.assetType
		assetSettingsUpdate["customer"] = mpaData.customerRef
		assetSettingsUpdate["entity"] = mpaData.entityRef
		assetSettingsUpdate["bucket"] = bucketRef
		assetSettingsUpdate["contract"] = nil

		if assetData.assetSettings != nil && assetData.assetSettings.TimeCreated.IsZero() {
			assetSettingsUpdate["timeCreated"] = time.Now()
		}
	}
	// Update asset settings with the new values
	assetData.asset.Tags = assetData.assetSettings.Tags
	// Entity will be updated with the entity reference of assetSettings if the doc exists.
	assetData.asset.Entity = mpaData.entityRef
	assetData.asset.Bucket = bucketRef

	*paths = append(*paths, []string{"entity"}, []string{"bucket"}, []string{"tags"})

	if contractRef, update := common.GetAssetContract(ctx, fs, assetData.asset, mpaData.customerRef, mpaData.entityRef, nil); update {
		assetSettingsUpdate["contract"] = contractRef
		assetData.asset.Contract = contractRef

		*paths = append(*paths, []string{"contract"})
	}

	if len(assetSettingsUpdate) > 0 {
		// Set assetSettings doc with the new values
		if err := batch.Set(ctx, assetData.assetSettingsRef, assetSettingsUpdate, firestore.MergeAll); err != nil {
			return err
		}
	}

	return nil
}

func (s *AWSAssetsService) buildPathsNoCustomer(ctx context.Context, batch batchIface.Batch, mpaData *mpaData, assetData *assetData, paths *[]firestore.FieldPath) error {
	if err := batch.Set(ctx, assetData.assetSettingsRef, map[string]interface{}{
		"customer": mpaData.customerRef,
		"entity":   nil,
		"contract": nil,
		"bucket":   nil,
		"tags":     firestore.Delete,
		"type":     mpaData.assetType,
	}, firestore.MergeAll); err != nil {
		return fmt.Errorf(batchSet, err)
	}

	assetData.asset.Entity = nil
	assetData.asset.Contract = nil
	assetData.asset.Bucket = nil

	*paths = append(*paths, []string{"entity"}, []string{"contract"}, []string{"bucket"})

	return nil
}

func (s *AWSAssetsService) ClearAllFlexsaveAssets(ctx context.Context) error {
	flexsaveAccountIDs, err := s.flexsaveAPI.ListFlexsaveAccountsWithCache(ctx, time.Minute*30)
	if err != nil {
		return err
	}

	flexsaveAccountMap := make(map[string]bool)

	for _, item := range flexsaveAccountIDs {
		flexsaveAccountMap[item] = true
	}

	assetsList, err := s.assetsDAL.ListAWSAssets(ctx, common.Assets.AmazonWebServices)
	if err != nil {
		return err
	}

	var accountIDList []string

	for _, asset := range assetsList {
		accountID := asset.Properties.AccountID

		if _, exists := flexsaveAccountMap[accountID]; exists {
			collectionID := common.Assets.AmazonWebServices + "-" + accountID

			accountIDList = append(accountIDList, collectionID)
		}
	}

	err = s.assetsDAL.DeleteAssets(ctx, accountIDList)
	if err != nil {
		return err
	}

	return nil
}

func (s *AWSAssetsService) GetAssetFromAccountNumber(ctx context.Context, accountNumber string) (*pkg.AWSAsset, error) {
	return s.assetsDAL.GetAWSAssetFromAccountNumber(ctx, accountNumber)
}
