package fixer

import (
	"context"
	"time"
)

func (s *FixerService) SyncCurrencyExchangeRateHistory(ctx context.Context) error {
	logger := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	now := time.Now().UTC()
	startDate := TimeseriesStartDate
	endDate := time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC)
	batch := fs.Batch()

	for startDate.Before(endDate) {
		e := startDate.AddDate(1, 0, -1)
		input := TimeseriesInput{
			Base:      USD,
			Symbols:   Currencies,
			StartDate: &startDate,
			EndDate:   &e,
		}

		result, err := s.Timeseries(ctx, &input)
		if err != nil {
			logger.Errorf("fixer currency timeseries sync error: %s", err)
			return err
		}

		if result.Success {
			docRef := fs.Collection(historyTimeSeriesFirestorePath).Doc(startDate.Format("2006"))
			batch.Set(docRef, result.Rates)
		} else {
			logger.Errorf("fixer currency timeseries unsuccessful: %+v", result)
			return err
		}

		startDate = startDate.AddDate(1, 0, 0)
	}

	if _, err := batch.Commit(ctx); err != nil {
		return err
	}

	return nil
}
