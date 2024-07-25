package bqutils

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type BigQueryTableLoaderRequest struct {
	DestinationProjectID   string
	DestinationDatasetID   string
	DestinationTableName   string
	ObjectDir              string
	ConfigJobID            string
	RequirePartitionFilter bool
	PartitionField         string
	WriteDisposition       bigquery.TableWriteDisposition
	Clustering             *[]string
}

type BigQueryTableLoaderParams struct {
	Client *bigquery.Client
	Schema *bigquery.Schema
	Rows   []interface{}
	Data   *BigQueryTableLoaderRequest
}

func BigQueryTableLoader(ctx context.Context, loadAttributes BigQueryTableLoaderParams) error {
	data := loadAttributes.Data
	bq := loadAttributes.Client

	gcs, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	defer gcs.Close()

	nl := []byte("\n")
	now := time.Now().UTC()
	bucketID := fmt.Sprintf("%s-bq-load-jobs", getProjectID())
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

	tableName := data.DestinationTableName
	if data.RequirePartitionFilter {
		tableName += "$" + now.Format("20060102")
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

func getProjectID() string {
	if common.ProjectID == "me-doit-intl-com" {
		return common.ProjectID
	}

	return "doitintl-cmp-dev"
}
