package googlecloud

import (
	"context"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/tests"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	exitCode := m.Run()
	os.Exit(exitCode)
}

func NewGoogleCloudServiceMock(ctx context.Context) (*GoogleCloudService, error) {
	logging, err := logger.NewLogging(ctx)
	if err != nil {
		log.Printf("main: could not initialize logging. error %s", err)
		return nil, err
	}

	// Initialize db connections clients
	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		log.Printf("main: could not initialize db connections. error %s", err)
		return nil, err
	}

	loggerProvider := func(ctx context.Context) logger.ILogger {
		return &loggerMocks.ILogger{}
	}

	return NewGoogleCloudService(loggerProvider, conn), nil
}

// Test for invalid user
// without manage settings permission or doit employee
func TestStopInstanceNoPermission(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	testGoogleCloudService, err := NewGoogleCloudServiceMock(ctx)
	if err != nil {
		t.Error(err)
	}

	// test for doit employee
	_, err = testGoogleCloudService.StopInstance(ctx, ReqBody{
		IsDoitEmployee: true,
		CustomerID:     "JhV7WydpOlW8DeVRVVNf",
	})

	assert.Equal(t, err.Error(), ErrorNoPermission.Error())

	// test for users
	_, err = testGoogleCloudService.StopInstance(ctx, ReqBody{
		IsDoitEmployee: false,
		CustomerID:     "JhV7WydpOlW8DeVRVVNf",
	})

	assert.Equal(t, err.Error(), ErrorNoPermission.Error())
}

// Test for valid user and customer with invalid service account
func TestStopInstanceNoSA(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	if err := tests.LoadTestData("Rightsizing"); err != nil {
		t.Error(err)
	}

	testGoogleCloudService, err := NewGoogleCloudServiceMock(ctx)
	if err != nil {
		t.Error(err)
	}

	_, err = testGoogleCloudService.StopInstance(ctx, ReqBody{
		IsDoitEmployee: false,
		CustomerID:     "EE8CtpzYiKp0dVAESVrB",
		UserID:         "9iWuAK1IPhMub29Y0XMs",
	})

	assert.Equal(t, err.Error(), ErrorGeneric.Error())
}

func TestStartInstance(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	if err := tests.LoadTestData("Rightsizing"); err != nil {
		t.Error(err)
	}

	testGoogleCloudService, err := NewGoogleCloudServiceMock(ctx)
	if err != nil {
		t.Error(err)
	}

	_, err = testGoogleCloudService.StartInstance(ctx, ReqBody{
		IsDoitEmployee: false,
		CustomerID:     "EE8CtpzYiKp0dVAESVrB",
		UserID:         "9iWuAK1IPhMub29Y0XMs",
	})

	assert.Equal(t, err.Error(), ErrorNoInstanceFound.Error())
}

func TestIsServiceEnabled(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	var cred common.GoogleCloudCredential
	ok := isServiceEnabled(ctx, cred, 2323242)

	assert.Equal(t, ok, false)
}

func TestIsRegionUsed(t *testing.T) {
	regionUsed := []*compute.Quota{
		&compute.Quota{Usage: 0},
		&compute.Quota{Usage: -1},
		&compute.Quota{Usage: 1},
	}

	isUsed := isRegionUsed(regionUsed)
	assert.Equal(t, isUsed, true)

	regionNotUsed := []*compute.Quota{
		&compute.Quota{Usage: 0},
		&compute.Quota{Usage: 0},
	}

	isUsed = isRegionUsed(regionNotUsed)
	assert.Equal(t, isUsed, false)
}
