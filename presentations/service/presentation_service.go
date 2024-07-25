package service

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/pkg"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	optimizerService "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service"
	optimizerIface "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service/iface"
	awsTablemgmt "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/services/tablemanagement"
	gcpBillingTableMgmt "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/service"
	gcpBillingTableMgmtIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/service/iface"
	metadataService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service"
	metadataDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/iface"
	azureTablemgmt "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure/tablemanagement"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customersDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/publicdashboards"
	entitiesDal "github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"
	awsstandalone "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/aws"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/presentations/domain"
	"github.com/doitintl/hello/scheduled-tasks/presentations/log"
	tierDal "github.com/doitintl/tiers/dal"
)

type PresentationService struct {
	Logger                logger.Provider
	conn                  *connection.Connection
	customersDAL          customersDal.CustomersFirestore
	entitiesDAL           entitiesDal.EntitiesFirestore
	tierDAL               tierDal.TierEntitlementsDAL
	accountManagersDAL    fsdal.AccountManagers
	integrationsDAL       fsdal.Integrations
	assetsDAL             assetsDal.AssetsFirestore
	assetSettingsDAL      assetsDal.AssetSettings
	metadataDAL           metadataDal.MetadataIface
	flexsaveAPI           flexapi.FlexAPI
	optimizerPresentation optimizerIface.OptimizerPresentation
	AWSBillingTables      awsTablemgmt.IBillingTableManagementService
	AzureBillingTables    azureTablemgmt.BillingTableManagementService
	GCPBillingTables      gcpBillingTableMgmtIface.BillingTableManagementService
	awsStandaloneService  *awsstandalone.AwsStandaloneService
}

var AbbreviationDictionary = map[string]string{
	"amazon-web-services": "AWS",
	"google-cloud":        "GCP",
	"microsoft-azure":     "Azure",
}

var (
	sharedDriveFolder      = "1JUb__V1JeQZJD6xTjfYJ9BNhHjvOCJVe"
	priorityID             = "000001"
	presentationCustomerID = "presentationcustomerAWSAzureGCP"
)

const FetchCustomerErr = "failed to retrieve presentation customers: %w"

const presentationModeTier = "GEMAgAsdpIlJRN2BcIhK"

func NewPresentationService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
) (*PresentationService, error) {
	flexAPIService, err := flexapi.NewFlexAPIService()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	optimizerPresentation := optimizerService.NewOptimizer(ctx, loggerProvider, conn)

	customerDAL := customersDal.NewCustomersFirestoreWithClient(conn.Firestore)

	awsBillingTables, err := awsTablemgmt.NewBillingTableManagementService(loggerProvider, conn, customerDAL)
	if err != nil {
		panic(err)
	}

	azureBillingTables, err := azureTablemgmt.NewBillingTableManagementService(loggerProvider, conn, customerDAL)
	if err != nil {
		panic(err)
	}

	gcpBillingTables := gcpBillingTableMgmt.NewBillingTableManagementService(loggerProvider, conn)

	awsStandaloneService, err := awsstandalone.NewAwsStandaloneService(loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	return &PresentationService{
		loggerProvider,
		conn,
		*customersDal.NewCustomersFirestoreWithClient(conn.Firestore),
		*entitiesDal.NewEntitiesFirestoreWithClient(conn.Firestore),
		*tierDal.NewTierEntitlementsDALWithClient(conn.Firestore(ctx)),
		fsdal.NewAccountManagersDALWithClient(conn.Firestore(context.Background())),
		fsdal.NewIntegrationsDALWithClient(conn.Firestore(context.Background())),
		*assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
		assetsDal.NewAssetSettingsFirestoreWithClient(conn.Firestore),
		metadataService.NewMetadataService(ctx, loggerProvider, conn),
		flexAPIService,
		optimizerPresentation,
		awsBillingTables,
		*azureBillingTables,
		gcpBillingTables,
		awsStandaloneService,
	}, nil
}

func (p *PresentationService) ChangePresentationMode(ctx context.Context, customerID string) error {
	p.Logger(ctx).Info("ChangePresentationMode")

	customerRef := p.customersDAL.GetRef(ctx, customerID)

	customerSnap, err := customerRef.Get(ctx)
	if err != nil {
		return err
	}

	var customer common.Customer
	if err := customerSnap.DataTo(&customer); err != nil {
		return err
	}

	presentationModeProps := customer.PresentationMode
	if presentationModeProps == nil {
		presentationModeProps = &common.PresentationMode{
			IsPredefined: false,
			Enabled:      false,
			CustomerID:   presentationCustomerID,
		}
	}

	presentationModeProps.Enabled = !presentationModeProps.Enabled

	_, err = customerRef.Set(ctx, map[string]interface{}{
		"presentationMode": presentationModeProps,
	}, firestore.MergeAll)
	if err != nil {
		return err
	}

	if presentationModeProps.Enabled {
		t, err := p.tierDAL.GetTierRefByName(ctx, tierDal.PresentationTierName, pkg.NavigatorPackageTierType)
		if err != nil {
			return err
		}

		if err := p.tierDAL.UpdateCustomerTier(ctx, customerRef, pkg.NavigatorPackageTierType, &pkg.CustomerTier{
			Tier: t,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (p *PresentationService) CreateCustomer(ctx *gin.Context) (*common.Customer, error) {
	p.Logger(ctx).Info("createCustomer")
	clouds, err := p.getClouds(ctx)
	if err != nil {
		return nil, web.NewRequestError(err, http.StatusBadRequest)
	}

	customerID := p.generateCustomerID(clouds)

	var customer common.Customer
	entities, err := p.GetDemoEntities(ctx, &customer, customerID)
	if err != nil {
		return nil, err
	}

	presentationModeProps := common.PresentationMode{
		IsPredefined: true,
	}

	sharedDriveFolderID := sharedDriveFolder

	customer = common.Customer{
		Name:                customerID,
		LowerName:           strings.ToLower(customerID),
		PrimaryDomain:       fmt.Sprintf("%s.com", customerID),
		Domains:             []string{fmt.Sprintf("%s.com", customerID)},
		Assets:              clouds,
		AccountManager:      nil,
		AccountManagers:     nil,
		AccountTeam:         nil,
		Entities:            entities,
		SharedDriveFolderID: &sharedDriveFolderID,
		Classification:      common.CustomerClassificationBusiness,
		TimeCreated:         time.Now().UTC(),
		Auth: common.Auth{
			Sso: &common.CustomerAuthSso{
				OIDC: nil,
				SAML: nil,
			},
		},
		PresentationMode: &presentationModeProps,
	}

	demoCustomerRef := p.customersDAL.GetRef(ctx, customerID)

	_, err = demoCustomerRef.Set(ctx, customer)
	if err != nil {
		return nil, err
	}

	accountConfiguration := map[string]interface{}{
		"isRecalculated": true,
	}

	_, err = demoCustomerRef.Collection("accountConfiguration").Doc(common.Assets.AmazonWebServices).Set(ctx, accountConfiguration)
	if err != nil {
		return nil, err
	}

	customerSnap, err := demoCustomerRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	if err := createRootOrganization(ctx, demoCustomerRef, &customer); err != nil {
		return nil, err
	}

	var newCustomer common.Customer
	if err = customerSnap.DataTo(&newCustomer); err != nil {
		return nil, err
	}

	newCustomer.Snapshot = customerSnap

	return &newCustomer, nil
}

func (p *PresentationService) generateCustomerID(inputArray []string) string {
	var outputElements []string

	for _, element := range inputArray {
		element = strings.ToLower(element)
		abbreviation, exists := AbbreviationDictionary[element]

		if !exists {
			abbreviation = element
		}

		outputElements = append(outputElements, abbreviation)
	}

	sort.Strings(outputElements)

	outputString := "presentationcustomer" + strings.Join(outputElements, "")

	return outputString
}

func (p *PresentationService) getClouds(ctx *gin.Context) ([]string, error) {
	var presentationReq domain.CreateCustomerReq
	if err := ctx.ShouldBindJSON(&presentationReq); err != nil {
		return nil, web.NewRequestError(err, http.StatusBadRequest)
	}

	sort.Strings(presentationReq.Cloud)

	return presentationReq.Cloud, nil
}

func (p *PresentationService) getEntityData(ctx context.Context, customerID string, country string, currency fixer.Currency) common.Entity {
	email := "foo@baz.com"
	name := "Art Vandelay"
	currencyStr := string(currency)
	rc := common.EntityContact{
		Email: &email,
		Name:  &name,
	}

	return common.Entity{
		Active:   true,
		Contact:  &rc,
		Country:  &country,
		Currency: &currencyStr,
		Customer: p.customersDAL.GetRef(ctx, customerID),
	}
}

func (p *PresentationService) createEntity(ctx context.Context, entity *common.Entity) (*firestore.DocumentRef, error) {
	docRef, _, err := p.entitiesDAL.GetEntitiesCollectionRef(ctx).Add(ctx, entity)
	if err != nil {
		return nil, err
	}

	return docRef, nil
}

func (p *PresentationService) GetDemoEntities(ctx context.Context, customer *common.Customer, customerID string) ([]*firestore.DocumentRef, error) {
	p.Logger(ctx).Info("GetDemoEntities")

	docSnaps, err := p.entitiesDAL.GetEntitiesCollectionRef(ctx).Where("customer", "==", p.customersDAL.GetRef(ctx, customerID)).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	if len(docSnaps) > 0 {
		var existingEntityRefs []*firestore.DocumentRef
		for _, snap := range docSnaps {
			existingEntityRefs = append(existingEntityRefs, snap.Ref)
		}

		return existingEntityRefs, nil
	}

	usEntity := p.getEntityData(ctx, customerID, "United States", fixer.USD)
	usEntity.Name = customer.Name + " US"
	usEntity.LowerName = customer.LowerName + " US"
	usEntity.PriorityCompany = "doitint"
	usEntity.PriorityID = "US" + priorityID
	usEntity.Payment = &common.EntityPayment{
		Type: "bill_com",
	}

	usEntityRef, err := p.createEntity(ctx, &usEntity)
	if err != nil {
		return nil, err
	}

	ukEntity := p.getEntityData(ctx, customerID, "United Kingdom", fixer.GBP)
	ukEntity.Name = customer.Name + " UK"
	ukEntity.LowerName = customer.LowerName + " UK"
	ukEntity.PriorityCompany = "doituk"
	ukEntity.PriorityID = "UK" + priorityID
	ukEntity.Payment = &common.EntityPayment{
		Type: "credit_card",
	}

	ukEntityRef, err := p.createEntity(ctx, &ukEntity)
	if err != nil {
		return nil, err
	}

	return []*firestore.DocumentRef{usEntityRef, ukEntityRef}, nil
}

func createRootOrganization(ctx context.Context, customerRef *firestore.DocumentRef, demoCustomer *common.Customer) error {
	var dashboards []string
	for _, d := range publicdashboards.DashboardsToAttach {
		dashboards = append(dashboards, d.DashboardID)
	}

	rootOrg := common.Organization{
		Customer:              customerRef,
		Name:                  organizations.RootOrgID,
		Description:           fmt.Sprintf("Root organiation for %s", demoCustomer.PrimaryDomain),
		Scope:                 []*firestore.DocumentRef{},
		TimeCreated:           time.Now().UTC(),
		TimeModified:          time.Now().UTC(),
		Dashboards:            dashboards,
		AllowCustomDashboards: true,
	}

	_, err := customerRef.Collection("customerOrgs").Doc(organizations.RootOrgID).Set(ctx, rootOrg)
	if err != nil {
		return err
	}

	return nil
}

// update customer assets, metadata
func (p *PresentationService) UpdateAssetsMetadataGCP(ctx *gin.Context) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "metadata")

	customers, err := p.customersDAL.GetPresentationCustomersWithAssetType(ctx, common.Assets.GoogleCloud)

	if err != nil {
		return fmt.Errorf(FetchCustomerErr, err)
	}

	for _, customer := range customers {
		customerID := customer.Ref.ID

		billingAccountID := domain.HashCustomerIdIntoABillingAccountId(customerID)

		if err := p.UpdateGCPAssets(ctx, customerID); err != nil {
			return fmt.Errorf("failed to update gcp assets for presentation customer %s: %v", customerID, err)
		}

		assetID := strings.Join([]string{common.Assets.GoogleCloud, billingAccountID}, "-")

		l.Infof("GCP metadata update for customer: %s", customerID)

		if err := p.metadataDAL.UpdateGCPBillingAccountMetadata(ctx, assetID, billingAccountID, nil); err != nil {
			return fmt.Errorf("failed to update gcp metadata for billing account %s: %v", billingAccountID, err)
		}
	}

	return nil
}

// update customer assets, metadata
func (p *PresentationService) UpdateAssetsMetadataAWS(ctx *gin.Context) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "metadata")

	docSnaps, err := p.customersDAL.GetPresentationCustomersWithAssetType(ctx, common.Assets.AmazonWebServices)
	if err != nil {
		return fmt.Errorf(FetchCustomerErr, err)
	}

	for _, docSnap := range docSnaps {
		customerID := docSnap.Ref.ID

		if err := p.UpdateAWSAssets(ctx, customerID); err != nil {
			return fmt.Errorf("failed to update aws assets for presentation customer %s: %v", customerID, err)
		}

		l.Infof("AWS metadata update for customer: %s", customerID)

		if err := p.metadataDAL.UpdateAWSCustomerMetadata(ctx, customerID, nil); err != nil {
			return fmt.Errorf("failed to update aws metadata for billing account %s: %v", awsDemoBillingAccountID, err)
		}
	}

	return nil
}

func (p *PresentationService) AggregateBillingDataGCP(ctx *gin.Context) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "aggregation")

	if errors := p.runForEachPresentationCustomerWithAssetType(ctx, common.Assets.AmazonWebServices, func(ctx *gin.Context, customerID string) error {
		l.Infof("GCP data aggregation for customer: %s", customerID)
		billingAccountID := domain.HashCustomerIdIntoABillingAccountId(customerID)

		if errors := p.GCPBillingTables.UpdateAllAggregatedTables(ctx, billingAccountID, "", 0, true); len(errors) > 0 {
			return fmt.Errorf("failed to aggregate GCP billing data for customer %s: %v", customerID, errors)
		}
		return nil
	}); len(errors) > 0 {
		err := fmt.Errorf("GCP data aggregation failed with: %v", errors)
		l.Error(err)

		return err
	}

	return nil
}

func (p *PresentationService) AggregateBillingDataAWS(ctx *gin.Context) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "aggregation")

	if errors := p.runForEachPresentationCustomerWithAssetType(ctx, common.Assets.AmazonWebServices, func(ctx *gin.Context, customerID string) error {
		l.Infof("AWS data aggregation for customer: %s", customerID)
		if errors := p.AWSBillingTables.UpdateAllAggregatedTables(ctx, customerID, true); len(errors) > 0 {
			return fmt.Errorf("failed to aggregate AWS billing data for customer %s: %v", customerID, errors)
		}
		return nil
	}); len(errors) > 0 {
		err := fmt.Errorf("AWS data aggregation failed with: %v", errors)
		l.Error(err)

		return err
	}

	l.Infof("AWS data aggregation completed")

	return nil
}

func (p *PresentationService) AggregateBillingDataAzure(ctx *gin.Context) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "aggregation")

	if errors := p.runForEachPresentationCustomerWithAssetType(ctx, common.Assets.MicrosoftAzure, func(ctx *gin.Context, customerID string) error {
		l.Infof("Azure data aggregation for customer: %s", customerID)

		errors := p.AzureBillingTables.UpdateAllAggregatedTables(ctx, customerID, true)
		if len(errors) > 0 {
			return fmt.Errorf("failed to aggregate Azure billing data for customer %s: %v", customerID, errors)
		}
		return nil
	}); len(errors) > 0 {
		err := fmt.Errorf("Azure data aggregation failed with: %v", errors)
		l.Error(err)

		return err
	}

	l.Infof("Azure data aggregation completed")

	return nil
}

func (p *PresentationService) UpdateAssetsMetadataAzure(ctx *gin.Context) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "metadata")

	docSnaps, err := p.customersDAL.GetPresentationCustomersWithAssetType(ctx, common.Assets.MicrosoftAzure)
	if err != nil {
		return fmt.Errorf(FetchCustomerErr, err)
	}

	for _, docSnap := range docSnaps {
		customerID := docSnap.Ref.ID

		if err := p.UpdateAzureAssets(ctx, customerID); err != nil {
			return fmt.Errorf("failed to update Azure assets for presentation customer %s: %v", customerID, err)
		}

		l.Infof("Azure metadata update for customer: %s", customerID)

		if err := p.metadataDAL.UpdateAzureCustomerMetadata(ctx, customerID); err != nil {
			return fmt.Errorf("failed to update Azure metadata for presentation customer %s: %v", customerID, err)
		}
	}

	return nil
}

func (p *PresentationService) GetPresentationCustomersAssetTypes(ctx *gin.Context) (map[string]bool, error) {
	assetTypes := make(map[string]bool)

	customers, err := p.customersDAL.GetPresentationCustomers(ctx)
	if err != nil {
		return nil, fmt.Errorf(FetchCustomerErr, err)
	}

	for _, customer := range customers {

		for _, assetType := range customer.Assets {
			assetTypes[assetType] = true
		}
	}

	return assetTypes, nil
}
