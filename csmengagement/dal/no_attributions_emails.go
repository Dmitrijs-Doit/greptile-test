package dal

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	atrDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const noAttrsCollection = "noAttributionsEmails"
const csmEngagementCollection = "csmEngagement"

type INoAttributionsEmail interface {
	GetAllCustomAttributions(ctx context.Context) ([]attribution.Attribution, error)
	GetEligibleUsersForCustomer(ctx context.Context, customerRef *firestore.DocumentRef) ([]*common.User, error)
	GetRequiredRolePermissions(ctx context.Context, roleName string) ([]*firestore.DocumentRef, error)
	GetAttributionsForUser(ctx context.Context, coll collab.Collaborator) ([]*firestore.DocumentRef, error)
	UserHasRequiredPermissions(ctx context.Context, user *common.User, requiredPermissions []common.Permission) error
	GetCustomersNewerThanThirtyDays(ctx context.Context) ([]*firestore.DocumentSnapshot, error)
	GetTracker() SentNotificationsTracker
}

type NoAttributionsEmail struct {
	fs *firestore.Client
}

func NewNoAttributionsEmail(fs *firestore.Client) *NoAttributionsEmail {
	return &NoAttributionsEmail{
		fs: fs,
	}
}

func (a *NoAttributionsEmail) GetTracker() SentNotificationsTracker {
	return &FsTracker{
		DocRef: a.fs.Collection(csmEngagementCollection).Doc(noAttrsCollection),
	}
}
func (a *NoAttributionsEmail) GetCustomersNewerThanThirtyDays(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	customerSnaps, err := a.fs.Collection("customers").Where("timeCreated", ">=", thirtyDaysAgo).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	return customerSnaps, nil
}

func (a *NoAttributionsEmail) GetAllCustomAttributions(ctx context.Context) ([]attribution.Attribution, error) {
	docs, err := a.fs.Collection(atrDal.AttributionsCollection).Where(
		"type", "==", "custom",
	).Documents(ctx).GetAll()

	if err != nil {
		return nil, err
	}

	atrs := make([]attribution.Attribution, len(docs))

	for i, doc := range docs {
		atr := &attribution.Attribution{}
		if err := doc.DataTo(atr); err != nil {
			return nil, err
		}

		if atr.Customer == nil {
			continue
		}

		atrs[i] = *atr
	}

	return atrs, nil
}

func (a *NoAttributionsEmail) GetEligibleUsersForCustomer(ctx context.Context, customerRef *firestore.DocumentRef) ([]*common.User, error) {
	userSnaps, err := a.fs.Collection("users").Where("customer.ref", "==", customerRef).Documents(ctx).GetAll()
	if err != nil {

		return nil, err
	}

	users := make([]*common.User, 0)

	for _, userDocSnap := range userSnaps {
		var user *common.User
		if err := userDocSnap.DataTo(&user); err != nil {
			continue
		}

		users = append(users, user)

	}
	return users, nil
}

func (a *NoAttributionsEmail) GetRequiredRolePermissions(ctx context.Context, roleName string) ([]*firestore.DocumentRef, error) {
	roleSnap, err := a.fs.Collection("roles").Where("name", "==", roleName).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var permissions common.Role

	if len(roleSnap) > 0 {

		if err := roleSnap[0].DataTo(&permissions); err != nil {
			return nil, err
		}

		return permissions.Permissions, nil

	}

	return nil, fmt.Errorf("role %s not found", roleName)
}

func (a *NoAttributionsEmail) GetAttributionsForUser(ctx context.Context, coll collab.Collaborator) ([]*firestore.DocumentRef, error) {
	atrSnaps, err := a.fs.Collection(atrDal.AttributionsCollection).Where(
		"collaborators", "array-contains", coll,
	).Documents(ctx).GetAll()

	if err != nil {
		return nil, err
	}

	atrRefs := make([]*firestore.DocumentRef, len(atrSnaps))

	for i, doc := range atrSnaps {
		atrRefs[i] = doc.Ref
	}

	return atrRefs, nil
}

func (a *NoAttributionsEmail) UserHasRequiredPermissions(ctx context.Context, user *common.User, requiredPermissions []common.Permission) error {
	return user.HasRequiredPermissions(ctx, requiredPermissions)
}
