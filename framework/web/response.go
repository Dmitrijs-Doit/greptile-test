package web

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/internal"
)

// Respond converts a Go value to JSON and sends it to the client with the corresponded status code.
func Respond(ctx *gin.Context, data interface{}, statusCode int) error {
	v, ok := internal.DataFromContext(ctx)
	if ok {
		v.StatusCode = statusCode
	}

	// If there is nothing to marshal then set status code and return.
	if data == nil || statusCode == http.StatusNoContent {
		ctx.Status(statusCode)
		return nil
	}

	ctx.JSON(statusCode, data)

	return nil
}

// RespondError sends an error response back to the client.
func RespondError(ctx *gin.Context, err error) error {
	if webErr, ok := err.(*Error); ok {
		errResponse := ErrorResponse{
			Error: webErr.Err.Error(),
		}
		if err := Respond(ctx, errResponse, webErr.Status); err != nil {
			return err
		}

		return nil
	}

	errResponse := ErrorResponse{
		Error: http.StatusText(http.StatusInternalServerError),
	}
	if err := Respond(ctx, errResponse, http.StatusInternalServerError); err != nil {
		return err
	}

	return nil
}

func RespondDownloadFile(ctx *gin.Context, data []byte, filename string) error {
	ctx.Header("Content-Disposition", "attachment; filename="+filename)
	ctx.Data(http.StatusOK, "application/octet-stream", data)

	return nil
}
