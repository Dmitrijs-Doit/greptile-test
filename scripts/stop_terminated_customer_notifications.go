package scripts

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

func StopTerminatedCustomerNotifications(ctx *gin.Context) []error {
	var errors []error

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	customers, err := fs.Collection("customers").
		Where("classification", "in", []string{"terminated", "suspendedForNonPayment"}).
		Documents(ctx).
		GetAll()
	if err != nil {
		errors = append(errors, err)
		return errors
	}

	common.RunConcurrentJobsOnCollection(ctx, customers, 5, func(ctx context.Context, customer *firestore.DocumentSnapshot) {
		customerRef := customer.Ref
		users, err := fs.Collection("users").
			Where("customer.ref", "==", customerRef).
			Where("userNotifications", "array-contains-any", []int{1, 2, 3, 4, 5, 6, 7, 8}).
			Documents(ctx).
			GetAll()

		if err != nil {
			errors = append(errors, err)
			return
		}

		for _, user := range users {
			err = clearUserNotifications(ctx, user)
			if err != nil {
				errors = append(errors, err)
			}
		}
	})

	return errors
}

func clearUserNotifications(ctx context.Context, user *firestore.DocumentSnapshot) error {
	notifications, err := user.DataAt("userNotifications")
	if err != nil {
		return err
	}

	_, err = user.DataAt("userNotificationsBackup")
	if err != nil && err.Error() == `firestore: no field "userNotificationsBackup"` {
		_, err = user.Ref.Update(ctx, []firestore.Update{{Path: "userNotifications", Value: []int64{}}, {Path: "userNotificationsBackup", Value: notifications}})
		if err != nil {
			fmt.Printf("Error updating user notifications, user: %s\n", user.Ref.ID)
			return err
		}

		fmt.Printf("Successful update user notifications, user: %s, old notifications: %v\n", user.Ref.ID, notifications)

		return nil
	}

	if err != nil {
		return err
	}

	fmt.Printf("userNotificationsBackup already exists, user: %s\n", user.Ref.ID)

	return nil
}
