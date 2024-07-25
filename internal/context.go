package internal

import (
	"time"

	"github.com/gin-gonic/gin"
)

// CtxDataKey is how request values or stored/retrieved.
const CtxDataKey = "app-context"

// Data represent state for each request.
type Data struct {
	TraceID    string
	StatusCode int
	Now        time.Time
}

// ContextWithData sets a gin.Context with context data.
func ContextWithData(ctx *gin.Context, data *Data) {
	ctx.Set(CtxDataKey, data)
}

// DataFromContext retrieves data from gin.Context.
func DataFromContext(ctx *gin.Context) (*Data, bool) {
	v, ok := ctx.Value(CtxDataKey).(*Data)
	return v, ok
}
