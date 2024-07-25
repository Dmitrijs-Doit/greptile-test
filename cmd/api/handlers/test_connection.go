package handlers

import (
	"net/http"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/test_connection"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

type TestConnectionHandler struct {
	testConnection *test_connection.TestConnection
}

func NewTestConnectionHandler(log logger.Provider, conn *connection.Connection) *TestConnectionHandler {
	testConnection := test_connection.NewTestConnection(log, conn)

	return &TestConnectionHandler{
		testConnection: testConnection,
	}
}

func (t *TestConnectionHandler) TestBillingConnection(ctx *gin.Context) error {
	var body dataStructures.TestBillingConnectionBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	return t.testConnection.TestBillingConnection(ctx, body.BillingAccountID, body.ServiceAccountEmail, &pkg.BillingTableInfo{
		ProjectID: body.ProjectID,
		DatasetID: body.DatasetID,
		TableID:   body.TableID,
	})
}
