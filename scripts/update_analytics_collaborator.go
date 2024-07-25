package scripts

import (
	"errors"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

type UpdateAnalyticsCollaboratorsInput struct {
	ProjectID  string `json:"project_id"`
	CustomerID string `json:"customer_id"`
	Path       string `json:"path"`
	Emails     []struct {
		NewEmail string `json:"new_email"`
		OldEmail string `json:"old_email"`
	} `json:"emails"`
}

type AnalyticsResource struct {
	Collaborators []collab.Collaborator `firestore:"collaborators"`
}

/*
UpdateAnalyticsCollaborators update emails of analytics resources collaborators, example payload:

	{
	    "project_id": "doitintl-cmp-dev",
		"customer_id": "abcxyz"
	    "path": "dashboards/google-cloud-reports/savedReports",
	    "emails": [
	        {
	            "old_email": "mariana@redislabs.com",
	            "new_email": "mariana@redis.com"
	        }
	    ]
	}
*/
func UpdateAnalyticsCollaborators(ctx *gin.Context) []error {
	var params UpdateAnalyticsCollaboratorsInput
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	if params.ProjectID == "" || params.Path == "" {
		err := errors.New("invalid input parameters")
		return []error{err}
	}

	params.Path = strings.TrimPrefix(params.Path, "/")

	fs, err := firestore.NewClient(ctx, params.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	customerRef := fs.Collection("customers").Doc(params.CustomerID)
	collection := fs.Collection(params.Path)
	batch := fb.NewAutomaticWriteBatch(fs, 250)

	for _, v := range params.Emails {
		oldEmail := v.OldEmail
		newEmail := v.NewEmail

		docSnaps, err := collection.
			Where("customer", "==", customerRef).
			Where("collaborators", common.ArrayContainsAny, []collab.Collaborator{
				{Email: oldEmail, Role: collab.CollaboratorRoleOwner},
				{Email: oldEmail, Role: collab.CollaboratorRoleEditor},
				{Email: oldEmail, Role: collab.CollaboratorRoleViewer},
			}).
			Select("customer", "collaborators").Documents(ctx).GetAll()
		if err != nil {
			return []error{err}
		}

		if len(docSnaps) > 0 {
			for _, docSnap := range docSnaps {
				var r AnalyticsResource
				if err := docSnap.DataTo(&r); err != nil {
					return []error{err}
				}

				for _, c := range r.Collaborators {
					if c.Email == oldEmail {
						newCollaborator := collab.Collaborator{Email: newEmail, Role: c.Role}

						batch.Update(docSnap.Ref, []firestore.Update{
							{FieldPath: []string{"collaborators"}, Value: firestore.ArrayRemove(c)},
						})
						batch.Update(docSnap.Ref, []firestore.Update{
							{FieldPath: []string{"collaborators"}, Value: firestore.ArrayUnion(newCollaborator)},
						})
					}

					break
				}
			}
		}
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs
	}

	return nil
}
