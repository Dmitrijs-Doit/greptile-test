package externalreport

import (
	"errors"

	"golang.org/x/net/context"

	doitFirestore "github.com/doitintl/firestore"
)

func (s *Service) hasDatahubMetrics(ctx context.Context, customerID string) (bool, error) {
	datahubMetrics, err := s.datahubMetricDAL.Get(ctx, customerID)
	if err != nil {
		if errors.Is(err, doitFirestore.ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	return datahubMetrics != nil && len(datahubMetrics.Metrics) > 0, nil
}
