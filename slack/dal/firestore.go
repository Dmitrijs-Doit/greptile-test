package dal

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"

	sharedFirestore "github.com/doitintl/firestore"
	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customer "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	notificationDomain "github.com/doitintl/notificationcenter/domain"
	"github.com/gin-gonic/gin"
)

const (
	oldDomain = "doit-intl.com"
	newDomain = "doit.com"
)

/*
	FirestoreDAL

Data Access Layer responsible for all interactions with firestore collections (slack, slackApp, customers etc)
*/
type FirestoreDAL struct {
	slack     sharedFirestore.Slack
	users     sharedFirestore.Users
	customers customer.Customers
}

func NewFirestoreDAL(ctx context.Context, conn *connection.Connection) *FirestoreDAL {
	slack := sharedFirestore.NewSlackDALWithClient(conn.Firestore(ctx))
	customers := customer.NewCustomersFirestoreWithClient(conn.Firestore) //sharedFirestore.NewCustomersDALWithClient(conn.Firestore(ctx))
	users := sharedFirestore.NewUsersDALWithClient(conn.Firestore(ctx))

	return &FirestoreDAL{
		slack:     slack,
		customers: customers,
		users:     users,
	}
}

func (d *FirestoreDAL) GetCustomer(ctx context.Context, customerID string) (*firestore.DocumentRef, *common.Customer, error) {
	customer, err := d.customers.GetCustomer(ctx, customerID)
	return d.customers.GetRef(ctx, customerID), customer, err
}

// GetWorkspaceDecrypted - given workspaceID returns parsed workspace, ID, decrypted bot token, decrypted user token
func (d *FirestoreDAL) GetWorkspaceDecrypted(ctx context.Context, workspaceID string) (*firestorePkg.SlackWorkspace, string, string, string, error) {
	workspace, err := d.slack.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, "", "", "", err
	}

	userToken, botToken, err := decryptWorkspace(workspace)
	if err != nil {
		return nil, "", "", "", err
	}

	return workspace, workspaceID, userToken, botToken, nil
}

// GetCustomerWorkspaceDecrypted - given customerID returns parsed workspace, ID, decrypted bot token, decrypted user token
func (d *FirestoreDAL) GetCustomerWorkspaceDecrypted(ctx context.Context, customerID string) (*firestorePkg.SlackWorkspace, string, string, string, error) {
	workspace, workspaceID, err := d.GetCustomerWorkspace(ctx, customerID)
	if err != nil {
		return nil, "", "", "", err
	}

	userToken, botToken, err := decryptWorkspace(workspace)
	if err != nil {
		return nil, "", "", "", err
	}

	return workspace, workspaceID, userToken, botToken, nil
}

// GetWorkspace - given customerID returns parsed workspace, ID
func (d *FirestoreDAL) GetCustomerWorkspace(ctx context.Context, customerID string) (*firestorePkg.SlackWorkspace, string, error) {
	return d.slack.GetCustomerWorkspace(ctx, d.customers.GetRef(ctx, customerID))
}

func (d *FirestoreDAL) SetCustomerWorkspace(ctx context.Context, workspaceID string, workspace *firestorePkg.SlackWorkspace) error {
	return d.slack.SetWorkspace(ctx, workspaceID, workspace)
}

func (d *FirestoreDAL) GetDoitEmployee(ctx context.Context, email string) (*firestorePkg.User, error) {
	doitEmployee, err := d.GetUser(ctx, email)
	if err != nil && err != sharedFirestore.ErrNotFound {
		return nil, err
	}

	if doitEmployee == nil { //	try to fetch doer email with the 2nd domain
		emailSplit := strings.Split(email, "@")
		user := emailSplit[0]
		domain := emailSplit[1]

		var emailWithOtherDomain string

		switch domain {
		case oldDomain:
			emailWithOtherDomain = fmt.Sprintf("%s@%s", user, newDomain)
		case newDomain:
			emailWithOtherDomain = fmt.Sprintf("%s@%s", user, oldDomain)
		default:
			return nil, fmt.Errorf("domain [%s] does not belong to doit", domain)
		}

		doitEmployee, err = d.GetUser(ctx, emailWithOtherDomain)
	}

	return doitEmployee, err
}

func (d *FirestoreDAL) GetUser(ctx context.Context, email string) (*firestorePkg.User, error) {
	return d.users.GetUser(ctx, email)
}

func (d *FirestoreDAL) UserHasCloudAnalyticsPermission(ctx context.Context, email string) (bool, error) {
	userRef, err := d.users.GetUserRef(ctx, email)
	if err != nil {
		return false, err
	}

	commonUser, err := common.GetUser(ctx.(*gin.Context), userRef)
	if err != nil {
		return false, err
	}

	return commonUser.HasCloudAnalyticsPermission(ctx.(*gin.Context)), nil
}

func (d *FirestoreDAL) GetSharedChannel(ctx context.Context, channelID string) (*firestorePkg.SharedChannel, error) {
	return d.slack.GetSharedChannel(ctx, channelID)
}

func (d *FirestoreDAL) GetCustomerSharedChannel(ctx context.Context, customerID string) (*firestorePkg.SharedChannel, error) {
	return d.slack.GetCustomerSharedChannel(ctx, customerID)
}

func (d *FirestoreDAL) SetCustomerSharedChannel(ctx context.Context, customerID string, channel *firestorePkg.SharedChannel) error {
	return d.slack.SetCustomerSharedChannel(ctx, customerID, channel)
}

func (d *FirestoreDAL) DeleteCustomerSharedChannel(ctx context.Context, customerID, channelID string) error {
	return d.slack.DeleteCustomerSharedChannel(ctx, customerID, channelID)
}

func (d *FirestoreDAL) CreateNotificationConfig(ctx context.Context, config notificationDomain.NotificationConfig) error {
	_, _, err := config.CustomerRef.Collection("notifications").Add(ctx, config)
	return err
}
