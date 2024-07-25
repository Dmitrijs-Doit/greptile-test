package googlecloud

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type IAMResourceRow struct {
	Customer string `json:"customer"`
	ID       string `json:"id"`
	Name     string `json:"name"`
}

type IAMResourceDoc struct {
	Customer  *firestore.DocumentRef `json:"customer"`
	Resources map[string]string      `json:"resources"`
	Timestamp time.Time              `json:"timestamp"`
}

func GetIAMResourcesTableName() string {
	if common.Production {
		return "gcp_iam_resources_v1"
	}

	return "gcp_iam_resources_v1beta"
}

func getIAMResources(ctx context.Context, fs *firestore.Client) ([]*IAMResourceRow, error) {
	docSnaps, err := fs.Collection("integrations").
		Doc(common.Assets.GoogleCloud).
		Collection("googleCloudResources").
		OrderBy("timestamp", firestore.Desc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	ids := map[string]struct{}{}
	resources := make([]*IAMResourceRow, 0)

	for _, docSnap := range docSnaps {
		var gcr IAMResourceDoc
		if err := docSnap.DataTo(&gcr); err != nil {
			return nil, err
		}

		for id, name := range gcr.Resources {
			if _, ok := ids[id]; ok {
				continue
			}

			resources = append(resources, &IAMResourceRow{
				Customer: gcr.Customer.ID,
				ID:       id,
				Name:     name,
			})
			ids[id] = struct{}{}
		}
	}

	return resources, nil
}

func (s *BillingTableManagementService) UpdateIAMResources(ctx context.Context) error {
	bq := s.conn.Bigquery(ctx)
	gcs := s.conn.CloudStorage(ctx)
	fs := s.conn.Firestore(ctx)

	iamResources, err := getIAMResources(ctx, fs)
	if err != nil {
		return err
	}

	schema := bigquery.Schema{
		{Name: "customer", Required: true, Type: bigquery.StringFieldType},
		{Name: "id", Required: true, Type: bigquery.StringFieldType},
		{Name: "name", Required: true, Type: bigquery.StringFieldType},
	}

	nl := []byte("\n")
	now := time.Now().UTC()
	bucketID := fmt.Sprintf("%s-bq-load-jobs", common.ProjectID)
	objectName := fmt.Sprintf("iam-resources/%s.gzip", now.Format(time.RFC3339))
	obj := gcs.Bucket(bucketID).Object(objectName)
	objWriter := obj.NewWriter(ctx)
	gzipWriter := gzip.NewWriter(objWriter)

	for _, iamResource := range iamResources {
		data, err := json.Marshal(iamResource)
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

	tableName := GetIAMResourcesTableName()
	gcsRef := bigquery.NewGCSReference(fmt.Sprintf("gs://%s/%s", bucketID, objectName))
	gcsRef.SkipLeadingRows = 0
	gcsRef.MaxBadRecords = 0
	gcsRef.Schema = schema
	gcsRef.SourceFormat = bigquery.JSON
	gcsRef.AutoDetect = false
	gcsRef.IgnoreUnknownValues = true
	loader := bq.DatasetInProject(domain.BillingProjectProd, domain.BillingDataset).Table(tableName).LoaderFrom(gcsRef)
	loader.WriteDisposition = bigquery.WriteTruncate
	loader.CreateDisposition = bigquery.CreateIfNeeded
	loader.TimePartitioning = &bigquery.TimePartitioning{Type: "DAY"}
	loader.Clustering = &bigquery.Clustering{Fields: []string{"customer"}}
	loader.JobIDConfig = bigquery.JobIDConfig{
		JobID:          "gcp_billing_iam_resources",
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

	return nil
}
