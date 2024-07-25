package scripts

import (
	"errors"
	"log"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

const (
	saAdminRoleName string = "Flexsave SA Admin"
	saUserRoleName  string = "Flexsave SA User"
)

type FlexsaveSARolesReq struct {
	Project          string  `json:"project"`
	ConsolidatedType string  `json:"consolidatedType"`
	StandaloneType   string  `json:"standaloneType"`
	SAAdminDocID     *string `json:"saAdminDocId"`
	SAAUserDocID     *string `json:"saUserDocId"`
}

// UpdateRolesForFSSA Update all existent non FSSA roles by adding a customerType field equal to consolidatedType.
// Adds two new roles designed for Flexsave SA having customerType = standaloneType.
// Non FSSA roles are all roles having customerType!=standaloneType
// Example payload
//
//	{
//		"project": "doitintl-cmp-dev",
//		"consolidatedType": "consolidated",
//		"standaloneType": "standalone",
//		"saAdminDocId": "123243"                   -- Optional
//		"saUserDocId": "123243"                   -- Optional
//		}
func UpdateRolesForFSSA(ctx *gin.Context) []error {
	var params FlexsaveSARolesReq

	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	if params.Project == "" || params.ConsolidatedType == "" || params.StandaloneType == "" {
		err := errors.New("invalid input parameters")
		return []error{err}
	}

	saAdPermissions := []string{
		"wfDH3k1FmYKHlQBwGIzZ",
		"AnJW2Hwipmucak00yko0",
		"1SmYWoSAO1frHKjt34Gz",
		"sfmBZeLN8uXWooCqJ4NO",
		"8zXuFyohNSiiLy2ZQ6Xu",
		"tvQnB14mSGr8LSU8reYH",
		"HN6A3cPzDBcAIlc3ncDy",
		"dEJbIiUcHn8GhW7IiWLW",
		"AIzQjXTUQDgeZjXqNsgF",
		"jg1YHuQhsRlg5msNhpZZ",
		"ZqLGIVDUhNiSEDtrEb0S",
	}

	saUsPermissions := []string{
		"AnJW2Hwipmucak00yko0",
		"1SmYWoSAO1frHKjt34Gz",
		"sfmBZeLN8uXWooCqJ4NO",
		"8zXuFyohNSiiLy2ZQ6Xu",
		"tvQnB14mSGr8LSU8reYH",
		"HN6A3cPzDBcAIlc3ncDy",
		"dEJbIiUcHn8GhW7IiWLW",
		"jg1YHuQhsRlg5msNhpZZ",
	}

	fs, err := firestore.NewClient(ctx, params.Project)

	if err != nil {
		return []error{err}
	}

	defer fs.Close()

	saAd := map[string]interface{}{
		"customer":     nil,
		"customerType": params.StandaloneType,
		"inUse":        -1,
		"name":         saAdminRoleName,
		"description":  "Admin role for Flexsave standalone",
		"permissions":  getPermissionRefs(fs, saAdPermissions),
		"timeCreated":  firestore.ServerTimestamp,
		"timeModified": firestore.ServerTimestamp,
		"type":         "preset",
	}
	saUs := map[string]interface{}{
		"customer":     nil,
		"customerType": params.StandaloneType,
		"inUse":        -1,
		"name":         saUserRoleName,
		"description":  "User role for Flexsave standalone",
		"permissions":  getPermissionRefs(fs, saUsPermissions),
		"timeCreated":  firestore.ServerTimestamp,
		"timeModified": firestore.ServerTimestamp,
		"type":         "preset",
	}

	batch := fb.NewAutomaticWriteBatch(fs, 500)
	docRef := fs.Collection("roles")

	if errs := updateExistingRoles(ctx, fs, batch, params.ConsolidatedType); len(errs) > 0 {
		return errs
	}

	sAAdminDocID := createRole(ctx, batch, docRef, saAd, params.SAAdminDocID)
	sAAUserDocID := createRole(ctx, batch, docRef, saUs, params.SAAUserDocID)

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs
	}

	log.Printf("\nFlexsave SA admin Role: %s \nFlexsave SA user Role: %s", sAAdminDocID, sAAUserDocID)

	return nil
}

// updateExistingRoles updates all existing roles which aren't SA Roles to be of type consolidatedType
func updateExistingRoles(ctx *gin.Context, fs *firestore.Client, batch *fb.AutomaticWriteBatch, consolidatedType string) []error {
	docSnaps, err := fs.Collection("roles").Documents(ctx).GetAll()

	if err != nil {
		return []error{err}
	}

	if len(docSnaps) > 0 {
		for _, docSnap := range docSnaps {
			name, ok := docSnap.Data()["name"].(string)
			if ok {
				if name == saAdminRoleName || name == saUserRoleName {
					log.Printf("%s role already exists", name)
					continue
				}
			}

			batch.Update(docSnap.Ref, []firestore.Update{
				{FieldPath: []string{"customerType"}, Value: consolidatedType},
				{FieldPath: []string{"timeModified"}, Value: firestore.ServerTimestamp},
			})
		}
	}

	return nil
}

// createRole creates a Flexsave standalone Role based on a given id or updates it if existent. If no ID is present then a new Role with random ID will be added
func createRole(ctx *gin.Context, batch *fb.AutomaticWriteBatch, docRef *firestore.CollectionRef, roleData map[string]interface{}, roleID *string) string {
	if roleID == nil {
		newDoc := docRef.NewDoc()
		batch.Create(newDoc, roleData)

		return newDoc.ID
	}

	docSnap, err := docRef.Doc(*roleID).Get(ctx)

	if err != nil {
		batch.Set(docRef.Doc(*roleID), roleData)
		return *roleID
	}

	if docSnap.Data()["name"] != roleData["name"] {
		log.Printf("\nRole %s is not a %s", *roleID, roleData["name"])
		return ""
	}

	batch.Update(docRef.Doc(*roleID), []firestore.Update{
		{FieldPath: []string{"customer"}, Value: roleData["customer"]},
		{FieldPath: []string{"customerType"}, Value: roleData["customerType"]},
		{FieldPath: []string{"inUse"}, Value: roleData["inUse"]},
		{FieldPath: []string{"name"}, Value: roleData["name"]},
		{FieldPath: []string{"permissions"}, Value: roleData["permissions"]},
		{FieldPath: []string{"timeModified"}, Value: roleData["timeModified"]},
		{FieldPath: []string{"type"}, Value: roleData["type"]},
	})

	return *roleID
}

func getPermissionRefs(fs *firestore.Client, permissions []string) []*firestore.DocumentRef {
	refs := make([]*firestore.DocumentRef, len(permissions))
	for i, pID := range permissions {
		refs[i] = fs.Collection("permissions").Doc(pID)
	}

	return refs
}
