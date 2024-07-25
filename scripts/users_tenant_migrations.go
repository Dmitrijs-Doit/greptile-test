package scripts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/auth/hash"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

const (
	batchSize = 1000
)

type MigrationUsersCustomerTenantRequest struct {
	CustomerID string `json:"customerId"`
	TenantID   string `json:"tenantId"`
}

func MigrateUsersToCustomerTenant(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	var req MigrationUsersCustomerTenantRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{err}
	}

	utms, err := NewUsersTenantsMigrationScript(ctx, l, req.CustomerID, req.TenantID)
	if err != nil {
		return []error{err}
	}

	if utms.customerID != "" && utms.tenantID != "" {
		// delete users from tenant
		if err := utms.deleteUsersFromTenant(ctx); err != nil {
			return []error{err}
		}
		// run migration script for a specific customer
		return utms.migrateUserToCustomerTenantByID(ctx)
	}

	return utms.MigrateUsersToCustomerTenant(ctx)
}

type UsersTenantsMigrationScript struct {
	fs         *firestore.Client
	fbAuth     *auth.Client
	l          logger.ILogger
	hashConfig hash.Scrypt
	customerID string
	tenantID   string
}
type hashConfig struct {
	Key     string `json:"base64_signer_key"`
	Salt    string `json:"base64_salt_separator"`
	Rounds  int    `json:"rounds"`
	MemCost int    `json:"mem_cost"`
}

func NewUsersTenantsMigrationScript(ctx context.Context, l logger.ILogger, customerID, tenantID string) (*UsersTenantsMigrationScript, error) {
	projectID := common.ProjectID

	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	fbApp, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: projectID,
	})
	if err != nil {
		return nil, err
	}

	fbAuth, err := fbApp.Auth(ctx)
	if err != nil {
		return nil, err
	}

	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretRootTenantHashKey)
	if err != nil {
		return nil, err
	}

	var hashConfig hashConfig

	err = json.Unmarshal(secret, &hashConfig)
	if err != nil {
		return nil, err
	}

	return &UsersTenantsMigrationScript{
		fs:     fs,
		fbAuth: fbAuth,
		l:      l,
		hashConfig: hash.Scrypt{
			Key:           b64Stddecode(hashConfig.Key),
			SaltSeparator: b64Stddecode(hashConfig.Salt),
			Rounds:        hashConfig.Rounds,
			MemoryCost:    hashConfig.MemCost,
		},
		customerID: customerID,
		tenantID:   tenantID,
	}, nil
}

func (utms *UsersTenantsMigrationScript) MigrateUsersToCustomerTenant(ctx context.Context) []error {
	utms.l.Debug("users tenants migration script starts")

	authFrom := utms.fbAuth

	usersByTenants := make(map[string][]*auth.UserToImport)

	currentBatchSize := 0
	iter := authFrom.Users(ctx, "")

	for {
		currentBatchSize++

		user, err := iter.Next()
		if err == iterator.Done {
			if len(usersByTenants) > 0 {
				utms.DumpBatch(ctx, usersByTenants)
			}

			break
		}

		if err != nil {
			return []error{fmt.Errorf("error listing users: %v", err)}
		}

		tenantID, err := utms.getTenant(ctx, user)
		if err != nil {
			utms.l.Warningf("%v\n", err)
			continue
		}

		userToImport := populateUserToImport(user, tenantID)

		users := usersByTenants[tenantID]
		if users == nil {
			users = []*auth.UserToImport{}
		}

		users = append(users, userToImport)
		usersByTenants[tenantID] = users

		if currentBatchSize == batchSize {
			utms.DumpBatch(ctx, usersByTenants)

			currentBatchSize = 0

			usersByTenants = make(map[string][]*auth.UserToImport)
		}
	}
	utms.l.Debug("users tenants migration script finished")

	return nil
}

func (utms *UsersTenantsMigrationScript) DumpBatch(ctx context.Context, usersByTenant map[string][]*auth.UserToImport) {
	for tenantID, users := range usersByTenant {
		utms.l.Debugf("tenantID: %s => Users counter: %v", tenantID, len(users))

		authTo, err := utms.fbAuth.TenantManager.AuthForTenant(tenantID)
		if err != nil {
			utms.l.Warningf("Error initilizing auth for tenant. tenantId: %s, err: %v", tenantID, err)
		}

		result, err := authTo.ImportUsers(ctx, users, auth.WithHash(utms.hashConfig))
		if err != nil {
			utms.l.Warningf("Error importing users. tenantId: %s, err: %v", tenantID, err)
		}

		for _, e := range result.Errors {
			utms.l.Warningf("Failed to import user. err: %v", e.Reason)
		}
	}
}

func (utms *UsersTenantsMigrationScript) getTenant(ctx context.Context, user *auth.ExportedUserRecord) (string, error) {
	if customerID, ok := user.CustomClaims["customerId"]; ok {
		tenantID, err := utms.getTenantOfCustomerByCustomerID(ctx, customerID.(string))
		if err == nil {
			return tenantID, nil
		}

		utms.l.Debugf("%v", err)
	}

	parts := strings.Split(user.Email, "@")
	domain := parts[1]

	tenantID, err := utms.getTenantOfCustomerByDomain(ctx, domain)
	if err == nil {
		return tenantID, nil
	}

	utms.l.Debugf("%v", err)

	tenantID, err = utms.getTenantOfPartnerByDomain(ctx, domain)
	if err == nil {
		return tenantID, nil
	}

	utms.l.Debugf("%v", err)

	return "", fmt.Errorf("failed to find tenantId for user. userID: %v", user.UID)
}

func (utms *UsersTenantsMigrationScript) getTenantOfCustomerByCustomerID(ctx context.Context, customerID string) (string, error) {
	docSnap, err := utms.fs.Collection("customers").Doc(customerID).Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed fetching customer doc. ID: %s, error: %v", customerID, err)
	}

	tenanatID, err := docSnap.DataAt(tenantIDPath)
	if err == nil {
		return tenanatID.(string), nil
	}

	return "", fmt.Errorf("customer doesn't have any tenant configured. customerId: %s", customerID)
}

func (utms *UsersTenantsMigrationScript) getTenantOfCustomerByDomain(ctx context.Context, domain string) (string, error) {
	docSnaps, err := utms.fs.Collection("customers").Where("domains", "array-contains", domain).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return "", fmt.Errorf("failed fetching tenantId from customer doc. domain: %v, error: %v", domain, err)
	}

	if len(docSnaps) <= 0 {
		return "", fmt.Errorf("domain '%v' doesn't belong to any customer", domain)
	}

	tenanatID, err := docSnaps[0].DataAt(tenantIDPath)
	if err == nil {
		return tenanatID.(string), nil
	}

	return "", fmt.Errorf("customer doesn't have any tenant configured. customerId: %v", docSnaps[0].Ref.ID)
}

func (utms *UsersTenantsMigrationScript) getTenantOfPartnerByDomain(ctx context.Context, domain string) (string, error) {
	docSnap, err := utms.fs.Collection("app").Doc("partner-access").Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed fetching 'partner-access' doc")
	}

	var partners common.Partners
	if err := docSnap.DataTo(&partners); err != nil {
		return "", fmt.Errorf("failed populating 'partners' doc. error: %v", err)
	}

	// for _, partner := range partners.Partners {
	// 	domains := partner.Domains
	// 	if contains(domains, domain) {
	// 		if len(partner.Auth.TenantID) > 0 {
	// 			return partner.Auth.TenantID, nil
	// 		}

	// 		return "", fmt.Errorf("partner has no tenantId configured. name: %v, domain: %v", "", domain)
	// 	}
	// }

	return "", fmt.Errorf("found no partner for domain. domain: %v", domain)
}

func (utms *UsersTenantsMigrationScript) migrateUserToCustomerTenantByID(ctx context.Context) []error {
	utms.l.Debugf("users tenants migration script starts for customer: %s", utms.customerID)

	authFrom := utms.fbAuth

	usersByTenants := make(map[string][]*auth.UserToImport)

	currentBatchSize := 0
	iter := authFrom.Users(ctx, "")

	for {
		currentBatchSize++

		user, err := iter.Next()
		if err == iterator.Done {
			if len(usersByTenants) > 0 {
				utms.DumpBatch(ctx, usersByTenants)
			}

			break
		}

		if err != nil {
			return []error{fmt.Errorf("error listing users: %v", err)}
		}

		customerID, ok := user.CustomClaims["customerId"]
		if !ok {
			continue
		}

		if customerID == utms.customerID {
			userToImport := populateUserToImport(user, utms.tenantID)

			users := usersByTenants[utms.tenantID]
			if users == nil {
				users = []*auth.UserToImport{}
			}

			users = append(users, userToImport)

			usersByTenants[utms.tenantID] = users
			if currentBatchSize == batchSize {
				utms.DumpBatch(ctx, usersByTenants)

				currentBatchSize = 0

				usersByTenants = make(map[string][]*auth.UserToImport)
			}
		}
	}
	utms.l.Debugf("users tenants migration for customer %s script finished", utms.customerID)

	return nil
}

func (utms *UsersTenantsMigrationScript) deleteUsersFromTenant(ctx context.Context) error {
	tenantAuth, err := utms.fbAuth.TenantManager.AuthForTenant(utms.tenantID)
	if err != nil {
		return err
	}

	iter := tenantAuth.Users(ctx, "")

	for {
		user, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err = tenantAuth.DeleteUser(ctx, user.UID); err != nil {
			return err
		}
	}

	return nil
}

func populateUserToImport(user *auth.ExportedUserRecord, tenantID string) *auth.UserToImport {
	customClaims := user.CustomClaims
	if customClaims != nil {
		customClaims[common.ClaimsTenantID] = tenantID
	}

	userToImport := (&auth.UserToImport{}).
		UID(user.UID).
		Email(user.Email).
		CustomClaims(customClaims).
		Disabled(user.Disabled).
		DisplayName(user.DisplayName).
		EmailVerified(user.EmailVerified).
		Metadata(user.UserMetadata).
		PhotoURL(user.PhotoURL)

	if len(user.PasswordHash) > 0 {
		userToImport.
			PasswordHash(b64URLdecode(user.PasswordHash)).
			PasswordSalt(b64URLdecode(user.PasswordSalt))
	}

	if len(user.PhoneNumber) > 0 {
		userToImport.PhoneNumber(user.PhoneNumber)
	}

	if user.ProviderUserInfo != nil {
		providers := make([]*auth.UserProvider, len(user.ProviderUserInfo))
		for i, provider := range user.ProviderUserInfo {
			providers[i] = &auth.UserProvider{
				UID:         provider.UID,
				Email:       provider.Email,
				DisplayName: provider.DisplayName,
				PhotoURL:    provider.PhotoURL,
				ProviderID:  provider.ProviderID,
			}
		}

		userToImport.ProviderData(providers)
	}

	return userToImport
}

func b64Stddecode(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		fmt.Printf("Failed to decode string. err: %v", err)
	}

	return b
}

func b64URLdecode(s string) []byte {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		fmt.Printf("Failed to decode string. err: %v", err)
	}

	return b
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}

	return false
}
