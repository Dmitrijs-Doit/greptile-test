package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

const (
	superQueryRole = "7GNaKah9VcAcYFhU0eZE"
	superQueryPerm = "QiI7JfgDTONkLkvKtWR1"
)

// removeSuperQueryPermission removes deprecated SuperQuery role and permission
func removeSuperQueryPermission(ctx *gin.Context) []error {
	logging, err := logger.NewLogging(ctx)
	if err != nil {
		return []error{err}
	}

	log := logging.Logger(ctx)
	fs, err := firestore.NewClient(ctx, common.ProjectID)

	if err != nil {
		log.Errorf("new firestore client %s", err)
		return []error{err}
	}

	if errs := replaceUserRole(ctx, fs, superQueryRole, string(common.PresetRoleSupportUser), log); errs != nil {
		log.Errorf("replace user roles %s", err)
		return errs
	}

	if errs := removePermissionFromRoles(ctx, fs, superQueryPerm, log); errs != nil {
		log.Errorf("remove permissions from roles %s", err)
		return errs
	}

	if err = removeRole(ctx, fs, superQueryRole, log); err != nil {
		log.Errorf("remove role %s", err)
		return []error{err}
	}

	if err = removePermission(ctx, fs, superQueryPerm, log); err != nil {
		log.Errorf("remove permission %s", err)
		return []error{err}
	}

	return nil
}

func replaceUserRole(ctx *gin.Context, fs *firestore.Client, oldRole, newRole string, logger logger.ILogger) []error {
	oldRoleRef := fs.Collection("roles").Doc(oldRole)
	newRoleRef := fs.Collection("roles").Doc(newRole)

	logger.Printf("oldRoleRef %v, newRoleRef %v", oldRoleRef, newRoleRef)

	usersWithOldRole, err := fs.Collection("users").
		Where("roles", "array-contains", oldRoleRef).
		Documents(ctx).
		GetAll()
	if err != nil {
		return []error{err}
	}

	userCount := len(usersWithOldRole)
	logger.Printf("%d users with old role %s were received", userCount, oldRole)

	if userCount == 0 {
		return nil
	}

	batch := fb.NewAutomaticWriteBatch(fs, 250)
	for _, usrSnap := range usersWithOldRole {
		batch.Update(usrSnap.Ref, []firestore.Update{
			{FieldPath: []string{"roles"}, Value: firestore.ArrayUnion(newRoleRef)}})
		batch.Update(usrSnap.Ref, []firestore.Update{
			{FieldPath: []string{"roles"}, Value: firestore.ArrayRemove(oldRoleRef)}})
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs
	}

	logger.Printf("old role %s was replaced with new role %s for %d users", oldRole, newRole, userCount)

	return nil
}

func removePermissionFromRoles(ctx *gin.Context, fs *firestore.Client, perm string, logger logger.ILogger) []error {
	permRef := fs.Collection("permissions").Doc(perm)

	logger.Printf("permRef %v", permRef)

	rolesWithPerm, err := fs.Collection("roles").
		Where("permissions", "array-contains", permRef).
		Documents(ctx).
		GetAll()
	if err != nil {
		return []error{err}
	}

	roleCount := len(rolesWithPerm)
	logger.Printf("%d roles with permission %s were received", roleCount, perm)

	if roleCount == 0 {
		return nil
	}

	batch := fb.NewAutomaticWriteBatch(fs, 250)
	for _, roleSnap := range rolesWithPerm {
		batch.Update(roleSnap.Ref, []firestore.Update{
			{FieldPath: []string{"permissions"}, Value: firestore.ArrayRemove(permRef)}})
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs
	}

	logger.Printf("permission %s was removed from %d roles", perm, roleCount)

	return nil
}

func removeRole(ctx *gin.Context, fs *firestore.Client, role string, logger logger.ILogger) error {
	_, err := fs.Collection("roles").Doc(role).Delete(ctx)
	logger.Printf("role %s was removed", role)

	return err
}

func removePermission(ctx *gin.Context, fs *firestore.Client, perm string, logger logger.ILogger) error {
	_, err := fs.Collection("permissions").Doc(perm).Delete(ctx)
	logger.Printf("permission %s was removed", perm)

	return err
}
