package common

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

func ValidateRegexp(ctx *gin.Context) {
	type body struct {
		Regexp string `json:"regexp"`
	}

	var b body
	if err := ctx.ShouldBindJSON(&b); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if _, err := regexp.Compile(b.Regexp); err != nil {
		ctx.Status(http.StatusBadRequest)
	}
}
