package dal

import (
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/tests"
)

func NewCreditsFirestoreWithClientMock(ctx context.Context) *CreditsFirestore {
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

	return NewCreditsFirestoreWithClient(conn.Firestore)
}

func TestGetCredits(t *testing.T) {
	ctx := context.Background()
	s := NewCreditsFirestoreWithClientMock(ctx)

	if err := tests.LoadTestData("Credits"); err != nil {
		t.Error(err)
	}

	result, err := s.GetCredits(ctx)
	if err != nil {
		t.Error(err)
	}

	for key, val := range result {
		assert.Equal(t, key.ID, "32mKVvOICSRAD4D6l5Wl")
		assert.Equal(t, val.Type, "google-cloud")
		assert.Equal(t, val.Utilization, map[string]map[string]float64{
			"2021-09": {
				"01DB4B-A012D3-1A1A05":          15237.252604749987,
				"01DB4B-A012D3-1A1A05-discount": 1262.7473952499995,
			},
		})
	}
}
