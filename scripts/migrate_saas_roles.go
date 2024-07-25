package scripts

import (
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

type migrateSaasRolesReq struct {
	Commit bool `json:"commit"`
}

// MigrateSaasRoles moves customers with the current saas admin and saas user roles to
// the regular admin and user roles
func MigrateSaasRoles(ctx *gin.Context) []error {
	logging, err := logger.NewLogging(ctx)
	if err != nil {
		return []error{err}
	}

	var req migrateSaasRolesReq
	if err := ctx.BindJSON(&req); err != nil {
		return []error{err}
	}

	log := logging.Logger(ctx)

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}

	defer fs.Close()

	adminRole := fs.Collection("roles").Doc(string(common.PresetRoleAdmin))
	userRole := fs.Collection("roles").Doc(string(common.PresetRoleStandardUser))
	// Using string id's because these roles 'should' be deleted from the database
	saasAdminRole := fs.Collection("roles").Doc("4x4AeO6fprj8iYjnFoDq")
	saasUserRole := fs.Collection("roles").Doc("GjM9ZwIyGeuXfBraOl47")

	iter := fs.Collection("users").
		Where("roles", common.ArrayContainsAny, []*firestore.DocumentRef{
			saasUserRole, saasAdminRole,
		}).Documents(ctx)

	batch := fs.BulkWriter(ctx)
	userErrs := []error{}

	for {
		doc, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}

			return []error{err}
		}

		var user common.User
		if err := doc.DataTo(&user); err != nil {
			userErrs = append(userErrs, fmt.Errorf("SKIPPED user %s: error unmarshalling user from firestore: %w", doc.Ref.ID, err))
			continue
		}

		for i, role := range user.Roles {
			switch role.Path {
			case saasUserRole.Path:
				user.Roles[i] = userRole
			case saasAdminRole.Path:
				user.Roles[i] = adminRole
			}
		}

		if !req.Commit {
			log.Infof("user %s roles would be updated", doc.Ref.ID)
			continue
		}

		_, _ = batch.Update(doc.Ref, []firestore.Update{{
			FieldPath: []string{"roles"},
			Value:     user.Roles,
		}})
	}

	batch.End()

	return userErrs
}
