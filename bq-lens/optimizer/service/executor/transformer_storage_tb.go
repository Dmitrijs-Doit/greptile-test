package executor

import (
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	firestoremodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TransformTableStorageTB(data []bqmodels.TableStorageTBResult, now time.Time) (dal.RecommendationSummary, error) {
	document := firestoremodels.TableStorageTBDocument{}

	for _, d := range data {
		key := fmt.Sprintf("%s:%s.%s", d.ProjectID, d.DatasetID, d.TableID)
		document[key] = firestoremodels.TableStorageTB{
			ProjectID:          d.ProjectID,
			DatasetID:          d.DatasetID,
			TableID:            d.TableID,
			StorageTB:          nullORFloat64(d.StorageTB),
			ShortTermStorageTB: nullORFloat64(d.ShortTermStorageTB),
			LongTermStorageTB:  nullORFloat64(d.LongTermStorageTB),
			LastUpdate:         now,
		}
	}

	return dal.RecommendationSummary{
		bqmodels.TableStorageTB: {
			bqmodels.TimeRangeMonth: document,
			bqmodels.TimeRangeWeek:  document,
			bqmodels.TimeRangeDay:   document,
		},
	}, nil
}

func TransformDatasetStorageTB(data []bqmodels.DatasetStorageTBResult, now time.Time) (dal.RecommendationSummary, error) {
	document := firestoremodels.DatasetStorageTBDocument{}

	for _, d := range data {
		key := fmt.Sprintf("%s:%s", d.ProjectID, d.DatasetID)
		document[key] = firestoremodels.DatasetStorageTB{
			ProjectID:          d.ProjectID,
			DatasetID:          d.DatasetID,
			StorageTB:          nullORFloat64(d.StorageTB),
			ShortTermStorageTB: nullORFloat64(d.ShortTermStorageTB),
			LongTermStorageTB:  nullORFloat64(d.LongTermStorageTB),
			LastUpdate:         now,
		}
	}

	return dal.RecommendationSummary{
		bqmodels.DatasetStorageTB: {
			bqmodels.TimeRangeMonth: document,
			bqmodels.TimeRangeWeek:  document,
			bqmodels.TimeRangeDay:   document,
		},
	}, nil
}

func TransformProjectStorageTB(data []bqmodels.ProjectStorageTBResult, now time.Time) (dal.RecommendationSummary, error) {
	document := firestoremodels.ProjectStorageTBDocument{}

	for _, d := range data {
		key := d.ProjectID
		document[key] = firestoremodels.ProjectStorageTB{
			ProjectID:          d.ProjectID,
			StorageTB:          nullORFloat64(d.StorageTB),
			ShortTermStorageTB: nullORFloat64(d.ShortTermStorageTB),
			LongTermStorageTB:  nullORFloat64(d.LongTermStorageTB),
			LastUpdate:         now,
		}
	}

	return dal.RecommendationSummary{
		bqmodels.ProjectStorageTB: {
			bqmodels.TimeRangeMonth: document,
			bqmodels.TimeRangeWeek:  document,
			bqmodels.TimeRangeDay:   document,
		},
	}, nil
}

func nullORFloat64(f bigquery.NullFloat64) *float64 {
	var value *float64

	if f.Valid {
		return &f.Float64
	}

	return value
}
