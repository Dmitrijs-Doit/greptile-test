package executor

import (
	"fmt"
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	firestoremodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TransformTableStoragePrice(
	timeRange bqmodels.TimeRange,
	data []bqmodels.TableStoragePriceResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	document := firestoremodels.TableStoragePriceDocument{}

	for _, d := range data {
		key := fmt.Sprintf("%s:%s.%s", d.ProjectID, d.DatasetID, d.TableID)
		document[key] = firestoremodels.TableStoragePrice{
			ProjectID:             d.ProjectID,
			DatasetID:             d.DatasetID,
			TableID:               d.TableID,
			LongTermStoragePrice:  nullORFloat64(d.LongTermStoragePrice),
			ShortTermStoragePrice: nullORFloat64(d.ShortTermStoragePrice),
			StoragePrice:          nullORFloat64(d.StoragePrice),
			LastUpdate:            now,
		}
	}

	return dal.RecommendationSummary{bqmodels.TableStoragePrice: {timeRange: document}}, nil
}

func TransformDatasetStoragePrice(
	timeRange bqmodels.TimeRange,
	data []bqmodels.DatasetStoragePriceResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	document := firestoremodels.DatasetStoragePriceDocument{}

	for _, d := range data {
		key := fmt.Sprintf("%s:%s", d.ProjectID, d.DatasetID)
		document[key] = firestoremodels.DatasetStoragePrice{
			ProjectID:             d.ProjectID,
			DatasetID:             d.DatasetID,
			LongTermStoragePrice:  nullORFloat64(d.LongTermStoragePrice),
			ShortTermStoragePrice: nullORFloat64(d.ShortTermStoragePrice),
			StoragePrice:          nullORFloat64(d.StoragePrice),
			LastUpdate:            now,
		}
	}

	return dal.RecommendationSummary{bqmodels.DatasetStoragePrice: {timeRange: document}}, nil
}

func TransformProjectStoragePrice(
	timeRange bqmodels.TimeRange,
	data []bqmodels.ProjectStoragePriceResult,
	now time.Time,
) (dal.RecommendationSummary, error) {
	document := firestoremodels.ProjectStoragePriceDocument{}

	for _, d := range data {
		key := d.ProjectID
		document[key] = firestoremodels.ProjectStoragePrice{
			ProjectID:             d.ProjectID,
			LongTermStoragePrice:  nullORFloat64(d.LongTermStoragePrice),
			ShortTermStoragePrice: nullORFloat64(d.ShortTermStoragePrice),
			StoragePrice:          nullORFloat64(d.StoragePrice),
			LastUpdate:            now,
		}
	}

	return dal.RecommendationSummary{bqmodels.ProjectStoragePrice: {timeRange: document}}, nil
}
