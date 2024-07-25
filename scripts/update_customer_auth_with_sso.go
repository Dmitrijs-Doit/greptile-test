package scripts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

type UpdateCustomerAuth struct {
	fs          *firestore.Client
	fbAuth      *auth.Client
	wb          *fb.AutomaticWriteBatch
	projectID   string
	enableWrite bool
}

type authUpdate struct {
	tenantID   string
	customerID string
	auth       *common.CustomerAuthSso
}

type params struct {
	Project     string `json:"project"`
	EnableWrite bool   `json:"enableWrite"`
}

// UpdateCustomerSSOSettings Updates and validates Customer sso from auth field. There is an option to dry run without actually writing to firestore.
// Using this one can see what the changes will be beforehand
// Example payload
//
//	{
//	   "project": "doitintl-cmp-dev",
//	   "enable-write": false
//	}
func UpdateCustomerSSOSettings(ctx *gin.Context) []error {
	var p params

	if err := ctx.ShouldBindJSON(&p); err != nil {
		return []error{err}
	}

	if p.Project == "" {
		err := errors.New("invalid input parameters")
		return []error{err}
	}

	cAuth, err := NewUpdateCustomerAuth(ctx, fb.App, p.Project, p.EnableWrite)
	if err != nil {
		return []error{err}
	}

	cAuth.StartUpdateCustomerAuth(ctx, 400)

	return nil
}

func NewUpdateCustomerAuth(ctx context.Context, app *firebase.App, projectID string, enableWrite bool) (*UpdateCustomerAuth, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var fApp = app

	if fApp == nil {
		fApp, err = firebase.NewApp(ctx, &firebase.Config{
			ProjectID: projectID,
		})

		if err != nil {
			return nil, err
		}
	}

	fbAuth, err := fApp.Auth(ctx)
	if err != nil {
		return nil, err
	}

	batch := fb.NewAutomaticWriteBatch(fs, 500)

	return &UpdateCustomerAuth{
		fs:          fs,
		fbAuth:      fbAuth,
		wb:          batch,
		projectID:   projectID,
		enableWrite: enableWrite,
	}, nil
}

func (c *UpdateCustomerAuth) StartUpdateCustomerAuth(ctx context.Context, numberOfWorkers int) {
	var noSsoCount = 0

	var notFoundTenants uint64

	var notFoundCustomers uint64

	tChannel, err := c.generateTenantsChannel(ctx)
	if err != nil {
		panic(err)
	}

	var workers = make([]<-chan *authUpdate, numberOfWorkers)

	for i := 0; i < numberOfWorkers; i++ {
		//Fan out pattern: one channel consumed by multiple channels
		workers[i] = c.ssoAuthWorker(ctx, tChannel, &notFoundTenants, &notFoundCustomers)
	}

	for res := range merge(workers...) {
		if res.auth.SAML == nil && res.auth.OIDC == nil {
			noSsoCount++
		} else {
			if j, err := json.Marshal(res.auth); err == nil {
				log.Printf("Customer: %s ---  Tenant: %s --- SSO: %s\n", res.customerID, res.tenantID, string(j))
			}
		}
	}

	log.Printf("Customers with no SSO configured %d \n", noSsoCount)
	log.Printf("There are %d not found tenants in tenantToCustomer collection\n", notFoundTenants)
	log.Printf("There are %d not found customers\n", notFoundCustomers)
}

func (c *UpdateCustomerAuth) generateTenantsChannel(ctx context.Context) (<-chan *auth.TenantClient, error) {
	out := make(chan *auth.TenantClient)
	it := c.fbAuth.TenantManager.Tenants(ctx, "")

	go func() {
		for {
			t, err := it.Next()
			if err != nil {
				if err == iterator.Done {
					break
				}

				log.Printf("tenant iterator error %v", err)

				return
			}

			a, err := c.fbAuth.TenantManager.AuthForTenant(t.ID)
			if err != nil {
				log.Printf("could not read Auth for tenant %v", err)
				continue
			}
			out <- a
		}
		close(out)
	}()

	return out, nil
}

func (c *UpdateCustomerAuth) ssoAuthWorker(ctx context.Context, in <-chan *auth.TenantClient, notFoundTenants *uint64, notFoundCustomers *uint64) <-chan *authUpdate {
	out := make(chan *authUpdate)

	go func() {
		for tAuth := range in {
			var newSSOAuth common.CustomerAuthSso

			customerRef, err := c.getCustomerRefByTenantID(ctx, tAuth.TenantID())
			if err != nil {
				handleNotFoundError(err, notFoundTenants)
				continue
			}

			customer, err := common.GetCustomer(ctx, customerRef)

			if err != nil {
				handleNotFoundError(err, notFoundCustomers)
				continue
			}

			newSSOAuth.SAML = getSAMLConfig(ctx, tAuth)
			newSSOAuth.OIDC = getOIDCConfig(ctx, tAuth)

			setLastSelectedConfig(customer.Auth.Sso, &newSSOAuth)
			setDisabledIfConfigured(&newSSOAuth)

			if !(validateConfigs(customerRef.ID, customer.Auth.Sso, &newSSOAuth)) {
				continue
			}

			out <- &authUpdate{
				customerID: customerRef.ID,
				tenantID:   tAuth.TenantID(),
				auth:       &newSSOAuth,
			}
			writeToCustomerFs(ctx, customerRef, c.enableWrite, &newSSOAuth)
		}

		close(out)
	}()

	return out
}

func setDisabledIfConfigured(new *common.CustomerAuthSso) {
	//if OIDC is configured and SAML nil -> OIDC becomes disabled
	if new.OIDC != nil && *new.OIDC == "configured" && new.SAML == nil {
		*new.OIDC = "disabled"
	}

	//if SAML is configured and OIDC nil -> SAML becomes disabled
	if new.SAML != nil && *new.SAML == "configured" && new.OIDC == nil {
		*new.SAML = "disabled"
	}

	//if both are configured one RANDOMLY becomes disabled
	if new.SAML != nil && *new.SAML == "configured" && new.OIDC != nil && *new.OIDC == "configured" {
		*new.SAML = "disabled"
	}
}

func setLastSelectedConfig(existent *common.CustomerAuthSso, new *common.CustomerAuthSso) {
	//Only set new OIDC config to disabled if it previously was configured/disabled/nil
	if existent != nil && existent.OIDC != nil && *existent.OIDC == "disabled" && (new.OIDC == nil || *new.OIDC != "enabled") {
		new.OIDC = existent.OIDC
	}

	//Only set new SAML config to disabled if it previously was configured/disabled/nil
	if existent != nil && existent.SAML != nil && *existent.SAML == "disabled" && (new.SAML == nil || *new.SAML != "enabled") {
		new.SAML = existent.SAML
	}
}

func validateConfigs(customerID string, existent *common.CustomerAuthSso, new *common.CustomerAuthSso) bool {
	if existent == nil {
		return true
	}

	//If both are existent and have the same value
	if new.SAML != nil && new.OIDC != nil && *new.SAML == *new.OIDC {
		log.Printf("Customer: %s Invalid configuration, both configurationsa are %s", customerID, *new.SAML)
		return false
	}

	//OIDC Identity platform is different from firestore
	if existent.OIDC != nil && new.OIDC == nil {
		log.Printf("Customer: %s Invalid OIDC Firestore: %s, Identity Platform: NOT CONFIGURED", customerID, *existent.OIDC)
		return false
	}

	//SAML Identity platform is different from firestore
	if existent.SAML != nil && new.SAML == nil {
		log.Printf("Customer: %s Invalid SAML Firestore: %s, Identity Platform: NOT CONFIGURED", customerID, *existent.SAML)
		return false
	}

	// OIDC firestore and Identity platform "Enabled" don't match
	if existent.OIDC != nil && new.OIDC != nil && *new.OIDC != *existent.OIDC && (*existent.OIDC == "enabled" || *new.OIDC == "enabled") {
		log.Printf("Customer: %s Invalid OIDC Firestore: %s, Identity Platform %s", customerID, *existent.OIDC, *new.OIDC)
		return false
	}

	// SAML firestore and Identity platform "Enabled" don't match
	if existent.SAML != nil && new.SAML != nil && *new.SAML != *existent.SAML && (*existent.SAML == "enabled" || *new.SAML == "enabled") {
		log.Printf("Customer: %s Invalid SAML Firestore: %s, Identity Platform %s", customerID, *existent.SAML, *new.SAML)
		return false
	}

	//One can not be enabled and the other disabled
	if new.SAML != nil && new.OIDC != nil && *new.SAML == "enabled" && *new.OIDC == "disabled" {
		log.Printf("Customer: %s Invalid configuration, SAML enabled and OIDC disabled", customerID)
		return false
	}

	//One can not be enabled and the other disabled
	if new.SAML != nil && new.OIDC != nil && *new.OIDC == "enabled" && *new.SAML == "disabled" {
		log.Printf("Customer: %s Invalid configuration, OIDC enabled and SAML disabled", customerID)
		return false
	}

	return true
}

func writeToCustomerFs(ctx context.Context, ref *firestore.DocumentRef, enableWrite bool, sso *common.CustomerAuthSso) {
	if !enableWrite {
		return
	}

	docSnap, err := ref.Get(ctx)

	if err != nil {
		log.Printf("error getting customer docSnap %v", err)
	}

	a := common.Auth{
		Sso: sso,
	}

	_, err = docSnap.Ref.Update(ctx, []firestore.Update{
		{FieldPath: []string{"auth"}, Value: a},
	})

	if err != nil {
		log.Printf("could not update customer %v", err)
	}
}

func handleNotFoundError(err error, notFound *uint64) {
	if status.Code(err) == codes.NotFound {
		atomic.AddUint64(notFound, 1)
		return
	}

	log.Printf("error getting tenantToCustomer Ref %s\n", err)
}

// Fan in pattern: multiple channels merged into one channel
func merge(cs ...<-chan *authUpdate) <-chan *authUpdate {
	var wg sync.WaitGroup

	out := make(chan *authUpdate)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan *authUpdate) {
		for n := range c {
			out <- n
		}

		wg.Done()
	}

	wg.Add(len(cs))

	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func getSAMLConfig(ctx context.Context, auth *auth.TenantClient) *string {
	samlIter := auth.SAMLProviderConfigs(ctx, "")

	var config string

	saml, err := samlIter.Next()
	if err == iterator.Done {
		return nil
	}

	for {
		if saml.Enabled {
			config = "enabled"
			break
		} else {
			config = "configured"
		}

		saml, err = samlIter.Next()
		if err == iterator.Done {
			return &config
		}
	}

	return &config
}

func getOIDCConfig(ctx context.Context, auth *auth.TenantClient) *string {
	oidcIter := auth.OIDCProviderConfigs(ctx, "")

	var config string

	oidc, err := oidcIter.Next()
	if err == iterator.Done {
		return nil
	}

	for {
		if oidc.Enabled {
			config = "enabled"
			break
		} else {
			config = "configured"
		}

		oidc, err = oidcIter.Next()
		if err == iterator.Done {
			return &config
		}
	}

	return &config
}

func (c *UpdateCustomerAuth) getCustomerRefByTenantID(ctx context.Context, tenantID string) (*firestore.DocumentRef, error) {
	var d tenantToCustomerDoc

	tenToCstmrDocRef := c.fs.Collection(fmt.Sprintf("tenants/%s/tenantToCustomer", c.projectID)).Doc(tenantID)
	doc, err := tenToCstmrDocRef.Get(ctx)

	if err != nil {
		return nil, err
	}

	if err = doc.DataTo(&d); err != nil {
		return nil, err
	}

	return c.fs.Collection("customers").Doc(d.CustomerID), nil
}
