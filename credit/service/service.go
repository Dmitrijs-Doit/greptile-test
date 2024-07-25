package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/bqutils"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/schema"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/credit"
	creditsDal "github.com/doitintl/hello/scheduled-tasks/credit/dal"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

type CreditsService struct {
	loggerProvider     logger.Provider
	bigQueryClientFunc connection.BigQueryFromContextFun
	creditsDAL         *creditsDal.CreditsFirestore
}

const (
	CostTypeCreditAdjustment = "Credit Adjustment"
	CostTypeCredit           = "Credit"
	adjustmentSuffix         = "-discount"
)

var (
	ErrInvalidCreditUtilizationKey = errors.New("invalid credit utilization key")
)

func getProjectID() string {
	if common.Production {
		return consts.CustomBillingProd
	}

	return consts.CustomBillingDev
}

func NewCreditsService(loggerProvider logger.Provider, firestoreFun connection.FirestoreFromContextFun, bigQueryFromContextFun connection.BigQueryFromContextFun) (*CreditsService, error) {
	return &CreditsService{
		loggerProvider,
		bigQueryFromContextFun,
		creditsDal.NewCreditsFirestoreWithClient(firestoreFun),
	}, nil
}

func (s *CreditsService) LoadCreditsToBQ(ctx context.Context) error {
	creditData, err := s.creditsDAL.GetCredits(ctx)
	if err != nil {
		return err
	}

	billingRows, err := s.createCreditRows(ctx, creditData)
	if err != nil {
		return err
	}

	tableLoaderPayload, err := s.getTableLoaderPayload(ctx, billingRows)
	if err != nil {
		return err
	}

	if err := bqutils.BigQueryTableLoader(ctx, *tableLoaderPayload); err != nil {
		return err
	}

	return nil
}

func (s *CreditsService) createCreditRows(ctx context.Context, creditData map[*firestore.DocumentRef]credit.BaseCredit) ([]schema.BillingRow, error) {
	logger := s.loggerProvider(ctx)

	var billingRows []schema.BillingRow

	for creditRef, credit := range creditData {
		for month, utilizationMap := range credit.Utilization {
			var billingRow schema.BillingRow
			billingRow.Customer = credit.Customer.ID
			billingRow.CloudProvider = credit.Type
			billingRow.Currency = string(fixer.USD)
			billingRow.CurrencyConversionRate = 1

			if err := updateDateFields(&billingRow, credit.Type, month); err != nil {
				return nil, err
			}

			for key, value := range utilizationMap {
				billingAccountID, isCreditDiscountAdjustment, err := s.validateUtilizationKey(credit.Type, key)
				if err != nil {
					logger.Warningf("validate utilization key [%s][%s] error: path [%s] %s", month, key, creditRef.Path, err)
					continue
				}

				if isCreditDiscountAdjustment {
					billingRow.CostType = CostTypeCreditAdjustment
				} else {
					billingRow.CostType = CostTypeCredit

					if adjustmentValue, ok := utilizationMap[key+adjustmentSuffix]; ok {
						value += adjustmentValue
					}

					value *= -1
				}

				billingRow.BillingAccountID = billingAccountID
				if credit.Type == common.Assets.AmazonWebServices {
					billingRow.BillingAccountID = credit.Customer.ID
					billingRow.ProjectID = bigquery.NullString{
						StringVal: billingAccountID,
						Valid:     true,
					}
				}

				billingRow.Report = s.getReportValues(credit.Name, value)
				billingRows = append(billingRows, billingRow)
			}
		}
	}

	return billingRows, nil
}

var gcpUtilizationKeyRegexp = regexp.MustCompile("^(?:[A-F0-9]{6}-){2}[A-F0-9]{6}(?:-discount)?$")
var awsUtilizationKeyRegexp = regexp.MustCompile("^[\\d]{12}$")

func (s *CreditsService) validateUtilizationKey(cloudProvider, key string) (string, bool, error) {
	switch cloudProvider {
	case common.Assets.GoogleCloud:
		if !gcpUtilizationKeyRegexp.MatchString(key) {
			return "", false, ErrInvalidCreditUtilizationKey
		}

		isCreditDiscoutAdjustment := strings.HasSuffix(key, adjustmentSuffix)

		return key[0:20], isCreditDiscoutAdjustment, nil
	case common.Assets.AmazonWebServices:
		if !awsUtilizationKeyRegexp.MatchString(key) {
			return "", false, ErrInvalidCreditUtilizationKey
		}

		return key, false, nil
	default:
		return "", false, fmt.Errorf("invalid credit cloud provider %s", cloudProvider)
	}
}

func (s *CreditsService) getReportValues(creditName string, creditValue float64) []schema.Report {
	reportRows := []schema.Report{
		{
			Cost: bigquery.NullFloat64{
				Float64: creditValue,
				Valid:   true,
			},
			Usage: bigquery.NullFloat64{
				Float64: 0,
				Valid:   true,
			},
			Savings: bigquery.NullFloat64{
				Float64: 0,
				Valid:   true,
			},
			Credit: bigquery.NullString{
				StringVal: creditName,
				Valid:     true,
			},
			ExtMetric: nil,
		},
	}

	return reportRows
}

func (s *CreditsService) getTableLoaderPayload(ctx context.Context, billingRows []schema.BillingRow) (*bqutils.BigQueryTableLoaderParams, error) {
	projectID := getProjectID()

	client := s.bigQueryClientFunc(ctx)

	requestData := bqutils.BigQueryTableLoaderRequest{
		DestinationProjectID:   projectID,
		DestinationDatasetID:   consts.CustomBillingDataset,
		DestinationTableName:   consts.CreditsTable,
		ObjectDir:              consts.CreditsTable,
		ConfigJobID:            consts.CreditsTable,
		WriteDisposition:       bigquery.WriteTruncate,
		RequirePartitionFilter: false,
		PartitionField:         domainQuery.FieldExportTime,
		Clustering:             &[]string{domainQuery.FieldCustomer, domainQuery.FieldCloudProvider},
	}

	rows := make([]interface{}, len(billingRows))
	for i, v := range billingRows {
		rows[i] = v
	}

	loaderAttributes := bqutils.BigQueryTableLoaderParams{
		Client: client,
		Schema: &schema.CreditsSchema,
		Rows:   rows,
		Data:   &requestData,
	}

	return &loaderAttributes, nil
}

func updateDateFields(billingRow *schema.BillingRow, creditType string, month string) error {
	usageStartTime, err := time.ParseInLocation(times.YearMonthLayout, month, time.UTC)
	if err != nil {
		return err
	}

	usageDateTime := usageStartTime

	if creditType == common.Assets.GoogleCloud {
		location, err := time.LoadLocation(domainQuery.TimeZonePST)
		if err != nil {
			return err
		}

		y, m, d := usageDateTime.Date()
		usageDateTime = time.Date(y, m, d, 0, 0, 0, 0, location)
	}

	billingRow.ExportTime = usageStartTime
	billingRow.UsageStartTime = usageStartTime
	billingRow.UsageEndTime = usageStartTime.AddDate(0, 1, -1)
	billingRow.UsageDateTime = bigquery.NullDateTime{
		DateTime: civil.DateTimeOf(usageDateTime),
		Valid:    true,
	}
	billingRow.Invoice = &schema.Invoice{
		Month: strings.ReplaceAll(month, "-", ""),
	}

	return nil
}
