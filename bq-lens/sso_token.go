package bqlens

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func InvokeBigQueryProcess(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	email := ctx.GetString("email")

	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	clientID := ctx.Request.URL.Query().Get("client")
	needToDelete := ctx.Request.URL.Query().Get("delete")

	l.Infof("clientID: %s, needToDelete: %s", clientID, needToDelete)

	removeData := false
	if needToDelete == "true" {
		removeData = true
	}

	// trigger BQ Lens onboarding
	err := TriggerBQLensProcess(ctx, clientID, removeData)
	if err != nil {
		l.Error(fmt.Sprintf("Error triggering BQ Lens onboarding: %s", err))
		ctx.AbortWithError(http.StatusInternalServerError, err)
	}
}
