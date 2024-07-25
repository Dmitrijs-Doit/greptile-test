package scripts

import (
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	"github.com/doitintl/hello/scheduled-tasks/priority"
)

type PriorityCompanies struct {
	Companies []priority.CompanyInfo `firestore:"companies"`
}

func UpdatePriorityCompanies(ctx *gin.Context) []error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	if _, err := fs.Collection("app").Doc("priority-v2").Set(ctx, PriorityCompanies{
		Companies: priority.CompaniesDetails,
	}); err != nil {
		return []error{err}
	}

	return nil
}
