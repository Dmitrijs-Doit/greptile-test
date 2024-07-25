package cloudanalytics

import (
	"context"
	"fmt"
	"strconv"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/bqutils"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/cspreport"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	errCustomerFmt  = "unable to process customer %s; %s"
	acctMngrMissing = "account manager missing for customer %s"
)

type CspRow struct {
	CustomerID     string                        `json:"customer_id"`
	EntityID       string                        `json:"entity_id"`
	ID             string                        `json:"id"`
	BillingJoinID  string                        `json:"billing_join_id"`
	Type           string                        `json:"type"`
	Classification common.CustomerClassification `json:"classification"`
	PrimaryDomain  string                        `json:"primary_domain"`
	Territory      string                        `json:"territory"`
	PayeeCountry   string                        `json:"payee_country"`
	PayerCountry   string                        `json:"payer_country"`
	FSR            string                        `json:"field_sales_representative"`
	SAM            string                        `json:"strategic_account_manager"`
	TAM            string                        `json:"technical_account_manager"`
	CSM            string                        `json:"customer_success_manager"`
}

type GenericAsset struct {
	AssetType string                 `firestore:"type"`
	Entity    *firestore.DocumentRef `firestore:"entity"`
}

// UpdateCustomersInfoTable creates the CSP customer info metadata table.
func (s *CloudAnalyticsService) UpdateCustomersInfoTable(ctx context.Context) error {
	schema := bigquery.Schema{
		{Name: "billing_join_id", Required: true, Type: bigquery.StringFieldType},
		{Name: "customer_id", Required: true, Type: bigquery.StringFieldType},
		{Name: "entity_id", Required: true, Type: bigquery.StringFieldType},
		{Name: "id", Required: true, Type: bigquery.StringFieldType},
		{Name: "type", Required: true, Type: bigquery.StringFieldType},
		{Name: "primary_domain", Required: true, Type: bigquery.StringFieldType},
		{Name: "classification", Required: true, Type: bigquery.StringFieldType},
		{Name: "payee_country", Required: true, Type: bigquery.StringFieldType},
		{Name: "payer_country", Required: true, Type: bigquery.StringFieldType},
		{Name: "territory", Required: true, Type: bigquery.StringFieldType},
		{Name: "field_sales_representative", Required: true, Type: bigquery.StringFieldType},
		{Name: "strategic_account_manager", Required: true, Type: bigquery.StringFieldType},
		{Name: "technical_account_manager", Required: true, Type: bigquery.StringFieldType},
		{Name: "customer_success_manager", Required: true, Type: bigquery.StringFieldType},
	}

	allCustomersInfo, err := s.GetCustomersInfo(ctx)
	if err != nil {
		return err
	}

	if allCustomersInfo != nil {
		params := bqutils.BigQueryTableLoaderParams{
			Client: s.conn.Bigquery(ctx),
			Schema: &schema,
			Rows:   allCustomersInfo,
			Data: &bqutils.BigQueryTableLoaderRequest{
				DestinationProjectID:   cspreport.GetCSPMetadataProject(),
				DestinationDatasetID:   cspreport.GetCSPMetadataDataset(),
				DestinationTableName:   cspreport.GetCSPMetadataTable(),
				ObjectDir:              "csp_data",
				ConfigJobID:            "csp_data",
				RequirePartitionFilter: false,
				WriteDisposition:       bigquery.WriteTruncate,
				Clustering:             &[]string{"id"},
			},
		}
		if err := bqutils.BigQueryTableLoader(ctx, params); err != nil {
			return err
		}
	}

	return nil
}

func (s *CloudAnalyticsService) GetCustomersInfo(ctx context.Context) ([]interface{}, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	customersDocSnaps, err := fs.Collection("customers").Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var countriesCodeMap common.CountriesInfo

	contriesDocSnaps, err := fs.Collection("app").Doc("countries").Get(ctx)
	if err != nil {
		return nil, err
	}

	if err := contriesDocSnaps.DataTo(&countriesCodeMap); err != nil {
		return nil, err
	}

	managersCache := make(map[string]*common.AccountManager)

	var cspDataRows []interface{}

	recalculatedCustomers := make(map[string]bool)

	for _, customerDocSnap := range customersDocSnaps {
		var customer common.Customer
		if err := customerDocSnap.DataTo(&customer); err != nil {
			l.Errorf(errCustomerFmt, customerDocSnap.Ref.ID, err)
			return nil, err
		}

		customerRef := customerDocSnap.Ref
		customerID := customerRef.ID

		assetTypes := []string{
			common.Assets.GoogleCloud,
			common.Assets.GoogleCloudStandalone,
			common.Assets.AmazonWebServices,
			common.Assets.AmazonWebServicesStandalone,
			common.Assets.MicrosoftAzure,
		}
		assetDocSnaps, err := fs.Collection("assets").
			Where("customer", "==", customerRef).
			Where("type", "in", assetTypes).
			Documents(ctx).GetAll()

		if err != nil {
			l.Errorf(errCustomerFmt, customerDocSnap.Ref.ID, err)
			return nil, err
		}

		if len(assetDocSnaps) == 0 {
			continue
		}

		accountTeam, err := getAccountTeam(ctx, &customer, managersCache)
		if err != nil {
			l.Errorf(acctMngrMissing, customerDocSnap.Ref.ID, err)
		}

		for _, assetDocSnap := range assetDocSnaps {
			var asset GenericAsset
			if err := assetDocSnap.DataTo(&asset); err != nil {
				l.Error(err)
				continue
			}

			assetID := assetDocSnap.Ref.ID[len(asset.AssetType)+1:]

			var (
				entity       common.Entity
				entityID     string
				territory    string
				payeeCountry string
				payerCountry string
			)

			if asset.Entity != nil {
				entityID = asset.Entity.ID

				entityDocSnap, err := asset.Entity.Get(ctx)
				if err != nil {
					l.Error(err)
					continue
				}

				if err := entityDocSnap.DataTo(&entity); err != nil {
					l.Error(err)
					continue
				}

				if entity.Country != nil {
					payerCountry = *entity.Country
				}

				payeeCountry = entity.PayeeCountry()
				if val, ok := countriesCodeMap.Code[payeeCountry]; ok {
					territory = val.Region
				}
			}

			billingJoinID, err := getBillingJoinID(ctx, customerID, asset.AssetType, assetID, customerDocSnap, assetDocSnap, recalculatedCustomers)
			if err != nil {
				l.Error(err)
				continue
			}

			cspDataRows = append(cspDataRows, &CspRow{
				CustomerID:     customerID,
				EntityID:       entityID,
				ID:             assetID,
				BillingJoinID:  billingJoinID,
				Type:           asset.AssetType,
				Classification: customer.Classification,
				PrimaryDomain:  customer.PrimaryDomain,
				Territory:      territory,
				PayeeCountry:   payeeCountry,
				PayerCountry:   payerCountry,
				FSR:            accountTeam["FSR"],
				SAM:            accountTeam["SAM"],
				TAM:            accountTeam["TAM"],
				CSM:            accountTeam["CSM"],
			})
		}
	}

	return cspDataRows, nil
}

func getAccountManager(ctx context.Context, managerRef *firestore.DocumentRef, cache map[string]*common.AccountManager) (*common.AccountManager, error) {
	if am, ok := cache[managerRef.ID]; ok {
		return am, nil
	}

	accountManager, err := common.GetAccountManager(ctx, managerRef)
	if err != nil {
		return nil, err
	}

	cache[managerRef.ID] = accountManager

	return accountManager, nil
}

func getAccountTeam(ctx context.Context, customer *common.Customer, cache map[string]*common.AccountManager) (map[string]string, error) {
	accountTeam := make(map[string]string)

	if len(customer.AccountTeam) == 0 {
		return accountTeam, nil
	}

	for _, am := range customer.AccountTeam {
		if am.Company != common.AccountManagerCompanyDoit {
			continue
		}

		var accountManager *common.AccountManager

		accountManager, err := getAccountManager(ctx, am.Ref, cache)
		if err != nil {
			return accountTeam, err
		}

		// This will pick the last employee for each of the roles a customer has.
		// so if a customer has 2 TAMs we will return the 2nd email
		switch accountManager.Role {
		case common.AccountManagerRoleFSR:
			accountTeam["FSR"] = accountManager.Email
		case common.AccountManagerRoleSAM:
			accountTeam["SAM"] = accountManager.Email
		case common.AccountManagerRoleTAM:
			accountTeam["TAM"] = accountManager.Email
		case common.AccountManagerRoleCSM:
			accountTeam["CSM"] = accountManager.Email
		}
	}

	return accountTeam, nil
}

func getBillingJoinID(
	ctx context.Context,
	customerID string,
	assetType string,
	assetID string,
	customerDocSnap *firestore.DocumentSnapshot,
	assetDocSnap *firestore.DocumentSnapshot,
	recalculatedCustomers map[string]bool,
) (string, error) {
	var billingJoinID string

	switch assetType {
	case common.Assets.AmazonWebServices:
		if _, ok := recalculatedCustomers[customerID]; !ok {
			isRecalculated, err := common.GetCustomerIsRecalculatedFlag(ctx, customerDocSnap.Ref)
			if err != nil {
				return "", fmt.Errorf("failed getting recalculated flag for %s with error %s", customerID, err)
			}

			recalculatedCustomers[customerID] = isRecalculated
		}

		if recalculatedCustomers[customerID] {
			billingJoinID = customerID
			break
		}

		var awsAsset amazonwebservices.Asset
		if err := assetDocSnap.DataTo(&awsAsset); err != nil {
			return "", err
		}

		chtCustomerID := awsAsset.GetCloudHealthCustomerID()
		if chtCustomerID != 0 {
			billingJoinID = strconv.FormatInt(chtCustomerID, 10)
		} else {
			billingJoinID = customerID
		}

	case common.Assets.AmazonWebServicesStandalone:
		billingJoinID = customerID

	case common.Assets.GoogleCloud, common.Assets.GoogleCloudStandalone:
		billingJoinID = assetID

	case common.Assets.MicrosoftAzure:
		billingJoinID = assetID

	default:
		return "", fmt.Errorf("unsupported asset type %s", assetType)
	}

	return billingJoinID, nil
}
