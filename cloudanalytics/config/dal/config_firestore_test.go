package dal

import (
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/config"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/tests"
)

func NewConfigsFirestoreWithClientMock(ctx context.Context) *ConfigsFirestore {
	logging, err := logger.NewLogging(ctx)
	if err != nil {
		log.Printf("main: could not initialize logging. error %s", err)
		return nil
	}

	// Initialize db connections clients
	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		log.Printf("main: could not initialize db connections. error %s", err)
		return nil
	}

	return NewConfigsFirestoreWithClient(conn.Firestore)
}

func TestGetExtendedMetrics(t *testing.T) {
	ctx := context.Background()
	s := NewConfigsFirestoreWithClientMock(ctx)

	_, err := s.GetExtendedMetrics(ctx)
	assert.Error(t, err)

	if err := tests.LoadTestData("Configs"); err != nil {
		t.Error(err)
	}

	extendedMetrics, err := s.GetExtendedMetrics(ctx)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, extendedMetrics[0], config.ExtendedMetric{
		Key:        "flexsave_waste_cost",
		Label:      "Flexsave Waste",
		Type:       "cost",
		Visibility: "csp",
	})
	assert.Equal(t, extendedMetrics[1], config.ExtendedMetric{
		Key:        "flexsave_net_revenue",
		Label:      "Flexsave Net Revenue",
		Type:       "cost",
		Visibility: "csp",
	})
}
