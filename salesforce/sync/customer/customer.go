package customer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/pkg"
	assetsDAL "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	contractDAL "github.com/doitintl/hello/scheduled-tasks/contract/dal"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	entitytDAL "github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/firebase/tenant"
	rdsDAL "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/dal"
	sagemakerDAL "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	salesforceClient "github.com/doitintl/hello/scheduled-tasks/salesforce/sync/client"
	tiersDal "github.com/doitintl/tiers/dal"
)

type Service struct {
	logger        *logger.Logging
	connection    *connection.Connection
	integrations  fsdal.Integrations
	contracts     contractDAL.ContractFirestore
	tenantService *tenant.TenantService
	tiers         tiersDal.TierEntitlementsIface
	assets        assetsDAL.Assets
	entities      entitytDAL.Entites
	sagemakerDAL  sagemakerDAL.FlexsaveSagemakerFirestore
	rdsDAL        rdsDAL.Service
}

func getAssetLabel(assetDoc map[string]interface{}) string {
	assetType := assetDoc["type"].(string)
	props := assetDoc["properties"].(map[string]interface{})

	switch assetType {
	case common.Assets.GSuite:
		return fmt.Sprintf("%s (%s)", props["subscription"].(map[string]interface{})["skuName"], props["customerDomain"])

	case common.Assets.GoogleCloud:
		return fmt.Sprintf("%s (%s)", props["displayName"], props["billingAccountId"])

	case common.Assets.GoogleCloudProject:
		return fmt.Sprintf("%s (%s)", props["projectId"], props["billingAccountId"])

	case common.Assets.AmazonWebServices:
		return fmt.Sprintf("%s (%s)", props["friendlyName"], props["accountId"])

	case common.Assets.MicrosoftAzure:
		return fmt.Sprintf("%s (%s)", props["subscription"].(map[string]interface{})["displayName"], props["customerDomain"])

	case common.Assets.Office365:
		return fmt.Sprintf("%s (%s)", props["subscription"].(map[string]interface{})["offerName"], props["customerDomain"])

	default:
		return ""
	}
}

func formatDate(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "1901-01-01"
	}

	return value.Format("2006-01-02")
}

func dateToStringOrNil(value *time.Time) *string {
	if value == nil || value.IsZero() {
		return nil
	}

	formatted := value.Format("2006-01-02")

	return &formatted
}

func formatTimestamp(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "1901-01-01T00:00:00Z"
	}

	return value.Format("2006-01-02T15:04:05Z")
}

func joinStrings(value []string) string {
	out := ""

	for index, val := range value {
		out += val

		if index != len(value)-1 {
			out += ";"
		}
	}

	return out
}

func boolToString(value bool) string {
	if value {
		return "true"
	}

	return "false"
}

const separator = ";"

type AssetItem struct {
	AssetID    string `json:"id"`
	AssetType  string `json:"type" firestore:"type"`
	LicenseQty int64  `json:"quantity,omitempty"`
	AssetName  string `json:"name"`
	URL        string `json:"url"`
	CreateTime int64  `json:"createTime"`
}

func (s *Service) getUserRolesAndPermissions(ctx context.Context, user *common.User) (string, string, error) {
	if user.Roles != nil && len(user.Roles) > 0 {
		permissionSnaps, err := s.connection.Firestore(ctx).Collection("permissions").Documents(ctx).GetAll()
		if err != nil {
			return "", "", err
		}

		type permission struct {
			Title string `firestore:"title"`
		}

		permissionMap := make(map[string]string)

		for _, snap := range permissionSnaps {
			var p permission
			if err := snap.DataTo(&p); err != nil {
				return "", "", err
			}

			permissionMap[snap.Ref.ID] = p.Title
		}

		roles := make([]string, 0, len(user.Roles))
		permissionSet := make(map[string]struct{})

		for _, role := range user.Roles {
			roleDocSnap, err := role.Get(ctx)
			if err != nil {
				continue
			}

			var r common.Role
			if err := roleDocSnap.DataTo(&r); err != nil {
				return "", "", err
			}

			roles = append(roles, r.Name)

			for _, p := range r.Permissions {
				permissionName, exists := permissionMap[p.ID]
				if exists {
					permissionSet[permissionName] = struct{}{}
				}
			}
		}

		permissions := make([]string, 0, len(permissionSet))
		for k := range permissionSet {
			permissions = append(permissions, k)
		}

		return strings.Join(roles, separator), strings.Join(permissions, separator), nil
	}

	return "", strings.Join(user.Permissions, separator), nil
}

func NewService(log *logger.Logging, conn *connection.Connection) *Service {
	contracts, err := contractDAL.NewContractFirestore(context.Background(), common.ProjectID)

	if err != nil {
		panic("no contract dal")
	}

	tenantService, err := tenant.NewTenantsService(conn)
	if err != nil {
		panic("no tenant service")
	}

	fs := conn.Firestore(context.Background())

	return &Service{
		log,
		conn,
		fsdal.NewIntegrationsDALWithClient(fs),
		*contracts,
		tenantService,
		tiersDal.NewTierEntitlementsDALWithClient(fs),
		assetsDAL.NewAssetsFirestoreWithClient(conn.Firestore),
		entitytDAL.NewEntitiesFirestoreWithClient(conn.Firestore),
		sagemakerDAL.SagemakerFirestoreDAL(fs),
		rdsDAL.NewService(fs),
	}
}

func (s *Service) SyncCompany(ctx context.Context, customerID string) error {
	log := s.logger.Logger(ctx)

	customerRef := s.connection.Firestore(ctx).Collection("customers").Doc(customerID)

	docSnap, err := customerRef.Get(ctx)
	if err != nil {
		return err
	}

	var customer common.Customer
	if err := docSnap.DataTo(&customer); err != nil {
		return err
	}

	var additionalDomains []string

	for _, domain := range customer.Domains {
		if domain != customer.PrimaryDomain {
			additionalDomains = append(additionalDomains, domain)
		}
	}

	now := time.Now().UTC()
	currentYear, currentMonth, currentDay := now.Date()

	var (
		startDate time.Time
		endDate   time.Time
	)

	startDate = time.Date(currentYear, currentMonth-1, 1, 0, 0, 0, 0, time.UTC)
	endDate = now

	if currentDay < 11 {
		startDate = time.Date(currentYear, currentMonth-2, 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(currentYear, currentMonth-1, 0, 0, 0, 0, 0, time.UTC)
	}

	query := s.connection.Firestore(ctx).Collection("invoices").Where("customer", "==", customerRef).
		Where("CANCELED", "==", false).
		Where("IVDATE", ">=", startDate).
		Where("IVDATE", "<=", endDate)

	fullInvoicesDocSnaps, err := query.OrderBy("IVDATE", firestore.Desc).Select("USDTOTAL", "PRODUCTS", "IVDATE").Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	gcpRevenue := 0.0
	awsRevenue := 0.0
	o365Revenue := 0.0
	azureRevenue := 0.0
	gSuiteRevenue := 0.0
	solveRevenue := 0.0
	navigatorRevenue := 0.0

	for _, fullInvoiceDocSnap := range fullInvoicesDocSnaps {
		var invoice invoices.FullInvoice
		if err := fullInvoiceDocSnap.DataTo(&invoice); err != nil {
			return err
		}

		products := invoice.Products

		for _, product := range products {
			switch product {
			case common.Assets.GoogleCloud:
				gcpRevenue += invoice.USDTotal
			case common.Assets.AmazonWebServices:
				awsRevenue += invoice.USDTotal
			case common.Assets.Office365:
				o365Revenue += invoice.USDTotal
			case common.Assets.MicrosoftAzure, common.Assets.MicrosoftAzureReseller, common.Assets.MicrosoftAzureStandalone:
				azureRevenue += invoice.USDTotal
			case common.Assets.GSuite:
				gSuiteRevenue += invoice.USDTotal
			case common.Assets.DoiTNavigator:
				navigatorRevenue += invoice.USDTotal
			case common.Assets.DoiTSolve:
				solveRevenue += invoice.USDTotal
			}
		}
	}

	formattedMonth := fmt.Sprintf("%d_%d", currentMonth, currentYear)

	computeSavings := 0.0

	computeCache, err := s.integrations.GetFlexsaveConfigurationCustomer(ctx, customerID)
	if err == nil {
		if computeCache.AWS.SavingsHistory != nil {
			month := computeCache.AWS.SavingsSummary.CurrentMonth.Month
			if formattedMonth == month {
				monthSavings, ok := computeCache.AWS.SavingsHistory[month]
				if ok {
					computeSavings = monthSavings.Savings
				}
			}
		}
	} else {
		if err != fsdal.ErrNotFound {
			return err
		}
	}

	sagemakerSavings := 0.0

	sagemakerCache, err := s.sagemakerDAL.Get(ctx, customerID)
	if err == nil {
		month := sagemakerCache.SavingsSummary.CurrentMonth
		if formattedMonth == month {
			monthSavings, ok := sagemakerCache.SavingsHistory[month]
			if ok {
				sagemakerSavings = monthSavings.Savings
			}
		}
	} else {
		if err != fsdal.ErrNotFound {
			return err
		}
	}

	rdsSavings := 0.0

	rdsCache, err := s.rdsDAL.Get(ctx, customerID)
	if err == nil {
		month := rdsCache.SavingsSummary.CurrentMonth

		if formattedMonth == month {
			monthSavings, ok := rdsCache.SavingsHistory[month]
			if ok {
				rdsSavings = monthSavings.Savings
			}
		}
	} else {
		if err != fsdal.ErrNotFound {
			return err
		}
	}

	isSolveTrial := false

	var solveFreeTrialExpirationDate *time.Time

	var solveFreeTrialStartDate *time.Time

	solveTier := ""

	match := customer.Tiers["solve"]

	if match != nil {
		if match.IsTrialActive(time.Now()) {
			isSolveTrial = true
		}

		solveFreeTrialStartDate = match.TrialStartDate

		solveFreeTrialExpirationDate = match.TrialEndDate

		if match.Tier != nil {
			tierDoc, err := s.connection.Firestore(ctx).Collection("tiers").Doc(match.Tier.ID).Get(ctx)
			if err != nil {
				return err
			}

			if tierDoc.Exists() {
				solveTier = tierDoc.Data()["displayName"].(string)
			}
		}
	}

	isNavigatorTrial := false

	var navigatorFreeTrialExpirationDate *time.Time

	var navigatorFreeTrialStartDate *time.Time

	navigatorTier := ""

	match = customer.Tiers["navigator"]

	if match != nil {
		if match.IsTrialActive(time.Now()) {
			isNavigatorTrial = true
		}

		navigatorFreeTrialStartDate = match.TrialStartDate

		navigatorFreeTrialExpirationDate = match.TrialEndDate

		if match.Tier != nil {
			tierDoc, err := s.connection.Firestore(ctx).Collection("tiers").Doc(match.Tier.ID).Get(ctx)
			if err != nil {
				return err
			}

			if tierDoc.Exists() {
				navigatorTier = tierDoc.Data()["displayName"].(string)
			}
		}
	}

	doitAccountMangers, err := common.GetCustomerAccountManagers(ctx, &customer, common.AccountManagerCompanyDoit)
	if err != nil {
		return err
	}

	aeEmail := ""
	amEmail := ""

	for _, m := range doitAccountMangers {
		if m.Role == common.AccountManagerRoleFSR {
			// we need just one. Will take whatever is last.
			aeEmail = m.Email
		} else if m.Role == common.AccountManagerRoleSAM {
			amEmail = m.Email
		}
	}

	contracts, err := s.contracts.GetActiveContractsForCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	advantage := ""

	advantageSet := make(map[string]struct{})

	for _, doc := range contracts {
		var contract pkg.Contract
		if err := doc.DataTo(&contract); err != nil {
			return err
		}

		switch contract.Type {
		case "amazon-web-services":
			advantageSet["AWS"] = struct{}{}
		case "google-cloud":
			advantageSet["GCP"] = struct{}{}
		case "microsoft-azure":
			advantageSet["Azure"] = struct{}{}
		}
	}

	if len(advantageSet) == 0 {
		advantage = "N/A"
	} else {
		advantages := make([]string, 0, len(advantageSet))

		for key := range advantageSet {
			advantages = append(advantages, key)
		}

		advantage = strings.Join(advantages, ", ")
	}

	payload := salesforceClient.Company{
		Name:                             customer.Name,
		AeEmail:                          aeEmail,
		AmEmail:                          amEmail,
		AdditionalDomains:                joinStrings(additionalDomains),
		Advantage:                        advantage,
		AWSRevenue:                       awsRevenue,
		AzureRevenue:                     azureRevenue,
		Classification:                   customer.Classification,
		ConsoleLink:                      "https://console.doit.com/customers/" + customerID,
		FlexsaveSavings:                  sagemakerSavings + rdsSavings + computeSavings,
		GCPRevenue:                       gcpRevenue,
		GSuiteRevenue:                    gSuiteRevenue,
		IsSolveFreeTrial:                 boolToString(isSolveTrial),
		NavigatorFreeTrialExpirationDate: dateToStringOrNil(navigatorFreeTrialExpirationDate),
		NavigatorFreeTrialStartDate:      dateToStringOrNil(navigatorFreeTrialStartDate),
		NavigatorFreeTrial:               boolToString(isNavigatorTrial),
		NavigatorRevenue:                 navigatorRevenue,
		NavigatorTier:                    navigatorTier,
		O365Revenue:                      o365Revenue,
		PrimaryDomain:                    customer.PrimaryDomain,
		SolveFreeTrialExpirationDate:     dateToStringOrNil(solveFreeTrialExpirationDate),
		SolveFreeTrialStartDate:          dateToStringOrNil(solveFreeTrialStartDate),
		SolveRevenue:                     solveRevenue,
		SolveTier:                        solveTier,
	}

	client, err := salesforceClient.NewClient()
	if err != nil {
		return err
	}

	err = client.UpsertCompany(ctx, customerID, payload)
	if err != nil {
		log.Errorf("company create error: %s", err.Error())

		return err
	}

	fmt.Print("c")

	auth, err := s.tenantService.GetTenantAuthClientByCustomer(ctx, customerID)
	if err != nil && !errors.Is(err, fsdal.ErrNotFound) {
		return err
	}

	docSnaps, err := s.connection.Firestore(ctx).Collection("users").
		WherePath([]string{"customer", "ref"}, "==", customerRef).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, docSnap := range docSnaps {
		var user common.User
		if err := docSnap.DataTo(&user); err != nil {
			continue
		}

		user.ID = docSnap.Ref.ID

		var createdAt time.Time

		var lastLogin time.Time

		if auth != nil {
			userRecord, _ := auth.GetUserByEmail(ctx, user.Email)

			if userRecord != nil && userRecord.UserMetadata != nil {
				createdAt = common.EpochMillisecondsToTime(userRecord.UserMetadata.CreationTimestamp)
				lastLogin = common.EpochMillisecondsToTime(userRecord.UserMetadata.LastLogInTimestamp)
			}
		}

		roles, permissions, err := s.getUserRolesAndPermissions(ctx, &user)
		if err != nil {
			return err
		}

		payload := salesforceClient.User{
			ConsoleCompanyID:   customerID,
			ConsoleFirstLogin:  formatTimestamp(&createdAt),
			ConsoleLastLogin:   formatTimestamp(&lastLogin),
			ConsolePermissions: permissions,
			ConsoleRoles:       roles,
			Email:              user.Email,
			FirstName:          user.FirstName,
			LastName:           user.LastName,
		}

		if err := client.UpsertUser(ctx, user.ID, payload); err != nil {
			log.Errorf("user create error: %s", err.Error())
		} else {
			fmt.Print("a")
		}
	}

	docSnaps, err = s.contracts.GetActiveContractsForCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	for _, docSnap := range docSnaps {
		var contract pkg.Contract
		if err := docSnap.DataTo(&contract); err != nil {
			return err
		}

		contract.ID = docSnap.Ref.ID

		tierName := ""
		tierPackageType := ""
		tierPrice := 0.0
		tierDescription := ""

		if contract.Tier != nil {
			tier, err := s.tiers.GetTier(ctx, contract.Tier.ID)
			if err != nil {
				return err
			}

			tierName = tier.Name

			tierPackageType = tier.PackageType

			price, ok := tier.Price["USD"]
			if ok {
				tierPrice = price
			}

			tierDescription = tier.Description
		}

		contractAssets := ""

		for _, assetRef := range contract.Assets {
			snap, err := s.connection.Firestore(ctx).Collection("assets").Doc(assetRef.ID).Get(ctx)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					continue
				}

				return err
			}

			var assetDoc = make(map[string]interface{})

			if err := snap.DataTo(&assetDoc); err != nil {
				return err
			}

			label := getAssetLabel(assetDoc)
			if len(label) > 0 {
				if len(contractAssets) > 0 {
					contractAssets = contractAssets + ", " + label
				} else {
					contractAssets = label
				}
			}
		}

		entityName := ""

		if contract.Entity != nil && contract.Entity.ID != "" {
			entity, err := s.entities.GetEntity(ctx, contract.Entity.ID)
			if err != nil {
				log.Error("unable to get entity", err)
			}

			if entity != nil {
				entityName = entity.Name
			}
		}

		commitmentPeriodsValue := []pkg.ContractCommitmentPeriod{}

		if contract.CommitmentPeriods != nil {
			commitmentPeriodsValue = contract.CommitmentPeriods
		}

		commitmentPeriods, err := json.Marshal(commitmentPeriodsValue)
		if err != nil {
			return err
		}

		properties, err := json.Marshal(contract.Properties)
		if err != nil {
			return err
		}

		typeValue := "No commit"

		if contract.IsCommitment {
			typeValue = "Hard commit"
		} else if contract.IsSoftCommitment {
			typeValue = "Soft commit"
		}

		payload := salesforceClient.Contract{
			Properties:         string(properties),
			ConsoleCompanyID:   customerID,
			Name:               tierDescription,
			Assets:             contractAssets,
			ChargePerTerm:      contract.ChargePerTerm,
			CommitmentMonths:   contract.CommitmentMonths,
			CommitmentPeriod:   string(commitmentPeriods),
			CommitmentRollover: contract.CommitmentRollover,
			Customer:           customerID,
			Discount:           contract.Discount,
			DiscountEndDate:    formatDate(contract.DiscountEndDate),
			DisplayName:        tierDescription,
			EndDate:            formatDate(contract.EndDate),
			Entity:             entityName,
			EstimatedValue:     contract.EstimatedValue,
			IsCommitment:       contract.IsCommitment,
			IsRenewal:          contract.IsRenewal,
			Notes:              contract.Notes,
			PackageType:        tierPackageType,
			PaymentTerm:        contract.PaymentTerm,
			Price:              tierPrice,
			ProductType:        contract.Type,
			PurchaseOrder:      contract.PurchaseOrder,
			StartDate:          formatDate(contract.StartDate),
			Tier:               tierName,
			Timestamp:          formatTimestamp(&contract.Timestamp),
			Type:               contract.Type,
			ContractType:       typeValue,
		}

		if err := client.UpsertContract(ctx, contract.ID, payload); err != nil {
			log.Errorf("contract create error: %s", err.Error())
		} else {
			fmt.Print("r")
		}
	}

	return nil
}
