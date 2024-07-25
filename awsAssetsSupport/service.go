package awsAssetsSupport

import (
	"context"
	"sort"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	assetsPkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AwsAssetsSupportService struct {
	*logger.Logging
	conn *connection.Connection
}

type MpaParams struct {
	PayerID      string `json:"payerId"`
	SupportModel string `json:"supportModel"`
}

type AssetSupport struct {
	AssetID          string                 `json:"assetID" firestore:"id"`
	Ref              *firestore.DocumentRef `json:"assetRef" firestore:"ref"`
	AssetSettingsRef *firestore.DocumentRef `json:"assetSettingsRef" firestore:"ref"`
	SupportTier      string                 `json:"SupportTier"`
	SupportModel     string                 `json:"supportModel"`
	IsPLESAsset      bool                   `json:"isPLESAsset"`
	IsOverridable    bool                   `json:"isOverridable"`
}

type QueryResultRow struct {
	AssetID       string `bigquery:"account_id"`
	SupportTier   string `bigquery:"support_tier"`
	SupportModel  string `bigquery:"support_model"`
	IsPLESAsset   string `bigquery:"is_ples_special_case"`
	IsOverridable string `bigquery:"is_overridable"`
}

type AssestSupportOverridenValues struct {
	OverridingSupportTier string     `json:"overridingSupportTie,omitempty" firestore:"overridingSupportTier"`
	OverrideReason        string     `json:"overrideReason,omitempty" firestore:"overrideReason"`
	OverriddenOn          *time.Time `json:"overriddenOn,omitempty" firestore:"overriddenOn"`
}

func NewAWSAssetsSupportService(log *logger.Logging, conn *connection.Connection) (*AwsAssetsSupportService, error) {
	return &AwsAssetsSupportService{
		log,
		conn,
	}, nil
}

func (s *AwsAssetsSupportService) GetAWSMasterPayerAccountsFromFS(ctx context.Context) ([]MpaParams, error) {
	mMpaAccounts := make([]MpaParams, 0)
	fs := s.conn.Firestore(ctx)

	mpasDocSnaps, err := fs.Collection("app").Doc("master-payer-accounts").Collection("mpaAccounts").Documents(ctx).GetAll()
	if err != nil {
		s.Logging.Logger(ctx).Errorf("Failed to get mpasDocSnaps Error: %v", err)
		return nil, err
	}

	for _, mpaSnap := range mpasDocSnaps {
		var mpa domain.MasterPayerAccount
		if err := mpaSnap.DataTo(&mpa); err != nil {
			continue
		}

		if mpa.AccountNumber != "" {
			var mpaData MpaParams
			mpaData.PayerID = mpa.AccountNumber

			if mpa.Support.Model == nil {
				mpaData.SupportModel = "resold"
			} else {
				mpaData.SupportModel = *mpa.Support.Model
			}

			mMpaAccounts = append(mMpaAccounts, mpaData)
		}
	}

	return mMpaAccounts, nil
}

func (s *AwsAssetsSupportService) GetAssetsByPayerID(ctx context.Context, payerID string, supportModel string) ([]AssetSupport, error) {
	fs := s.conn.Firestore(ctx)

	assetSnaps, err := fs.Collection("assets").Where("properties.organization.payerAccount.id", "==", payerID).Documents(ctx).GetAll()
	if err != nil {
		s.Logging.Logger(ctx).Errorf("Failed to get assets docs snaps: %v", err)
		return nil, err
	}

	assets := make([]AssetSupport, 0)

	for _, docSnap := range assetSnaps {
		var asset assetsPkg.AWSAsset
		if err := docSnap.DataTo(&asset); err != nil {
			s.Logging.Logger(ctx).Errorf("Failed to convert docSnap into AWS asset struct, Account ID is %s ", asset.Properties.AccountID)
			continue
		}

		if asset.Properties.AccountID != "" && docSnap.Ref != nil {
			assetAccountID := asset.Properties.AccountID
			// building the structured object
			var assetSupportData AssetSupport
			assetSupportData.AssetID = assetAccountID
			assetSupportData.Ref = docSnap.Ref
			assetSupportData.SupportTier = ""
			assetSupportData.SupportModel = supportModel
			assetSupportData.IsPLESAsset = false
			assetSupportData.IsOverridable = true
			assetSupportData.AssetSettingsRef = fs.Collection("assetSettings").Doc(docSnap.Ref.ID)
			assets = append(assets, assetSupportData)
		}
	}

	return assets, nil
}

func (s *AwsAssetsSupportService) GetAssets(ctx context.Context) ([]AssetSupport, error) {
	payerAccounts, err := s.GetAWSMasterPayerAccountsFromFS(ctx)
	if err != nil {
		s.Logging.Logger(ctx).Errorf("Failed to get payerAccounts Error: %v", err)
		return nil, err
	}

	var assetsList []AssetSupport

	for _, payerAccountData := range payerAccounts {
		if payerAccountData.PayerID != "" {
			payerAssetsList, err := s.GetAssetsByPayerID(ctx, payerAccountData.PayerID, payerAccountData.SupportModel)
			if err != nil {
				s.Logging.Logger(ctx).Errorf("Failed to get payerAssetsList: %v", err)
				continue
			}

			assetsList = append(assetsList, payerAssetsList...)
		}
	}

	return assetsList, nil
}

func (s *AwsAssetsSupportService) ConvertResults(ctx context.Context, iter *bigquery.RowIterator) (map[string]map[string]string, error) {
	results := make(map[string]map[string]string, 0)

	for {
		var row QueryResultRow

		err := iter.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			s.Logging.Logger(ctx).Errorf("Failed to iterate over results: %v", err)
			continue
		}

		results[row.AssetID] = map[string]string{}
		results[row.AssetID]["SupportModel"] = row.SupportModel
		results[row.AssetID]["SupportTier"] = row.SupportTier
		results[row.AssetID]["PLES"] = row.IsPLESAsset
		results[row.AssetID]["IsOverridable"] = row.IsOverridable
	}

	return results, nil
}

func (s *AwsAssetsSupportService) GetSupportPerAssetMapFromQuery(ctx context.Context, queryString string) (map[string]map[string]string, error) {
	q := s.conn.Bigquery(ctx).Query(queryString)

	rows, err := q.Read(ctx)
	if err != nil {
		s.Logging.Logger(ctx).Errorf("error reading query: %v", err)
		return nil, err
	}

	results, err := s.ConvertResults(ctx, rows)
	if err != nil {
		s.Logging.Logger(ctx).Errorf("Failed to convert results rows: %v", err)
		return nil, err
	}

	return results, nil
}

func (s *AwsAssetsSupportService) GetAwsAssetsSupportFromBilling(ctx context.Context) (map[string]map[string]string, error) {
	query := `
		WITH src AS (
			SELECT
				insert_timestamp,
				JSON_VALUE(metadata['contract_type']) AS support_model,
				CASE WHEN JSON_VALUE(metadata['original_support_type']) IS NULL THEN JSON_VALUE(metadata['support_type']) ELSE JSON_VALUE(metadata['original_support_type']) END AS support_tier,
				JSON_VALUE(metadata['is_ples_special_case']) AS is_ples_special_case,
				JSON_VALUE(metadata['payer_account_id']) AS payer_account_id,
				metadata
			FROM
				` + `me-doit-intl-com.measurement.data_api_logs` + `
			WHERE
				DATE(insert_timestamp) >= DATE_SUB(DATE(insert_timestamp), INTERVAL 3 DAY)
				AND operation = "cmp.aws_billing.recalculation"
				AND category = "get_support_billing_rows"
				AND context = "support"
				AND sub_context = "metadata_logging"
			),

			latest_session AS (
			SELECT
				ANY_VALUE(JSON_VALUE(metadata['session_id']) HAVING MAX insert_timestamp) AS session_id
			FROM
				src
			GROUP BY
				support_model, support_tier, is_ples_special_case, payer_account_id
		),

		support_values AS (SELECT
			support_model,
			support_tier,
			JSON_VALUE(metadata['related_accounts']) AS related_accounts,
			is_ples_special_case,
			payer_account_id,
		FROM
			src
		JOIN
			latest_session
		ON
		session_id = JSON_VALUE(metadata['session_id'])
		GROUP BY
			1, 2, 3, 4, 5)

		SELECT
			CASE WHEN payer_account_id = '561602220360' THEN 'partner-led' ELSE support_model END AS support_model,
			CASE WHEN payer_account_id = '561602220360' THEN 'business' ELSE support_tier END AS support_tier,
			TRIM(account_id) AS account_id,
			is_ples_special_case,
			CASE WHEN
					-- enterprise resold customers have a contract on which all the accounts must be enterprise
					support_model = 'resold' AND support_tier = 'enterprise' OR

					-- customers on shared payer 1 canot be editable
					payer_account_id = '561602220360' THEN 'false' ELSE 'true'
					END AS is_overridable
		FROM
			support_values, UNNEST(SPLIT(related_accounts, ',')) AS account_id

		UNION ALL

		(WITH accounts AS (
			SELECT
			   payer_id,
			   account_id
			FROM
			` + `doitintl-cmp-aws-data.accounts.accounts_history` + `
			WHERE
			  DATE(timestamp) >= DATE_SUB(CURRENT_DATE(), INTERVAL 31 DAY)
			  -- only shared payers 1, 2, 7 (CHT customers)
			  AND payer_id IN ('017920819041', '279843869311', '561602220360')
			GROUP BY
			  1, 2
		  ),

		  mid AS (SELECT
			CASE WHEN payer_id IN ('017920819041', '279843869311') THEN 'resold' ELSE 'partner-led' END AS support_model,
			CASE
			  WHEN payer_id = '561602220360' THEN 'business'
			  WHEN LOWER(description) LIKE '%basic%' OR LOWER(service_description) LIKE '%basic%' THEN 'basic'
			  WHEN LOWER(description) LIKE '%developer%' OR LOWER(service_description) LIKE '%developer%' THEN 'developer'
			  WHEN LOWER(description) LIKE '%business%' OR LOWER(service_description) LIKE '%business%' THEN 'business'
			  WHEN LOWER(description) LIKE '%enterprise%' OR LOWER(service_description) LIKE '%enterprise%' THEN 'enterprise'
			  WHEN LOWER(description) LIKE '%business%' AND service_id = 'OCBPremiumSupport' THEN 'business'
			  WHEN LOWER(description) LIKE '%enterprise%' AND service_id = 'OCBPremiumSupport' THEN 'enterprise'
			  END AS support_tier,
			project_id,
			'false' AS is_ples_special_case,
			MAX(export_time) AS max_export_time
		  FROM
		  ` + `doitintl-cmp-aws-data.aws_billing.doitintl_billing_export_v1` + `
		  JOIN
			accounts
		  ON
			account_id = project_id
		  WHERE
			DATE(export_time) >= DATE_SUB(CURRENT_DATE(), INTERVAL 31 DAY)
			-- filter for CHT id's
			AND SAFE_CAST(billing_account_id AS INT64) IS NOT NULL
			AND (service_id LIKE '%AWS%Support%' OR service_id = 'OCBPremiumSupport' OR service_id = 'AWS Support Costs')
			AND cost_type NOT IN ('Tax', 'Credit')
		  GROUP BY
			1, 2, 3, 4)

		  SELECT
			support_model,
			support_tier,
			project_id AS account_id,
			is_ples_special_case,
			"false" AS is_overridable
		  FROM(
			SELECT
			  *,
			  ROW_NUMBER() OVER(PARTITION BY support_model, support_tier, project_id ORDER BY max_export_time DESC) as rank
			FROM
			  mid)
		  WHERE
			rank = 1)`

	supportTypeFromQuery, err := s.GetSupportPerAssetMapFromQuery(ctx, query)
	if err != nil {
		s.Logging.Logger(ctx).Errorf("Failed to get assets support type from invoice query: %v", err)
		return nil, err
	}

	return supportTypeFromQuery, nil
}

func (s *AwsAssetsSupportService) GetAwsAssetsSupport(ctx context.Context) ([]AssetSupport, error) {
	// Locate & create a list of all AssetsSupport structs
	assetsIDRefAndSupportType, err := s.GetAssets(ctx)
	if err != nil {
		s.Logging.Logger(ctx).Errorf("Failed to get assets map: %v", err)
		return nil, err
	}

	// create a list of all assets IDs
	var assetIDList []string
	for _, assetInfo := range assetsIDRefAndSupportType {
		assetIDList = append(assetIDList, assetInfo.AssetID)
	}

	// Query all assetsIds together for their support type from invoicing tables - in 1 query
	if len(assetIDList) > 0 {
		assetsSupportMap, err := s.GetAwsAssetsSupportFromBilling(ctx)
		if err != nil {
			s.Logging.Logger(ctx).Errorf("Failed to get assets support from billing: %v", err)
			return nil, err
		}
		// sort the slice by assetid for easier search and inserting data
		sort.SliceStable(assetsIDRefAndSupportType, func(i, j int) bool {
			return assetsIDRefAndSupportType[i].AssetID < assetsIDRefAndSupportType[j].AssetID
		})
		// save the result in the structs, search by assetID of each item in results
		for assetID, assetSupportInfo := range assetsSupportMap {
			// find the relevant assetSupport element in assetsIDRefAndSupportType by comparing asset Id
			i := sort.Search(len(assetsIDRefAndSupportType), func(i int) bool { return assetsIDRefAndSupportType[i].AssetID >= assetID })
			if i < len(assetsIDRefAndSupportType) && assetsIDRefAndSupportType[i].AssetID == assetID {
				// store the supportInfo from the query in the matching element
				assetsIDRefAndSupportType[i].IsPLESAsset = assetSupportInfo["PLES"] == "true"
				assetsIDRefAndSupportType[i].IsOverridable = assetSupportInfo["IsOverridable"] == "true"
				assetsIDRefAndSupportType[i].SupportModel = assetSupportInfo["SupportModel"]
				assetsIDRefAndSupportType[i].SupportTier = assetSupportInfo["SupportTier"]
			}
		}

		return assetsIDRefAndSupportType, nil
	}

	return nil, nil
}

func GetAssetSettingsCurrentOverridingValues(ctx context.Context, assetSettingsRef *firestore.DocumentRef) (*AssestSupportOverridenValues, error) {
	dsnap, err := assetSettingsRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var (
		overridingSupportTier string
		overrideReason        string
		overriddenOn          *time.Time
	)

	if fieldVal, err := dsnap.DataAt("settings.support.overridingSupportTier"); err == nil {
		if v, ok := fieldVal.(string); ok {
			overridingSupportTier = v
		}
	}

	if fieldVal, err := dsnap.DataAt("settings.support.overrideReason"); err == nil {
		if v, ok := fieldVal.(string); ok {
			overrideReason = v
		}
	}

	if fieldVal, err := dsnap.DataAt("settings.support.overriddenOn"); err != nil {
		overriddenOn = nil
	} else if v, ok := fieldVal.(time.Time); ok && !v.IsZero() {
		overriddenOn = &v
	}

	return &AssestSupportOverridenValues{
		OverridingSupportTier: overridingSupportTier,
		OverrideReason:        overrideReason,
		OverriddenOn:          overriddenOn,
	}, nil
}

func (s *AwsAssetsSupportService) BatchUpdateFSAssetsSupportType(ctx context.Context, assetsSupportToUpdate []AssetSupport) error {
	fs := s.conn.Firestore(ctx)
	batch := doitFirestore.NewBatchProviderWithClient(fs, 150).Provide(ctx)

	for _, asset := range assetsSupportToUpdate {
		if asset.Ref != nil {
			overridingValues, err := GetAssetSettingsCurrentOverridingValues(ctx, asset.AssetSettingsRef)
			if err != nil {
				s.Logging.Logger(ctx).Errorf("Failed to get assetSetting document from firestore: %v", err)
				continue
			}

			if err := batch.Update(ctx, asset.AssetSettingsRef, []firestore.Update{
				{
					FieldPath: []string{"settings", "support"},
					Value: pkg.AWSSettingsSupport{
						SupportModel:          asset.SupportModel,
						SupportTier:           asset.SupportTier,
						IsPLESAsset:           asset.IsPLESAsset,
						IsOverridable:         asset.IsOverridable,
						OverridingSupportTier: &overridingValues.OverridingSupportTier,
						OverrideReason:        &overridingValues.OverrideReason,
						OverriddenOn:          overridingValues.OverriddenOn,
					},
				},
			},
			); err != nil {
				s.Logging.Logger(ctx).Errorf("Failed to batch update assetSetting in firestore - got this error: %v", err)
				continue
			}
		}
	}

	if err := batch.Commit(ctx); err != nil {
		s.Logging.Logger(ctx).Errorf("Failed to batch commit: %v", err)
		return err
	}

	return nil
}

func (s *AwsAssetsSupportService) AssetsToUpdate(ctx context.Context) ([]AssetSupport, error) {
	var assetsToUpdate []AssetSupport

	assetsData, err := s.GetAwsAssetsSupport(ctx)
	if err != nil {
		s.Logging.Logger(ctx).Errorf("Failed to Identify & get support type for payer related assets: %v", err)
		return nil, err
	}
	// store them in the struct-based list with the support type (string resulted from BQ? use it. Empty? means its has no support)
	for _, asset := range assetsData {
		if asset.SupportTier == "" {
			asset.SupportTier = "basic"
		}

		assetsToUpdate = append(assetsToUpdate, asset)
	}

	return assetsToUpdate, nil
}

func (s *AwsAssetsSupportService) UpdateAWSSupportAssetsTypeInFS(ctx context.Context) error {
	// prepare an empty asset list based on struct
	var assetsToUpdate []AssetSupport

	// // 1. Identify & get support type for payer related assets
	assetsToUpdate, err := s.AssetsToUpdate(ctx)
	if err != nil {
		s.Logging.Logger(ctx).Errorf("Error Identifying & getting support type for payer accounts: %v", err)
	}

	// 2. Update support field in assetSettings collection for all Assets
	if err = s.BatchUpdateFSAssetsSupportType(ctx, assetsToUpdate); err != nil {
		s.Logging.Logger(ctx).Errorf("Failed to batch update support type field for all assets: %v", err)
		return err
	}

	s.Logging.Logger(ctx).Info("UpdateAWSSupportAssetsTypeInFS function finished")

	return nil
}
