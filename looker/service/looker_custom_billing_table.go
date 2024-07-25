package service

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/bqutils"
	gcpTableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/schema"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
	"github.com/doitintl/hello/scheduled-tasks/looker/domain"
	lookerUtils "github.com/doitintl/hello/scheduled-tasks/looker/utils"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

const (
	dayInHours = 24
)

func getProjectID() string {
	if common.Production {
		return consts.CustomBillingProd
	}

	return consts.CustomBillingDev
}
func (s *AssetsService) LoadLookerContractsToBQ(ctx *gin.Context, request domain.UpdateTableInterval) error {
	log := s.Logging.Logger(ctx)

	updateTableInterval, err := extractUpdateTableInterval(request)
	if err != nil {
		return err
	}

	contracts, err := s.contractsDAL.GetActiveCustomerContractsForProductType(ctx, "looker")
	if err != nil {
		log.Error(err)
		return err
	}

	billingRowsByPartition, err := s.CreateLookerRows(ctx, contracts, updateTableInterval)
	if err != nil {
		return err
	}

	tableExists := false

	sortedKeys := sortKeys(billingRowsByPartition)
	for _, key := range sortedKeys {
		tableLoaderPayload, err := s.GetTableLoaderPayload(ctx, billingRowsByPartition[key])
		if err != nil {
			return err
		}

		if !tableExists {
			tableExists, _, err = common.BigQueryTableExists(ctx, tableLoaderPayload.Client, tableLoaderPayload.Data.DestinationProjectID, tableLoaderPayload.Data.DestinationDatasetID, tableLoaderPayload.Data.DestinationTableName)
			if err != nil {
				return err
			}
		}

		if err := s.LookerBigQueryTableLoader(ctx, *tableLoaderPayload, key, tableExists); err != nil {
			log.Errorf("looker custom billing table - failed to load partition %s to table %s. error [%s]",
				key.Format("20060102"), tableLoaderPayload.Data.DestinationTableName, err)
			return err
		}

		tableExists = true
	}

	return nil
}

func extractUpdateTableInterval(updateTableInterval domain.UpdateTableInterval) ([]time.Time, error) {
	dates, err := getDataInterval(updateTableInterval)
	if err != nil {
		return nil, err
	}

	return dates, nil
}

func getDataInterval(updateTableInterval domain.UpdateTableInterval) ([]time.Time, error) {
	var days []time.Time

	var startDate, endDate time.Time

	var err error

	if updateTableInterval.StartDate != "" {
		startDate, err = time.Parse(times.YearMonthDayLayout, updateTableInterval.StartDate)
		if err != nil {
			return nil, err
		}
	}

	if updateTableInterval.EndDate != "" {
		if updateTableInterval.StartDate == "" {
			return nil, fmt.Errorf("end date %s is set, but start date is not", updateTableInterval.EndDate)
		}

		endDate, err = time.Parse(times.YearMonthDayLayout, updateTableInterval.EndDate)
		if err != nil {
			return nil, err
		}
	}

	startDate, err = checkDate(startDate)
	if err != nil {
		return nil, err
	}

	endDate, err = checkDate(endDate)
	if err != nil {
		return nil, err
	}

	// if start date equals end date, return start date
	if startDate.Equal(endDate) {
		days = append(days, startDate)
	} else if startDate.After(endDate) {
		return nil,
			fmt.Errorf("start date %s is after end date %s",
				startDate.Format(times.YearMonthDayLayout), endDate.Format(times.YearMonthDayLayout))
	} else {
		// return interval from start to end date, including both
		date := startDate
		for !date.After(endDate) {
			days = append(days, date)
			date = date.AddDate(0, 0, 1)
		}
	}

	return days, nil
}

func checkDate(date time.Time) (time.Time, error) {
	now := time.Now()
	sanitizedDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// if date is zero return today's date
	if date.IsZero() {
		date = sanitizedDate
	} else if date.After(sanitizedDate) {
		return date, fmt.Errorf("date %s is after today", date.Format(times.YearMonthDayLayout))
	}

	return date, nil
}

func (s *AssetsService) CreateLookerRows(ctx *gin.Context, contracts []*pkg.Contract, updateTableInterval []time.Time) (map[time.Time][]schema.BillingRow, error) {
	log := s.Logging.Logger(ctx)
	billingRowsByPartition := make(map[time.Time][]schema.BillingRow)

	for _, contract := range contracts {
		if lookerUtils.IsOldFormat(*contract) {
			continue
		}

		var properties domain.LookerContractProperties

		properties, err := properties.DecodePropertiesMapIntoStruct(contract.Properties)
		if err != nil {
			log.Errorf("invalid contract properties. error [%s]", err)
			continue
		}

		ba, err := getBillingAccountFromAssets(*contract)
		if err != nil {
			log.Infof("contract %s is not attached to any assets \n", contract.ID)
			continue
		}

		for _, day := range updateTableInterval {
			if properties.Skus != nil {
				for _, sku := range properties.Skus {
					if IsDayBillableForSku(sku, properties, day) {
						// add 24 rows for each day
						for i := 0; i < dayInHours; i++ {
							row := schema.BillingRow{
								Customer:               contract.Customer.ID,
								BillingAccountID:       ba,
								Cost:                   calculateHourlyCost(sku, sku.SkuName.MonthlyListPrice, properties, day),
								Currency:               string(fixer.USD),
								CurrencyConversionRate: 1,
								CostType:               "regular",
								SkuID:                  bigquery.NullString{StringVal: sku.SkuName.GoogleSKU, Valid: hasSKUID(sku.SkuName.GoogleSKU)},
								SkuDescription:         bigquery.NullString{StringVal: sku.SkuName.Label, Valid: true},
								CloudProvider:          common.Assets.GoogleCloud,
								ServiceDescription:     bigquery.NullString{StringVal: "Looker", Valid: true},
								Usage: &schema.Usage{
									Amount:               float64(sku.Quantity),
									AmountInPricingUnits: float64(sku.Quantity),
								},
								ServiceID: bigquery.NullString{StringVal: gcpTableMgmtDomain.LookerServiceID, Valid: true},
								Report: []schema.Report{
									{
										Cost: bigquery.NullFloat64{
											Float64: calculateHourlyCost(sku, sku.MonthlySalesPrice, properties, day) * utils.ToProportion(contract.Discount),
											Valid:   true,
										},
										Usage: bigquery.NullFloat64{
											Float64: float64(sku.Quantity),
											Valid:   true,
										},
									},
								},
								SystemLabels: []schema.Label{
									{
										Key:   "cmp/source",
										Value: "looker",
									},
								},
							}

							location, err := time.LoadLocation(domainQuery.TimeZonePST)
							if err != nil {
								break
							}

							y, m, d := day.Date()
							usageDateTime := time.Date(y, m, d, i, 0, 0, 0, location)

							if err := updateDateFields(&row, usageDateTime); err != nil {
								return nil, err
							}

							billingRowsByPartition[row.ExportTime] = append(billingRowsByPartition[row.ExportTime], row)
						}
					}
				}
			}
		}
	}

	return billingRowsByPartition, nil
}

func (s *AssetsService) GetTableLoaderPayload(ctx context.Context, billingRows []schema.BillingRow) (*bqutils.BigQueryTableLoaderParams, error) {
	projectID := getProjectID()
	client := s.bigQueryFromContextFun(ctx)
	requestData := bqutils.BigQueryTableLoaderRequest{
		DestinationProjectID:   projectID,
		DestinationDatasetID:   consts.CustomBillingDataset,
		DestinationTableName:   consts.LookerTable,
		ObjectDir:              consts.LookerTable,
		ConfigJobID:            consts.LookerTable,
		WriteDisposition:       bigquery.WriteTruncate,
		RequirePartitionFilter: true,
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

func updateDateFields(billingRow *schema.BillingRow, usageDateTime time.Time) error {
	// convert the pst day to utc
	billingRow.UsageStartTime = usageDateTime.UTC()
	billingRow.UsageEndTime = usageDateTime.UTC().Add(time.Hour)
	billingRow.UsageDateTime = bigquery.NullDateTime{
		DateTime: civil.DateTimeOf(usageDateTime),
		Valid:    true,
	}
	month := fmt.Sprintf("%d-%02d", usageDateTime.Year(), usageDateTime.Month())
	billingRow.Invoice = &schema.Invoice{
		Month: strings.ReplaceAll(month, "-", ""),
	}
	billingRow.ExportTime = time.Date(usageDateTime.Year(), usageDateTime.Month(), usageDateTime.Day(), 0, 0, 0, 0, time.UTC)

	return nil
}

func IsDayBillableForSku(sku domain.LookerContractSKU, properties domain.LookerContractProperties, reportDay time.Time) bool {
	end := sku.StartDate.AddDate(0, int(sku.Months), -1)

	frequency := int(properties.InvoiceFrequency)
	if frequency == lookerUtils.MonthlyInvoicingFrequency {
		return (sku.StartDate.Before(reportDay) || sku.StartDate.Equal(reportDay)) && (end.After(reportDay) || end.Equal(reportDay))
	}

	billingIteration := 0

	for {
		s := sku.StartDate.AddDate(0, billingIteration*frequency, 0)
		// if start + some multiple of the invoice frequency is the current invoice month && is before/equal to the end date: return true
		if (s.Month() == reportDay.Month() && s.Year() == reportDay.Year() && reportDay.Day() >= s.Day()) &&
			(s.Before(end) || s.Equal(end)) {
			return true
		} else if s.After(reportDay) || s.After(end) {
			return false
		}

		billingIteration++
	}
}
func getBillingAccountFromAssets(contract pkg.Contract) (string, error) {
	if len(contract.Assets) == 0 {
		return "", fmt.Errorf("no assets found for contract")
	}

	parts := strings.Split(contract.Assets[0].ID, "-")

	return strings.Join(parts[len(parts)-3:], "-"), nil
}

func hasSKUID(skuID string) bool {
	return skuID != ""
}

func sortKeys(billingRowsByPartition map[time.Time][]schema.BillingRow) []time.Time {
	// Step 1: Extract the keys (partitions) into a separate slice
	keys := make([]time.Time, 0, len(billingRowsByPartition))
	for key := range billingRowsByPartition {
		keys = append(keys, key)
	}

	// Step 2: Sort the slice of keys in ascending order
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})

	return keys
}

func calculateHourlyCost(sku domain.LookerContractSKU, monthlyPrice float64, properties domain.LookerContractProperties, invoiceMonth time.Time) float64 {
	totalDaysInMonth := lookerUtils.GetMonthLength(invoiceMonth)
	billableDaysInMonth := lookerUtils.GetNumOfBillableDaysInMonth(sku, int(sku.Months), invoiceMonth)

	if properties.InvoiceFrequency == lookerUtils.MonthlyInvoicingFrequency {
		return (monthlyPrice * float64(sku.Quantity) / float64(totalDaysInMonth)) / dayInHours
	}
	// return monthly price * remaining months in billing period * quantity / by billable days in month
	return ((monthlyPrice * float64(lookerUtils.GetRemainingMonthsInBillingPeriod(invoiceMonth, sku.StartDate, int(sku.Months), int(properties.InvoiceFrequency))) * float64(sku.Quantity)) / float64(billableDaysInMonth)) / dayInHours
}

func (s *AssetsService) LookerBigQueryTableLoader(ctx context.Context, loadAttributes bqutils.BigQueryTableLoaderParams, partition time.Time, tableExists bool) error {
	data := loadAttributes.Data
	bq := loadAttributes.Client
	gcs, err := storage.NewClient(ctx)

	if err != nil {
		return err
	}

	defer gcs.Close()

	nl := []byte("\n")
	now := time.Now().UTC()
	bucketID := fmt.Sprintf("%s-bq-load-jobs", common.ProjectID)
	objectName := fmt.Sprintf("%s/%s.gzip", data.ObjectDir, now.Format(time.RFC3339Nano))
	obj := gcs.Bucket(bucketID).Object(objectName)
	objWriter := obj.NewWriter(ctx)
	gzipWriter := gzip.NewWriter(objWriter)

	for _, row := range loadAttributes.Rows {
		jsonData, err := json.Marshal(row)
		if err != nil {
			return err
		}

		jsonData = append(jsonData, nl...)
		if _, err := gzipWriter.Write(jsonData); err != nil {
			return err
		}
	}

	if err := gzipWriter.Close(); err != nil {
		return err
	}

	if err := objWriter.Close(); err != nil {
		return err
	}

	if _, err := obj.Update(ctx, storage.ObjectAttrsToUpdate{
		ContentType:     "application/json",
		ContentEncoding: "gzip",
	}); err != nil {
		return err
	}

	gcsRef := bigquery.NewGCSReference(fmt.Sprintf("gs://%s/%s", bucketID, objectName))
	gcsRef.SkipLeadingRows = 0
	gcsRef.MaxBadRecords = 0
	gcsRef.Schema = *loadAttributes.Schema
	gcsRef.SourceFormat = bigquery.JSON
	gcsRef.AutoDetect = false
	gcsRef.IgnoreUnknownValues = true

	if data.RequirePartitionFilter && tableExists {
		data.DestinationTableName += "$" + partition.Format("20060102")
	}

	loader := bq.DatasetInProject(data.DestinationProjectID, data.DestinationDatasetID).Table(data.DestinationTableName).LoaderFrom(gcsRef)
	loader.WriteDisposition = data.WriteDisposition
	loader.CreateDisposition = bigquery.CreateIfNeeded

	if data.RequirePartitionFilter {
		loader.TimePartitioning = &bigquery.TimePartitioning{Type: "DAY"}
	}

	if data.PartitionField != "" {
		loader.TimePartitioning = &bigquery.TimePartitioning{Type: "DAY", Field: data.PartitionField}
	}

	if data.Clustering != nil {
		loader.Clustering = &bigquery.Clustering{Fields: *data.Clustering}
	}

	loader.JobIDConfig = bigquery.JobIDConfig{
		JobID:          data.ConfigJobID,
		AddJobIDSuffix: true,
	}

	job, err := loader.Run(ctx)
	if err != nil {
		return err
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return err
	}

	if err := status.Err(); err != nil {
		return err
	}

	table := bq.DatasetInProject(data.DestinationProjectID, data.DestinationDatasetID).Table(data.DestinationTableName)
	if md, err := table.Metadata(ctx); err == nil {
		if data.RequirePartitionFilter && !md.RequirePartitionFilter {
			mdtu := bigquery.TableMetadataToUpdate{
				RequirePartitionFilter: true,
			}
			if _, err := table.Update(ctx, mdtu, md.ETag); err != nil {
				return err
			}
		}
	} else {
		return err
	}

	return nil
}
