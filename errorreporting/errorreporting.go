package errorreporting

import (
	"context"
	"log"
	"net/http"

	"cloud.google.com/go/errorreporting"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/gin-gonic/gin"
)

var erc *errorreporting.Client

type Metadata struct {
	Req   *http.Request
	User  string
	Stack []byte
}

func init() {
	var err error

	ctx := context.Background()
	erc, err = errorreporting.NewClient(ctx, common.ProjectID, errorreporting.Config{
		ServiceName:    common.GAEService,
		ServiceVersion: common.GAEVersion,
	})

	if err != nil {
		log.Fatalln(err)
	}
}

func Report(err error, md *Metadata) {
	if err == nil || common.IsLocalhost {
		return
	}

	e := errorreporting.Entry{
		Error: err,
	}

	if md != nil {
		e.User = md.User
		e.Req = md.Req
		e.Stack = md.Stack
	}

	erc.Report(e)
}

func ReportRequestError(ctx *gin.Context, err error) {
	Report(err, &Metadata{
		Req: ctx.Request,
	})
}

func AbortWithErrorReport(ctx *gin.Context, code int, err error) {
	Report(err, &Metadata{
		User: ctx.GetString("email"),
		Req:  ctx.Request,
	})

	ctx.AbortWithError(code, err)
}
