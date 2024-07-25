package flexsaveresold

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	mpaDAL "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	domainAmazonwebservices "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	assetspkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

const recommendationsConcurrency = 15

type instanceSavingsGenerator interface {
	getInstanceSavingsByCustomer(ctx context.Context, pricingParams types.Recommendation, customerID string, resultChannel chan<- types.RecommendationsResultChannel, pos int)
}

func (s *Service) GetRecommendations(ctx context.Context, customerID string) ([]types.Recommendation, error) {
	fs := s.Firestore(ctx)
	log := s.Logger(ctx)

	customerRef := fs.Collection("customers").Doc(customerID)

	chtInstanceSnaps, err := fs.Collection("integrations").
		Doc("cloudhealth").
		Collection("cloudhealthInstances").
		Where("customer", "==", customerRef).
		Where("disabled", "==", false).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var potential types.PotentialResponse

	if len(chtInstanceSnaps) == 0 {
		return nil, fmt.Errorf("customer: %v is invalid CHT customer", customerID)
	}

	if len(chtInstanceSnaps) != 1 {
		return nil, fmt.Errorf("expected exactly one document but got %v for customer %v", len(chtInstanceSnaps), customerID)
	} else {
		err = chtInstanceSnaps[0].DataTo(&potential)
		if err != nil {
			return nil, err
		}
	}

	linkedAccountMappings, err := getLinkedToPayerMapping(ctx, fs, customerRef)
	if err != nil {
		return nil, err
	}

	savingsInput := make([]types.Recommendation, 0)

	for _, rec := range potential.Data {
		instanceFamily, instanceSize := getInstanceSizeAndFamily(rec.InstanceType)
		if instanceFamily == "" || instanceSize == "" {
			continue
		}

		reformatted := types.Recommendation{
			Region:          rec.Region.Value,
			InstanceFamily:  instanceFamily,
			PayerAccountID:  linkedAccountMappings[rec.Account],
			OperatingSystem: rec.OperatingSystem.Value,
			LinkedAccountID: rec.Account,
			InstanceSize:    instanceSize,
			NumInstances:    rec.NumInstances,
		}

		savingsInput = append(savingsInput, reformatted)
	}

	savingsInputChunks := makeChunks(savingsInput, recommendationsConcurrency)

	var recommendationsResults []types.Recommendation

	var lastError error

	log.Infof("executing %v chunks for customer %v", len(savingsInputChunks), customerID)

	for _, eachSavingsInput := range savingsInputChunks {
		recommendationsResultChunk, lastErrorForChunk := s.generateRecommendations(s, ctx, customerID, eachSavingsInput)
		recommendationsResults = append(recommendationsResults, recommendationsResultChunk...)
		lastError = lastErrorForChunk
	}

	if len(recommendationsResults) == 0 && lastError != nil {
		return nil, lastError
	}

	return recommendationsResults, nil
}

func (s *Service) generateRecommendations(isg instanceSavingsGenerator, ctx context.Context, customerID string, savingsInput []types.Recommendation) ([]types.Recommendation, error) {
	log := s.Logger(ctx)

	resultChannel := make(chan types.RecommendationsResultChannel, 1)
	for i := 0; i < len(savingsInput); i++ {
		go isg.getInstanceSavingsByCustomer(ctx, savingsInput[i], customerID, resultChannel, i)
	}

	var recommendationsResults []types.Recommendation

	var lastError error

	for i := 0; i < len(savingsInput); i++ {
		res := <-resultChannel
		if err := res.Errors; err != nil {
			log.Error(err)
			lastError = err

			continue
		}

		if warning := res.Warning; warning != nil {
			log.Warning(warning)
			continue
		}

		savingsInput[res.Pos].Savings = res.Savings
		savingsInput[res.Pos].OnDemand = res.OnDemand
		savingsInput[res.Pos].PriceAfterDiscount = res.PriceAfterDiscount
		recommendationsResults = append(recommendationsResults, savingsInput[res.Pos])
	}

	return recommendationsResults, lastError
}

func makeChunks(recommendations []types.Recommendation, size int) (recommendationChunks [][]types.Recommendation) {
	recommendationChunks = [][]types.Recommendation{}

	for size < len(recommendations) {
		recommendationChunks = append(recommendationChunks, recommendations[0:size])
		recommendations = recommendations[size:]
	}

	if len(recommendations) > 0 {
		recommendationChunks = append(recommendationChunks, recommendations)
	}

	return recommendationChunks
}

func getInstanceSizeAndFamily(instanceType string) (string, string) {
	return strings.Split(instanceType, ".")[0], strings.Split(instanceType, ".")[1]
}

func getLinkedToPayerMapping(ctx context.Context, fs *firestore.Client, customerRef *firestore.DocumentRef) (map[string]string, error) {
	mappings := make(map[string]string)

	docs, err := fs.Collection("assets").Where("customer", "==", customerRef).Documents(ctx).GetAll()
	if err != nil {
		return mappings, err
	}

	for _, docSnap := range docs {
		var asset assetspkg.AWSAsset
		if err := docSnap.DataTo(&asset); err != nil {
			continue
		}

		if asset.Properties != nil && asset.Properties.OrganizationInfo != nil && asset.Properties.OrganizationInfo.PayerAccount != nil {
			payer := asset.Properties.OrganizationInfo.PayerAccount.AccountID
			linked := asset.Properties.AccountID
			mappings[linked] = payer
		}
	}

	return mappings, nil
}

func (s *Service) CreateFlexsaveOrders(ctx context.Context) error {
	fs := s.Firestore(ctx)
	log := s.Logger(ctx)

	flexsaveConfigSnaps, err := fs.Collection("integrations").Doc("flexsave").Collection("configuration").Where("AWS.enabled", "==", true).Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	log.Infof("Creating Flexsave Autopilot orders for %v customers.", len(flexsaveConfigSnaps))

	successfulCustomers := 0

	for _, docSnap := range flexsaveConfigSnaps {
		customerID := docSnap.Ref.ID

		customerRef := fs.Collection("customers").Doc(customerID)

		customer, err := common.GetCustomer(ctx, customerRef)
		if err != nil {
			log.Errorf("could not get customer doc '%v' - skipping. error: %v", customerID, err)
			continue
		}

		hasSharedPayerAssets, err := s.assets.HasSharedPayerAWSAssets(ctx, customerRef)
		if err != nil {
			log.Errorf("could not determine shared payer status for customer '%s' - skipping. error: %v", customerID, err)
			continue
		}

		if !hasSharedPayerAssets {
			continue
		}

		var data pkg.FlexsaveConfiguration
		if err := docSnap.DataTo(&data); err != nil {
			log.Errorf("failed parsing cache document for customer %v - err: %v", customerID, err)
			continue
		}
		// this reasonCantEnable is only applicable to dedicated payers so not relevant here
		if data.AWS.ReasonCantEnable == ErrLowSpend.Error() {
			continue
		}

		if data.AWS.ReasonCantEnable != "" {
			log.Warningf("failed autopilot validation for customer %v - reason can not enable is: %v", customerID, data.AWS.ReasonCantEnable)
			continue
		}

		if err := s.createFlexsaveOrdersForCustomer(ctx, customer, 0); err != nil {
			log.Errorf("could not generate flexsave orders for customer %v - skipping. error: %v", customerID, err)
			continue
		}

		successfulCustomers = successfulCustomers + 1
	}

	log.Infof("Generated orders for %v customers. Failed for %v customers", successfulCustomers, len(flexsaveConfigSnaps)-successfulCustomers)

	return nil
}

func (s *Service) createFlexsaveOrdersForCustomer(ctx context.Context, customer *common.Customer, offset int) error {
	log := s.Logger(ctx)

	recommendations, err := s.GetRecommendations(ctx, customer.Snapshot.Ref.ID)
	if err != nil {
		return err
	}

	successfulOrders := 0

	for _, recommendation := range recommendations {
		if err := s.createFlexsaveOrder(ctx, recommendation, customer, offset); err != nil {
			log.Errorf("couldn't create order for customer %v based on %v. err: %v", customer.Snapshot.Ref.ID, recommendation, err)
			continue
		}

		successfulOrders = successfulOrders + 1
	}

	log.Infof("Generated %v orders for customer %v. Failed %v orders", successfulOrders, customer.Snapshot.Ref.ID, len(recommendations)-successfulOrders)

	return nil
}

func (s *Service) createFlexsaveOrder(ctx context.Context, recommendation types.Recommendation, customer *common.Customer, offset int) error {
	log := s.Logger(ctx)
	fs := s.Firestore(ctx)

	payerAccounts, err := mpaDAL.GetMasterPayerAccountsByPayerIDs(ctx, fs, recommendation.PayerAccountID)
	if err != nil {
		return err
	}

	if !payerAccounts.IsFlexsaveAllowed(recommendation.PayerAccountID) {
		return NewServiceError("payer account doesn't support flexsave", web.ErrBadRequest)
	}

	if !payerAccounts.IsValidRegion(recommendation.PayerAccountID, recommendation.Region) {
		if slice.Contains(domainAmazonwebservices.GovRegions, recommendation.Region) {
			log.Infof("gov cloud regions are not supported for shared payers, region recommended %v for payer account %v", recommendation.Region, recommendation.PayerAccountID)
		} else {
			return NewServiceError(fmt.Sprintf("incorrect region %v for payer account %v", recommendation.Region, recommendation.PayerAccountID), web.ErrBadRequest)
		}
	}

	instanceType := fmt.Sprintf("%v.%v", recommendation.InstanceFamily, recommendation.InstanceSize)
	tenancy := "default"
	numInstances := int64(recommendation.NumInstances)
	today := time.Now().UTC()
	startDate := time.Date(today.Year(), today.Month()+time.Month(offset), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(today.Year(), today.Month()+time.Month(offset+1), 1, 0, 0, 0, 0, time.UTC).Add(time.Millisecond * -1)

	config := FlexRIOrderConfig{
		Region:          &recommendation.Region,
		InstanceType:    &instanceType,
		OperatingSystem: &recommendation.OperatingSystem,
		InstanceFamily:  &recommendation.InstanceFamily,
		Tenancy:         &tenancy,
		NumInstances:    &numInstances,
		PayerAccountID:  &recommendation.PayerAccountID,
		AccountID:       &recommendation.LinkedAccountID,
		StartDate:       &startDate,
		EndDate:         &endDate,
		Note:            "Created by Flexsave Autopilot",
		AutoRenew:       nil,
	}

	chtCustomerID, err := s.getChtCustomerID(ctx, *config.AccountID)
	if err != nil {
		return err
	}

	sizeFlexible := false
	if isSizeFlexible(&config) {
		sizeFlexible = true
	}

	config.SizeFlexible = &sizeFlexible

	order := FlexRIOrder{
		Customer:           customer.Snapshot.Ref,
		Entity:             nil,
		Status:             OrderStatusPending,
		Email:              "",
		UID:                "",
		Config:             config,
		ClientID:           chtCustomerID,
		InvoiceAdjustments: FlexRIOrderInvoiceAdjustments{},
		NormalizedUnits:    nil,
		Pricing:            nil,
		Execution:          OrderExecAutopilot,
		Utilization:        map[string]float64{},
		Metadata: map[string]interface{}{
			"customer": map[string]interface{}{
				"primaryDomain": customer.PrimaryDomain,
				"name":          customer.Name,
			},
		},
	}

	if err := commitOrder(ctx, fs, order); err != nil {
		return err
	}

	return nil
}

func (s *Service) getChtCustomerID(ctx context.Context, accountID string) (int64, error) {
	fs := s.Firestore(ctx)

	assetRef := fs.Collection("assets").Doc(fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, accountID))

	assetDocSnap, err := assetRef.Get(ctx)
	if err != nil {
		return 0, err
	}

	var asset amazonwebservices.Asset
	if err := assetDocSnap.DataTo(&asset); err != nil {
		return 0, err
	}

	if asset.GetCloudHealthCustomerID() == 0 {
		docSnaps, err := fs.Collection("integrations").
			Doc("cloudhealth").
			Collection("cloudhealthInstances").
			Where("customer", "==", asset.Customer).
			Documents(ctx).GetAll()
		if err != nil {
			return 0, err
		}

		if len(docSnaps) == 0 {
			return 0, NewServiceError("no cloudhealth customer id found for account", web.ErrBadRequest)
		}

		if len(docSnaps) > 1 {
			return 0, NewServiceError("more than one cloudhealth doc found for customer", web.ErrBadRequest)
		}

		ID, err := strconv.ParseInt(docSnaps[0].Ref.ID, 10, 64)
		if err != nil {
			return 0, err
		}

		return ID, nil
	}

	return asset.GetCloudHealthCustomerID(), nil
}
