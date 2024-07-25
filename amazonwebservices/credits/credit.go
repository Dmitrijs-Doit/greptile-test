package credits

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/aws"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const appCollection = "app"
const creditsFlagDoc = "invoicing-shared-payer-credits"

type AwsSharedPayersCreditService struct {
	*logger.Logging
	conn *connection.Connection
}

type MonthsFlag struct {
	Months map[string]time.Time `firestore:"months"`
}

type QueryBillingIsOverResultRow struct {
	TotalRowsCount   int `bigquery:"total_rows_count"`
	InvoiceRowsCount int `bigquery:"invoice_rows_count"`
}

type QueryCreditsResultRow struct {
	Name      string  `bigquery:"description"`
	AccountID string  `bigquery:"account_id"`
	Amount    float64 `bigquery:"cost"`
}

func NewAWSSharedPayersCredits(log *logger.Logging, conn *connection.Connection) (*AwsSharedPayersCreditService, error) {
	return &AwsSharedPayersCreditService{
		log,
		conn,
	}, nil
}

func GetLastMonthStartDate() time.Time {
	t := time.Now()
	t = t.AddDate(0, -1, 0)

	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func GetCurrentMonthStartDate() time.Time {
	t := time.Now()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func (s *AwsSharedPayersCreditService) GetMonthUpdateFlagDocRef(ctx context.Context) *firestore.DocumentRef {
	return s.conn.Firestore(ctx).Collection(appCollection).Doc(creditsFlagDoc)
}

func (s *AwsSharedPayersCreditService) GetMonthUpdateFlags(ctx context.Context) (MonthsFlag, error) {
	l := s.Logger(ctx)

	monthsFs, err := s.GetMonthUpdateFlagDocRef(ctx).Get(ctx)
	if err != nil {
		l.Errorf("Failed to get mpasDocSnaps Error: %v", err)
		return MonthsFlag{}, err
	}

	var months MonthsFlag
	if err := monthsFs.DataTo(&months); err != nil {
		return MonthsFlag{}, err
	}

	return months, nil
}

func GetLastMonthString() string {
	t := time.Now()
	t = t.AddDate(0, -1, 0)

	return t.Format("200601")
}

func (s *AwsSharedPayersCreditService) IsUpdatedFlagSet(ctx context.Context) (bool, error) {
	monthsFs, _ := s.GetMonthUpdateFlags(ctx)

	lastMonth := GetLastMonthString()

	if _, found := monthsFs.Months[lastMonth]; found {
		return true, nil
	}

	return false, nil
}

func (s *AwsSharedPayersCreditService) SetUpdatedFlagForMonth(ctx context.Context) error {
	monthsFs, _ := s.GetMonthUpdateFlags(ctx)

	lastMonth := GetLastMonthString()

	monthsFs.Months[lastMonth] = time.Now()

	if _, err := s.GetMonthUpdateFlagDocRef(ctx).Set(ctx, monthsFs); err != nil {
		return err
	}

	return nil
}

func (s *AwsSharedPayersCreditService) GetCreditsBaseSelectQuery(ctx context.Context) (string, error) {
	usersDocs, err := s.conn.Firestore(ctx).CollectionGroup("accountConfiguration").Where("useAnalyticsDataForInvoice", "==", false).Documents(ctx).GetAll()
	if err != nil {
		return "", err
	}

	tableFormat := "`doitintl-cmp-aws-data.aws_billing_%s.doitintl_billing_export_v1_%s`"

	var chtBilledCustomers []string

	// check if customer table exists in BQ before adding it to the query
	for _, user := range usersDocs {
		customerID := strings.Split(user.Ref.Path, "/")[6]
		exists, err := s.tableExists(ctx, customerID)
		if err != nil {
			return "", err
		}

		if exists {
			chtBilledCustomers = append(chtBilledCustomers, customerID)
		}
	}

	specialCaseCustomers := ""

	for _, customerID := range chtBilledCustomers {
		tableName := fmt.Sprintf(tableFormat, customerID, customerID)
		basQuery := `
		UNION ALL SELECT * FROM ` + tableName +
			`
		`
		specialCaseCustomers += basQuery
	}

	querySelect := `(SELECT * FROM ` + `doitintl-cmp-aws-data.payer_accounts.payer_account_doit_reseller_account_n1_561602220360` + `
	UNION ALL
	SELECT * FROM ` + `doitintl-cmp-aws-data.payer_accounts.payer_account_doit_reseller_account_n7_279843869311` + `

	-- Because we bill these customers with CHT we need to get their credits like we do from the CHT report these automation is replacing
	%s
	)`

	query := fmt.Sprintf(querySelect, specialCaseCustomers) + `
	WHERE
	DATE(export_time) BETWEEN DATE_TRUNC(DATE_SUB(CURRENT_DATE(), INTERVAL 1 MONTH), MONTH) AND LAST_DAY(DATE_TRUNC(DATE_SUB(CURRENT_DATE(), INTERVAL 1 MONTH), MONTH))
	AND cost_type IN ('Credit', 'Refund')
	AND project_id NOT IN ('561602220360', '279843869311') -- credits on the root accounts of the shared payers belong to us
	-- no spp/edp
	AND NOT (
			(cost_type IN ('Credit', 'Refund') AND (LOWER(description) LIKE '%spp-%'OR LOWER(description) LIKE 'aws solution provider program discount%'))
			OR cost_type = 'SppDiscount'
		)`

	return query, nil
}

func (s *AwsSharedPayersCreditService) tableExists(ctx context.Context, customerID string) (bool, error) {
	l := s.Logger(ctx)

	s.conn.Bigquery(ctx)
	dataset := s.conn.Bigquery(ctx).DatasetInProject("doitintl-cmp-aws-data", "aws_billing_"+customerID)
	table := dataset.Table("doitintl_billing_export_v1_" + customerID)

	_, err := table.Metadata(ctx)
	if err != nil {
		// Check if the error is because the table does not exist
		if e, ok := err.(*googleapi.Error); ok && e.Code == 404 {
			l.Warningf("tableExists() table not found : %v", err)
			return false, nil
		}
		l.Warningf("tableExists() error checking table metadata: %v", err)
		return false, err
	}

	return true, nil
}

func (s *AwsSharedPayersCreditService) IsBillingMonthIsOver(ctx context.Context) (bool, error) {
	l := s.Logger(ctx)

	baseSelectQuery, err := s.GetCreditsBaseSelectQuery(ctx)
	if err != nil {
		l.Errorf("error buliding query: %v", err)
		return false, err
	}

	queryTemplate := `WITH src AS (SELECT
		system_labels,
		row_id
	  FROM
	  	%s
	 ),

	  rows_count AS (
		SELECT
		  COUNT(*) AS count
		FROM
		  src
	  ),

	  invoice_count AS (
		SELECT
		  COUNT(*) AS count
		FROM
		  src, UNNEST(system_labels) sl
		WHERE
		  sl.key = 'aws/invoice_id'
	  )

	  SELECT
		(SELECT count FROM rows_count) AS total_rows_count,
		(SELECT count FROM invoice_count) AS invoice_rows_count`

	query := fmt.Sprintf(queryTemplate, baseSelectQuery)

	q := s.conn.Bigquery(ctx).Query(query)

	rows, err := q.Read(ctx)
	if err != nil {
		l.Errorf("error reading query: %v", err)
		return false, err
	}

	for {
		var row QueryBillingIsOverResultRow

		err := rows.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			l.Errorf("Failed to iterate over results: %v", err)
			continue
		}

		return row.TotalRowsCount == row.InvoiceRowsCount, nil
	}

	return false, nil
}

func (s *AwsSharedPayersCreditService) GetAssetDetails(ctx context.Context, accountID string) (*pkg.AWSAsset, error) {
	l := s.Logger(ctx)
	docID := fmt.Sprintf("amazon-web-services-%s", accountID)

	snap, err := s.conn.Firestore(ctx).Collection("assets").Doc(docID).Get(ctx)

	if err != nil {
		l.Errorf("Failed to get asset doc: %v", err)
		return nil, err
	}

	var asset *pkg.AWSAsset
	if err := snap.DataTo(&asset); err != nil {
		return nil, err
	}

	return asset, nil
}

func (s *AwsSharedPayersCreditService) GetEntityDetails(ctx context.Context, docID string) (*common.Entity, error) {
	l := s.Logger(ctx)

	snap, err := s.conn.Firestore(ctx).Collection("entities").Doc(docID).Get(ctx)
	if err != nil {
		l.Errorf("Failed to get entity doc: %v", err)
		return nil, err
	}

	var entity *common.Entity
	if err := snap.DataTo(&entity); err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *AwsSharedPayersCreditService) GetCustomerDocRef(ctx context.Context, customerID string) *firestore.DocumentRef {
	return s.conn.Firestore(ctx).Collection("customers").Doc(customerID)
}

func (s *AwsSharedPayersCreditService) GeCustomerDetails(ctx context.Context, customerID string) (*common.Customer, error) {
	l := s.Logger(ctx)

	snap, err := s.GetCustomerDocRef(ctx, customerID).Get(ctx)
	if err != nil {
		l.Errorf("Failed to get customer doc: %v", err)
		return nil, err
	}

	var customer *common.Customer
	if err := snap.DataTo(&customer); err != nil {
		return nil, err
	}

	return customer, nil
}

func (s *AwsSharedPayersCreditService) UpdateCustomerCredits(ctx context.Context) error {
	l := s.Logger(ctx)

	baseSelectQuery, err := s.GetCreditsBaseSelectQuery(ctx)
	if err != nil {
		l.Errorf("error buliding query: %v", err)
		return err
	}

	queryTemplate := `WITH src AS (SELECT
			description,
			project_id AS account_id,
			ABS(ROUND(SUM(cost))) AS cost,
		FROM
			%s
		GROUP BY
		1, 2
		)

		SELECT
			*
		FROM
			src
		WHERE
			cost > 0`

	query := fmt.Sprintf(queryTemplate, baseSelectQuery)

	q := s.conn.Bigquery(ctx).Query(query)

	rows, err := q.Read(ctx)
	if err != nil {
		l.Errorf("error reading query: %v", err)
		return err
	}

	fs := s.conn.Firestore(ctx)
	batch := doitFirestore.NewBatchProviderWithClient(fs, 250).Provide(ctx)

	lastMonthStartDate := GetLastMonthStartDate()
	currentMonthStartDate := GetCurrentMonthStartDate()
	invoiceMonth := GetLastMonthString()

	for {
		var row QueryCreditsResultRow

		err := rows.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			l.Errorf("Failed to iterate over results: %v", err)
			continue
		}

		assetDetails, err := s.GetAssetDetails(ctx, row.AccountID)
		if err != nil {
			l.Errorf("Failed to get asset details: %v", err)
			continue
		}

		entityDetails, err := s.GetEntityDetails(ctx, assetDetails.Entity.ID)
		if err != nil {
			l.Errorf("Failed to get entity details: %v", err)
			continue
		}

		customerDetails, err := s.GeCustomerDetails(ctx, assetDetails.Customer.ID)
		if err != nil {
			l.Errorf("Failed to get customer details: %v", err)
			continue
		}

		credit := aws.CustomerCreditAmazonWebServices{
			Name:          row.Name,
			Amount:        row.Amount,
			Currency:      string(fixer.USD),
			StartDate:     lastMonthStartDate,
			EndDate:       currentMonthStartDate,
			DepletionDate: nil,
			Utilization:   make(map[string]map[string]float64),
			Type:          common.Assets.AmazonWebServices,
			Customer:      s.GetCustomerDocRef(ctx, assetDetails.Customer.ID),
			Entity:        assetDetails.Entity,
			Assets:        make([]*firestore.DocumentRef, 0),
			UpdatedBy: map[string]interface{}{
				"email": "noreply@doit.com",
				"name":  "Automated",
			},
			Metadata: map[string]interface{}{
				"customer": map[string]interface{}{
					"primaryDomain": customerDetails.PrimaryDomain,
					"name":          customerDetails.Name,
					"priorityId":    entityDetails.PriorityID,
				},
				"service":      "aws-shared-payer-credits",
				"invoiceMonth": invoiceMonth,
			},
		}

		newDocRef := s.GetCustomerDocRef(ctx, assetDetails.Customer.ID).Collection("customerCredits").NewDoc()

		_ = batch.Set(ctx, newDocRef, credit)
	}

	if err := batch.Commit(ctx); err != nil {
		l.Errorf("Failed to commit to fs: %v", err)
		return err
	}

	return nil
}

func (s *AwsSharedPayersCreditService) UpdateAWSSharedPayersCredits(ctx context.Context) error {
	l := s.Logger(ctx)
	l.Infof("UpdateAWSSharedPayersCredits() - started")

	fsUpdateFlag, _ := s.IsUpdatedFlagSet(ctx)
	if fsUpdateFlag {
		l.Infof("FS flag is already updated")
		return nil
	}

	billingOverFlag, err := s.IsBillingMonthIsOver(ctx)
	if err != nil {
		l.Errorf("Error checking if billing month is over: %v", err)
		return err
	}

	if !billingOverFlag {
		l.Infof("Billing are not done yet")
		return nil
	}

	err = s.UpdateCustomerCredits(ctx)
	if err != nil {
		l.Errorf("Error updating customer credits: %v", err)
		return err
	}

	err = s.SetUpdatedFlagForMonth(ctx)
	if err != nil {
		l.Errorf("Error updating customer credits flag: %v", err)
		return err
	}

	return nil
}
