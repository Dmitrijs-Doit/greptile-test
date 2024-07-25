package logger

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLoggingDoesNotThrowErrorWhenNoHeader(t *testing.T) {
	var testLogger *Logger

	var testErr error

	handler := func(w http.ResponseWriter, r *http.Request) {
		testLogger.Info("hello world")
		io.WriteString(w, "<html><body>Hello World!</body></html>")
	}

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	testLogger, testErr = NewLogger(ctx)

	handler(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	testLogger.Infof("testing... %v", body)
	assert.Equal(t, 200, resp.StatusCode)
	assert.NoError(t, testErr)
}

func TestLoggingDoesNotThrowErrorWhenHeaderIsSupplied(t *testing.T) {
	var testLogger *Logger

	var testErr error

	handler := func(w http.ResponseWriter, r *http.Request) {
		testLogger.Info("hello world")
		io.WriteString(w, "<html><body>Hello World!</body></html>")
	}

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req.Header = make(map[string][]string)
	req.Header.Set("X-Cloud-Trace-Context", "test987/uselessSuffix?")

	w := httptest.NewRecorder()

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	testLogger, testErr = NewLogger(ctx)

	handler(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	testLogger.Infof("testing... %v", body)
	assert.Equal(t, 200, resp.StatusCode)
	assert.NoError(t, testErr)
}
