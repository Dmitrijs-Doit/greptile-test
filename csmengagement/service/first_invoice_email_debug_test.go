//go:build integration
// +build integration

package service

import (
	"context"
	"testing"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
)

func TestService_SendFirstInvoiceEmailDebug(t *testing.T) {
	gCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx := context.Background()
	fs := common.GetFirestoreClient(gCtx)
	s := NewService(ctx, fs, nil)

	if err := s.SendFirstInvoiceEmail(ctx, "2LAIuL6mqexozk2pM1qk"); err != nil {
		t.Fatal("err")
	}
}
