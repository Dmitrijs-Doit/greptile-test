package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	gcpTableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	datasetID        = "analytics"
	bucketSuffix     = "firestore-export"
	collectionPrefix = "/kind_"
	collectionSuffix = "/all_namespaces_kind_"
	objectSuffix     = ".export_metadata"
	offset           = len(collectionPrefix)
)

var (
	// Collection listed here will only import the specified fields slice into bigquery
	projectionFields = map[string][]string{
		"contracts":                 {"type", "customer", "entity", "assets", "active", "terminated", "isCommitment", "commitmentPeriods", "commitmentRollover", "discount", "estimatedValue", "startDate", "endDate", "timestamp", "timeCreated", "accountManager", "notes", "purchaseOrder", "isRenewal", "vendorContract", "partnerMargin", "plpsPercent"},
		"customerCredits":           {"amount", "assets", "currency", "customer", "depletionDate", "endDate", "entity", "name", "startDate", "timestamp", "type", "updatedBy"},
		"flexibleReservedInstances": {"clientId", "config", "createdAt", "customer", "email", "entity", "id", "normalizedUnits", "pricing", "status", "uid", "execution"},
	}
)

func FirestoreImportBigQueryHandler(ctx context.Context) error {
	l := logger.FromContext(ctx)

	now := time.Now().UTC()

	gcs, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	bq, err := bigquery.NewClient(ctx, common.ProjectID)
	if err != nil {
		return err
	}

	bq.Location = "US"
	dataset := bq.Dataset(datasetID)

	objects := make([]string, 0)
	prefix := ""
	bucketID := fmt.Sprintf("%s-%s", common.ProjectID, bucketSuffix)
	objectsQuery := storage.Query{Delimiter: "output", Prefix: now.Format("2006-01-02T")}
	objectsIter := gcs.Bucket(bucketID).Objects(ctx, &objectsQuery)

	for {
		objectAttrs, err := objectsIter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return err
		}

		if strings.HasSuffix(objectAttrs.Name, objectSuffix) {
			objects = append(objects, objectAttrs.Name)

			parts := strings.SplitN(objectAttrs.Name, "/", 2)
			if parts[0] > prefix {
				prefix = parts[0]
			}
		}
	}

	for _, object := range objects {
		if !strings.HasPrefix(object, prefix) {
			continue
		}

		l.Debugf("Object: %s", object)
		collectionPrefixIndex := strings.Index(object, collectionPrefix)
		collectionSuffixIndex := strings.Index(object, collectionSuffix)

		if collectionPrefixIndex < 0 || collectionSuffixIndex < 0 {
			continue
		}

		collectionID := object[collectionPrefixIndex+offset : collectionSuffixIndex]
		l.Debugf("Collection: %s", collectionID)

		gcsRef := bigquery.NewGCSReference("gs://" + bucketID + "/" + object)
		gcsRef.SourceFormat = bigquery.DatastoreBackup
		gcsRef.AutoDetect = true

		// save this collection in the gcp_billing dataset as well, to be used with reports
		if collectionID == "googleCloudBillingSkus" {
			table := bq.DatasetInProject(gcpTableMgmtDomain.BillingProjectProd, gcpTableMgmtDomain.BillingDataset).Table(gcpTableMgmtDomain.GetBillingSkusTableName())
			if err := runLoadJob(ctx, table, gcsRef); err != nil {
				l.Errorf("failed to run load job for collection %s in dataset %s with error %s", collectionID, gcpTableMgmtDomain.BillingDataset, err)
			}
		}

		if err := runLoadJob(ctx, dataset.Table(collectionID), gcsRef); err != nil {
			l.Errorf("failed to run load job for collection %s with error %s", collectionID, err)
		}
	}

	return nil
}

func runLoadJob(ctx context.Context, table *bigquery.Table, src *bigquery.GCSReference) error {
	loader := table.LoaderFrom(src)
	loader.WriteDisposition = bigquery.WriteTruncate
	loader.CreateDisposition = bigquery.CreateIfNeeded

	if fields, prs := projectionFields[table.TableID]; prs {
		loader.ProjectionFields = fields
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

	return nil
}
