package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	doitFS "github.com/doitintl/firestore"
	fsIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/pkg"
)

const (
	jobsSinksMetadataCollection = "superQuery/jobs-sinks/jobsSinksMetadata"
)

type JobsSinksMetadataDal struct {
	firestoreClient             *firestore.Client
	documentsHandler            fsIface.DocumentsHandler
	jobsSinksMetadataCollection string
}

func NewJobsSinksMetadataDal(fs *firestore.Client) *JobsSinksMetadataDal {
	return &JobsSinksMetadataDal{
		firestoreClient:             fs,
		documentsHandler:            doitFS.DocumentHandler{},
		jobsSinksMetadataCollection: jobsSinksMetadataCollection,
	}
}

func (d *JobsSinksMetadataDal) GetSinkMetadata(ctx context.Context, sinkID string) (*pkg.SinkMetadata, error) {
	docSnap, err := d.firestoreClient.Collection(d.jobsSinksMetadataCollection).Doc(sinkID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var doc pkg.SinkMetadata

	if err := docSnap.DataTo(&doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

func (d *JobsSinksMetadataDal) UpdateBackfillProgress(ctx context.Context, sinkID string, projectsToBeBackfilled []string) error {
	var backfillDone bool

	if len(projectsToBeBackfilled) == 0 {
		backfillDone = true
	}

	_, err := d.firestoreClient.Collection(d.jobsSinksMetadataCollection).
		Doc(sinkID).
		Set(ctx, map[string]interface{}{
			"backfillProjectsNumber": len(projectsToBeBackfilled),
			"projectsToBeBackfilled": projectsToBeBackfilled,
			"backfillDone":           backfillDone,
		}, firestore.MergeAll)

	if err != nil {
		return err
	}

	if backfillDone {
		backfillCollection := d.firestoreClient.Collection(d.jobsSinksMetadataCollection).
			Doc(sinkID).
			Collection("backfill")

		bulkWriter := d.firestoreClient.BulkWriter(ctx)
		defer bulkWriter.Flush()

		return deleteCollection(ctx, bulkWriter, backfillCollection)
	}

	return nil
}

func (d *JobsSinksMetadataDal) UpdateSinkProjectProgress(ctx context.Context, sinkID, project string, progress int) error {
	var backfillDone bool

	if progress == 100 {
		backfillDone = true
	}

	_, err := d.firestoreClient.Collection(d.jobsSinksMetadataCollection).
		Doc(sinkID).
		Collection("backfill").
		Doc(project).
		Set(ctx, map[string]interface{}{
			"backfillDone":     backfillDone,
			"backfillProgress": progress,
		}, firestore.MergeAll)

	return err
}

func (d *JobsSinksMetadataDal) GetSinkProjects(ctx context.Context, sinkID string) ([]*domain.ProjectBackfillInfo, error) {
	docs, err := d.firestoreClient.Collection(d.jobsSinksMetadataCollection).
		Doc(sinkID).
		Collection("backfill").
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	var projects []*domain.ProjectBackfillInfo

	for _, doc := range docs {
		var project domain.ProjectBackfillInfo

		if err := doc.DataTo(&project); err != nil {
			return nil, err
		}

		projects = append(projects, &project)
	}

	return projects, nil
}

func (d *JobsSinksMetadataDal) UpdateBackfillForProjectAndDate(ctx context.Context, sinkID, project string, date time.Time, dateBackInfo *domain.DateBackfillInfo) error {
	updateFields := map[string]interface{}{
		"backfillMinCreationTime": dateBackInfo.BackfillMinCreationTime,
		"backfillMaxCreationTime": dateBackInfo.BackfillMaxCreationTime,
		"backfillDone":            dateBackInfo.BackfillDone,
	}

	if dateBackInfo.BackfillDone {
		updateFields["backfillProcessEndTime"] = dateBackInfo.BackfillProcessEndTime
		updateFields["backfillProcessLastUpdateTime"] = dateBackInfo.BackfillProcessLastUpdateTime
	}

	_, err := d.firestoreClient.Collection(d.jobsSinksMetadataCollection).
		Doc(sinkID).
		Collection("backfill").
		Doc(project).
		Collection("days").
		Doc(date.Format("2006-01-02")).
		Set(ctx, updateFields, firestore.MergeAll)

	return err
}

func (d *JobsSinksMetadataDal) GetSinkProjectDates(ctx context.Context, sinkID, backfillProject string) ([]*domain.DateBackfillInfo, error) {
	docs, err := d.firestoreClient.Collection(d.jobsSinksMetadataCollection).
		Doc(sinkID).
		Collection("backfill").
		Doc(backfillProject).
		Collection("days").
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	var dates []*domain.DateBackfillInfo

	for _, doc := range docs {
		var date domain.DateBackfillInfo
		if err := doc.DataTo(&date); err != nil {
			return nil, err
		}

		dates = append(dates, &date)
	}

	return dates, nil
}

func (d *JobsSinksMetadataDal) DeleteSinkMetadata(ctx context.Context, jobID string) error {
	docRef := d.firestoreClient.Collection(d.jobsSinksMetadataCollection).Doc(jobID)

	bulkWriter := d.firestoreClient.BulkWriter(ctx)
	defer bulkWriter.Flush()

	return d.documentsHandler.DeleteDocAndSubCollections(ctx, docRef, bulkWriter)
}

func deleteCollection(ctx context.Context, bulkWriter *firestore.BulkWriter, collRef *firestore.CollectionRef) error {
	it := collRef.DocumentRefs(ctx)
	documentsHandler := doitFS.DocumentHandler{}

	for {
		ref, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
		}

		if err := documentsHandler.DeleteDocAndSubCollections(ctx, ref, bulkWriter); err != nil {
			return err
		}
	}

	return nil
}
