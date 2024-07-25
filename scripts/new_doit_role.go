package scripts

import (
	"errors"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
)

// Request Params..
type DoiTRoleCreateParams struct {
	Project     string   `json:"project"`
	RoleName    string   `json:"roleName"`
	Description *string  `json:"description"`
	Users       []string `json:"users"`
	DocID       *string  `json:"docID"`
}

// Adds a new DoiT Employess role to the collection at app/doit-employees/doitRoles
// Example request...
//
//	{
//		"project": "doitintl-cmp-dev",
//		"roleName": "example-role",
//		"users": ["mr-user@doit-intl.com"], -- Optional
//		"docID": "123243"                   -- Optional
//		}
func NewDoiTRole(ctx *gin.Context) []error {
	var params DoiTRoleCreateParams
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	if params.Project == "" || params.RoleName == "" {
		err := errors.New("invalid input parameters")
		return []error{err}
	}

	collectionPath := "app/doit-employees/doitRoles"

	fs, err := firestore.NewClient(ctx, params.Project)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	if params.DocID == nil {
		_, _, err = fs.Collection(collectionPath).Add(ctx, map[string]interface{}{
			"description": &params.Description,
			"roleName":    params.RoleName,
			"users":       params.Users,
		})
	} else {
		_, err = fs.Collection(collectionPath).Doc(*params.DocID).Set(ctx, map[string]interface{}{
			"description": &params.Description,
			"roleName":    params.RoleName,
			"users":       params.Users,
		})
	}

	if err != nil {
		return []error{err}
	}

	return nil
}
