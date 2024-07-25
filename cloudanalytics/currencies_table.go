package cloudanalytics

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"cloud.google.com/go/storage"
	"golang.org/x/time/rate"

	gcpTableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
)

type CurrencyRow struct {
	InvoiceMonth   civil.Date `json:"invoice_month"`
	Currency       string     `json:"currency"`
	ConversionRate float64    `json:"currency_conversion_rate"`
}

func GetCurrenciesTableName() string {
	if common.Production {
		return "gcp_currencies_v1"
	}

	return "gcp_currencies_v1beta"
}

func (s *CloudAnalyticsService) UpdateCurrenciesTable(ctx context.Context) error {
	l := s.loggerProvider(ctx)
	bq := s.conn.Bigquery(ctx)
	gcs := s.conn.CloudStorage(ctx)

	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	schema := bigquery.Schema{
		{Name: "invoice_month", Required: true, Type: bigquery.DateFieldType},
		{Name: "currency", Required: true, Type: bigquery.StringFieldType},
		{Name: "currency_conversion_rate", Required: true, Type: bigquery.FloatFieldType},
	}

	// Fetch rates for historical months
	rows := make([]*CurrencyRow, 0)
	hrInput := fixer.HistoricalRatesInput{
		Base:    fixer.USD,
		Symbols: fixer.Currencies,
	}
	limiter := rate.NewLimiter(4, 1)

	current := time.Date(2017, 8, 1, 0, 0, 0, 0, time.UTC)
	for current.Before(currentMonth) {
		invoiceMonth := civil.DateOf(current)
		date := time.Date(current.Year(), current.Month()+1, 0, 0, 0, 0, 0, time.UTC)
		hrInput.Date = &date

		result, err := s.fixerService.HistoricalRates(ctx, &hrInput)
		if err != nil {
			return err
		}

		if !result.Success {
			err := errors.New("failed to fetch historical currency conversion rates")
			return err
		}

		for currency, convRate := range result.Rates {
			rows = append(rows, &CurrencyRow{
				InvoiceMonth:   invoiceMonth,
				Currency:       string(currency),
				ConversionRate: convRate,
			})
		}

		current = current.AddDate(0, 1, 0)

		limiter.Wait(ctx)
	}

	// Fetch rates for current month
	result, err := s.fixerService.LatestRates(ctx, &fixer.LatestRatesInput{
		Base:    fixer.USD,
		Symbols: fixer.Currencies,
	})
	if err != nil {
		return err
	}

	if !result.Success {
		err := errors.New("failed to fetch latest currency conversion rates")
		return err
	}

	for currency, convRate := range result.Rates {
		rows = append(rows, &CurrencyRow{
			InvoiceMonth:   civil.DateOf(currentMonth),
			Currency:       string(currency),
			ConversionRate: convRate,
		})
	}

	nl := []byte("\n")
	bucketID := fmt.Sprintf("%s-bq-load-jobs", common.ProjectID)
	objectName := fmt.Sprintf("currencies/%s.gzip", now.Format(time.RFC3339))
	obj := gcs.Bucket(bucketID).Object(objectName)
	objWriter := obj.NewWriter(ctx)
	gzipWriter := gzip.NewWriter(objWriter)

	for _, row := range rows {
		data, err := json.Marshal(row)
		if err != nil {
			return err
		}

		data = append(data, nl...)
		if _, err := gzipWriter.Write(data); err != nil {
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

	tableName := GetCurrenciesTableName()
	gcsRef := bigquery.NewGCSReference(fmt.Sprintf("gs://%s/%s", bucketID, objectName))
	gcsRef.SkipLeadingRows = 0
	gcsRef.MaxBadRecords = 0
	gcsRef.Schema = schema
	gcsRef.SourceFormat = bigquery.JSON
	gcsRef.AutoDetect = false
	gcsRef.IgnoreUnknownValues = true
	// TODO: move the table to another dataset/project
	loader := bq.DatasetInProject(gcpTableMgmtDomain.BillingProjectProd, gcpTableMgmtDomain.BillingDataset).Table(tableName).LoaderFrom(gcsRef)
	// loader := bq.DatasetInProject(BillingProjectProd, BillingDataset).Table(tableName + "$" + now.Format("20060102")).LoaderFrom(gcsRef)
	loader.WriteDisposition = bigquery.WriteTruncate
	loader.CreateDisposition = bigquery.CreateIfNeeded
	loader.TimePartitioning = &bigquery.TimePartitioning{Type: "DAY", Field: "invoice_month"}
	loader.Clustering = &bigquery.Clustering{Fields: []string{"currency"}}
	loader.JobIDConfig = bigquery.JobIDConfig{
		JobID:          "gcp_billing_currencies",
		AddJobIDSuffix: true,
	}

	job, err := loader.Run(ctx)
	if err != nil {
		return err
	}

	l.Infof("job id: %s", job.ID())

	status, err := job.Wait(ctx)
	if err != nil {
		return err
	}

	if err := status.Err(); err != nil {
		return err
	}

	return nil
}
