package scripts

import (
	"encoding/json"
	"errors"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
)

type UpdateCloudConnectPermissionsInput struct {
	ProjectID string `json:"project_id"`
}

// updateCloudConnectPermissions will generate the /app/cloud-connect document in firestore.
func updateCloudConnectPermissions(ctx *gin.Context) []error {
	var params UpdateCloudConnectPermissionsInput

	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	if params.ProjectID == "" {
		err := errors.New("missing project id")
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, params.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	data, err := os.ReadFile("./scripts/data/cloudconnect_permissions.json")
	if err != nil {
		return []error{err}
	}

	var permissions map[string]interface{}

	if err := json.Unmarshal(data, &permissions); err != nil {
		return []error{err}
	}

	if _, err := fs.Collection("app").
		Doc("cloud-connect").
		Set(ctx, permissions); err != nil {
		return []error{err}
	}

	return nil
}
