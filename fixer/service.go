package fixer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/http"
)

type FixerService struct {
	loggerProvider logger.Provider
	*connection.Connection
	fixerAPI *http.Client
}

const (
	historyTimeSeriesFirestorePath = "app/fixer/fixerExchangeRates"
)

var (
	ErrInvalidDate      = errors.New("invalid date")
	ErrInvalidStartDate = errors.New("invalid start date")
	ErrInvalidEndDate   = errors.New("invalid end date")
	ErrInvalidSymbols   = errors.New("invalid symbols list")
)

var (
	TimeseriesStartDate                     time.Time
	TimeseriesEndDate                       time.Time
	CurrencyHistoricalTimeseriesInitialized bool
	CurrencyHistoricalTimeseries            map[int]map[string]map[string]float64
)

func NewFixerService(log logger.Provider, conn *connection.Connection) (*FixerService, error) {
	ctx := context.Background()

	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretFixer)
	if err != nil {
		return nil, err
	}

	type secretJSON struct {
		BaseURL   string `json:"base_url"`
		AccessKey string `json:"access_key"`
	}

	var secret secretJSON
	if err := json.Unmarshal(data, &secret); err != nil {
		return nil, err
	}

	fixerAPI, err := http.NewClient(ctx, &http.Config{
		BaseURL: secret.BaseURL,
		Headers: map[string]string{
			"Accept": "application/json",
		},
		QueryParams: map[string][]string{
			"access_key": {secret.AccessKey},
		},
	})
	if err != nil {
		return nil, err
	}

	svc := &FixerService{
		log,
		conn,
		fixerAPI,
	}

	if CurrencyHistoricalTimeseriesInitialized {
		return svc, nil
	}

	now := time.Now().UTC()
	TimeseriesStartDate = time.Date(2011, 1, 1, 0, 0, 0, 0, time.UTC)
	TimeseriesEndDate = time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)

	// initialize historical exchange rates from firestore or fixer API
	if err := svc.initHistoricalRates(ctx); err != nil {
		if common.Production {
			return nil, err
		}
	}

	return svc, nil
}

func (s *FixerService) parseSymbols(symbols []Currency) (string, error) {
	if len(symbols) == 0 {
		return "", ErrInvalidSymbols
	}

	v := make([]string, len(symbols))
	for i, symbol := range symbols {
		v[i] = string(symbol)
	}

	return strings.Join(v, ","), nil
}

func (s *FixerService) initHistoricalRates(ctx context.Context) error {
	fs := s.Firestore(ctx)

	docSnaps, err := fs.Collection(historyTimeSeriesFirestorePath).Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("currency time series init failed to load documents: %s", err)
	}

	// If firestore is not initialized with exchange rates yet
	// then fetch it from fixer API and recall this method.
	if len(docSnaps) == 0 {
		if err := s.SyncCurrencyExchangeRateHistory(ctx); err != nil {
			return err
		}

		return s.initHistoricalRates(ctx)
	}

	CurrencyHistoricalTimeseries = make(map[int]map[string]map[string]float64)

	for _, docSnap := range docSnaps {
		var data map[string]map[string]float64
		if err := docSnap.DataTo(&data); err != nil {
			return fmt.Errorf("currency time series init docSnapDataTo failed: %s", err)
		}

		year, err := strconv.Atoi(docSnap.Ref.ID)
		if err != nil {
			return fmt.Errorf("currency time series strconv.Atoi init failed: %s", err)
		}

		CurrencyHistoricalTimeseries[year] = data
	}

	CurrencyHistoricalTimeseriesInitialized = true

	return nil
}
