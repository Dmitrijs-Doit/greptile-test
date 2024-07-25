package mixpanel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/http"
	"github.com/doitintl/mixpanel"
	"github.com/doitintl/tests"
)

func NewMockActiveUsersReportService(t *testing.T) *ActiveUsersReportService {
	ctx := context.Background()
	log, err := logger.NewLogging(ctx)

	if err != nil {
		t.Error(err)
	}

	conn, err := connection.NewConnection(ctx, log)
	if err != nil {
		t.Error(err)
	}

	c, err := http.NewClient(ctx, &http.Config{
		BaseURL: "/",
		Timeout: 0,
	})
	if err != nil {
		t.Error(err)
	}

	mixpanelService := mixpanel.NewServiceWithClients(c, c, c)

	return &ActiveUsersReportService{
		log,
		conn,
		mixpanelService,
	}
}

func TestGetActiveUsersReportCached(t *testing.T) {
	ctx := context.Background()
	s := NewMockActiveUsersReportService(t)

	if err := tests.LoadTestData("Mixpanel"); err != nil {
		t.Error(err)
	}

	// get active users cached report version
	r, err := s.GetActiveUsersReport(ctx, "ImoC9XkrutBysJvyqlBm", nil)
	if err != nil {
		t.Error(err)
	}

	t.Log(len(r.Data.Series))

	assert.Equal(t, len(r.Data.Series), 31)
}

func TestGetActiveUsersReportWithoutCached(t *testing.T) {
	ctx := context.Background()
	s := NewMockActiveUsersReportService(t)

	if err := tests.LoadTestData("Mixpanel"); err != nil {
		t.Error(err)
	}

	// get real time active users report
	_, err := s.GetActiveUsersReport(ctx, "EE8CtpzYiKp0dVAESVrB", nil)
	assert.EqualError(t, err, "Get \"/segmentation\": unsupported protocol scheme \"\"")
}

func TestBuildActiveUsersReportConfig(t *testing.T) {
	ctx := context.Background()
	s := NewMockActiveUsersReportService(t)

	// check report config start date valid
	rc := s.BuildActiveUsersReportConfig(ctx, "ImoC9XkrutBysJvyqlBm")

	now := time.Now()
	lastMonth := now.AddDate(0, -1, 0)

	assert.Equal(t, rc["from_date"][0], lastMonth.Format("2006-01-02"))
}
