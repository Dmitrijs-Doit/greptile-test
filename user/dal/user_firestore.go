package dal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	userDALIface "github.com/doitintl/hello/scheduled-tasks/user/dal/iface"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	firestoreIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/user/domain"
)

const (
	userCollection          = "users"
	customerCollection      = "customers"
	userNotifications       = "userNotifications"
	userNotificationsBackup = "userNotificationsBackup"
	entitiesCollection      = "entities"
	roles                   = "roles"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrCustomerIsEmpty = errors.New("customer ref cannot be empty")
)

type UserFirestoreDAL struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

func NewUserFirestoreDAL(ctx context.Context, projectID string) (userDALIface.IUserFirestoreDAL, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewUserFirestoreDALWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

func NewUserFirestoreDALWithClient(fun firestoreIface.FirestoreFromContextFun) *UserFirestoreDAL {
	return &UserFirestoreDAL{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *UserFirestoreDAL) userCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(userCollection)
}

func (d *UserFirestoreDAL) customerCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(customerCollection)
}

func (d *UserFirestoreDAL) entitiesCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(entitiesCollection)
}

func (d *UserFirestoreDAL) GetUserByEmail(
	ctx context.Context,
	email string,
	customerID string,
) (*domain.User, error) {
	customerRef := d.customerCollection(ctx).Doc(customerID)
	usersRef := d.userCollection(ctx)

	usersQuery := usersRef.
		Where("email", "==", email).
		Where("customer.ref", "==", customerRef).
		Limit(1)

	usersSnaps, err := usersQuery.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	if len(usersSnaps) == 0 {
		return nil, ErrUserNotFound
	}

	var user domain.User
	if err := usersSnaps[0].DataTo(&user); err != nil {
		return nil, err
	}

	user.ID = usersSnaps[0].Ref.ID

	return &user, nil
}

func (d *UserFirestoreDAL) ListUsers(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	limit int,
) ([]*domain.User, error) {
	if customerRef == nil {
		return nil, ErrCustomerIsEmpty
	}

	query := d.
		firestoreClientFun(ctx).
		Collection(userCollection).
		Where("customer.ref", "==", customerRef)

	if limit > 0 {
		query = query.Limit(limit)
	}

	snaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	users := make([]*domain.User, len(snaps))

	for i, snap := range snaps {
		var user domain.User
		if err := snap.DataTo(&user); err != nil {
			return nil, err
		}

		users[i] = &user
	}

	return users, nil
}

func (d *UserFirestoreDAL) Get(ctx context.Context, id string) (*common.User, error) {
	userRef := d.userCollection(ctx).Doc(id)

	userSnap, err := userRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var user common.User
	if err := userSnap.DataTo(&user); err != nil {
		return nil, err
	}

	user.ID = userSnap.Ref.ID

	return &user, nil
}

func (d *UserFirestoreDAL) GetRef(ctx context.Context, ID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(userCollection).Doc(ID)
}

func (d *UserFirestoreDAL) GetCustomerUsersWithNotifications(ctx context.Context, customerID string, isRestore bool) ([]*pkg.User, error) {
	var users []*pkg.User

	notificationPath := userNotifications
	if isRestore {
		notificationPath = userNotificationsBackup
	}

	customerRef := d.customerCollection(ctx).Doc(customerID) //  d.GetRef(ctx, customerID)

	usersRef := d.firestoreClientFun(ctx).Collection(userCollection).
		Where("customer.ref", "==", customerRef).
		Where(notificationPath, "array-contains-any", []int{1, 2, 3, 4, 5, 6, 7, 8}).
		Documents(ctx)

	usersSnap, err := d.documentsHandler.GetAll(usersRef)
	if err != nil {
		return nil, err
	}

	for _, userSnap := range usersSnap {
		var user pkg.User

		err := userSnap.DataTo(&user)
		if err != nil {
			continue
		}

		user.ID = userSnap.ID()

		users = append(users, &user)
	}

	return users, nil
}

func (d *UserFirestoreDAL) ClearUserNotifications(ctx context.Context, user *pkg.User) error {
	if user.NotificationsBackup != nil {
		return nil
	}

	userRef := d.GetRef(ctx, user.ID)

	_, err := userRef.Update(ctx, []firestore.Update{{Path: userNotifications, Value: []int64{}}, {Path: userNotificationsBackup, Value: user.Notifications}})
	if err != nil {
		return err
	}

	return nil
}

func (d *UserFirestoreDAL) RestoreUserNotifications(ctx context.Context, user *pkg.User) error {
	if len(user.Notifications) > 0 {
		return fmt.Errorf("skipping user %s, notifications are not empty", user.ID)
	}

	userRef := d.GetRef(ctx, user.ID)

	_, err := userRef.Update(ctx, []firestore.Update{{Path: userNotifications, Value: user.NotificationsBackup}, {Path: userNotificationsBackup, Value: firestore.Delete}})
	if err != nil {
		return err
	}

	return nil
}

func (d *UserFirestoreDAL) GetCustomerUsersWithInvoiceNotification(ctx context.Context, customerID, entityID string) ([]*common.User, error) {
	var users []*common.User

	customerRef := d.customerCollection(ctx).Doc(customerID)

	userDocQuery := d.userCollection(ctx).
		Where("customer.ref", "==", customerRef)

	if entityID != "" {
		entityRef := d.entitiesCollection(ctx).Doc(entityID)
		userDocQuery = userDocQuery.Where("entities", "array-contains", entityRef)
	}

	userDocSnaps, err := userDocQuery.Documents(ctx).GetAll()
	if err != nil {
		return users, err
	}

	for _, docSnap := range userDocSnaps {
		var user common.User
		if err := docSnap.DataTo(&user); err != nil {
			continue
		}

		if user.Email == "" || !user.HasInvoicesPermission(ctx) || !user.NotificationsInvoicesEnabled() {
			continue
		}

		users = append(users, &user)
	}

	return users, nil
}

func (d *UserFirestoreDAL) UpdateUserRoles(ctx context.Context, userID string, rolesRefs []*firestore.DocumentRef) error {
	userRef := d.GetRef(ctx, userID)

	_, err := userRef.Update(ctx, []firestore.Update{{Path: roles, Value: rolesRefs}})
	if err != nil {
		return err
	}

	return nil
}

func (d *UserFirestoreDAL) GetCustomerUsersByRoles(ctx context.Context, customerID string, roles []common.PresetRole) ([]*common.User, error) {
	var users []*common.User

	customerRef := d.customerCollection(ctx).Doc(customerID)
	roleRefs := make([]*firestore.DocumentRef, len(roles))

	for i, role := range roles {
		roleRefs[i] = d.firestoreClientFun(ctx).Collection("roles").Doc(string(role))
	}

	userDocSnaps, err := d.userCollection(ctx).
		Where("customer.ref", "==", customerRef).
		Where("roles", "array-contains-any", roleRefs).Documents(ctx).GetAll()
	if err != nil {
		return users, err
	}

	for _, docSnap := range userDocSnaps {
		var user common.User
		if err := docSnap.DataTo(&user); err != nil {
			return users, err
		}

		users = append(users, &user)
	}

	return users, nil
}

func (d *UserFirestoreDAL) GetUsersWithRecentEngagement(ctx context.Context) ([]common.User, error) {
	engagementCutOffDate := time.Now().AddDate(0, -1, 0)

	snaps, err := d.documentsHandler.GetAll(d.userCollection(ctx).Where("lastLogin", ">", engagementCutOffDate).Documents(ctx))
	if err != nil {
		return nil, err
	}

	var users []common.User

	for _, snap := range snaps {
		var user common.User
		if err := snap.DataTo(&user); err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

func (d *UserFirestoreDAL) GetLastUserEngagementTimeForCustomer(ctx context.Context, customerID string) (*time.Time, error) {
	customerRef := d.firestoreClientFun(ctx).Collection(customerCollection).Doc(customerID)

	snaps, err := d.documentsHandler.GetAll(d.userCollection(ctx).Where("customer.ref", "==", customerRef).Documents(ctx))
	if err != nil {
		return nil, err
	}

	var lastEngagement *time.Time

	for _, snap := range snaps {
		var user common.User
		if err := snap.DataTo(&user); err != nil {
			return nil, err
		}

		if lastEngagement == nil {
			lastEngagement = &user.LastLogin
		} else {
			if lastEngagement.Before(user.LastLogin) {
				lastEngagement = &user.LastLogin
			}
		}
	}

	return lastEngagement, nil
}
