package rowsvalidator

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/rowsvalidator/segments"
	"google.golang.org/api/iterator"
)

var _ RowsCounter = (*RowsCounterImpl)(nil)

type RowsCounterImpl struct {
	loggerProvider        logger.Provider
	getRowsCountQueryFunc func(table *common.TableInfo, billingAccountID string, segment *segments.Segment) (string, segments.SegmentLength, error)
}

func NewRowsCounter(loggerProvider logger.Provider, getRowsCountQueryFunc func(table *common.TableInfo, billingAccountID string, segment *segments.Segment) (string, segments.SegmentLength, error)) *RowsCounterImpl {
	return &RowsCounterImpl{
		loggerProvider,
		getRowsCountQueryFunc,
	}
}

func (rs *RowsCounterImpl) GetRowsCount(ctx context.Context, bq *bigquery.Client, table *common.TableInfo, billingAccountID string, segment *segments.Segment) (map[segments.HashableSegment]int, error) {
	logger := rs.loggerProvider(ctx)

	query, segmentLength, err := rs.getRowsCountQueryFunc(table, billingAccountID, segment)
	if err != nil {
		return nil, err
	}

	type row struct {
		TimeStamp time.Time `bigquery:"time_stamp"`
		RowsCount int       `bigquery:"rows_count"`
	}

	allRows := []row{}
	rowsCount := make(map[segments.HashableSegment]int)

	rows, err := bq.Query(query).Read(ctx)
	if err != nil {
		logger.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
		return nil, err
	}

	for {
		var r row

		err = rows.Next(&r)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		allRows = append(allRows, r)
	}

	for _, r := range allRows {
		startTime := r.TimeStamp
		endTime := r.TimeStamp

		switch segmentLength {
		case segments.SegmentLengthHour:
			endTime = endTime.Add(time.Hour)
		case segments.SegmentLengthDay:
			endTime = endTime.Add(24 * time.Hour)
		case segments.SegmentLengthMonth:
			endTime = endTime.AddDate(0, 1, 0)
		}

		rowsCount[segments.HashableSegment{
			StartTime: startTime,
			EndTime:   endTime,
		}] = r.RowsCount
	}

	return rowsCount, nil
}
