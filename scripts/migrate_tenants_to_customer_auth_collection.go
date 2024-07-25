package scripts

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type tenantIDSynchronizer struct {
	fs        *firestore.Client
	fbAuth    *auth.Client
	l         logger.ILogger
	tenantMap map[string]string
	emailMap  sync.Map
	wb        *fb.AutomaticWriteBatch
	wg        *sync.WaitGroup
	projectID string
}

type customerToTenantDoc struct {
	TenantID string `firestore:"tenantId"`
}

type tenantToCustomerDoc struct {
	CustomerID string `firestore:"customerId"`
}

type emailToTenantDoc struct {
	TenantID string `firestore:"tenantId"`
}

func newTenantIDSynchronizer(ctx context.Context, l logger.ILogger, projectID string) (*tenantIDSynchronizer, error) {
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

	batch := fb.NewAutomaticWriteBatch(fs, 200)

	ts := tenantIDSynchronizer{
		fs:        fs,
		fbAuth:    fbAuth,
		l:         l,
		wb:        batch,
		wg:        &sync.WaitGroup{},
		projectID: projectID,
	}

	return &ts, nil
}

func (ts *tenantIDSynchronizer) getTenants(ctx context.Context) error {
	maxGoRoutines := 500
	guard := make(chan struct{}, maxGoRoutines)
	it := ts.fbAuth.TenantManager.Tenants(ctx, "")
	counter := 0
	tenantsMap := make(map[string]string)
	ts.emailMap = sync.Map{}

	for {
		counter++

		tenant, err := it.Next()
		if err == iterator.Done {
			ts.l.Infof("Finsihed iterating %d tenants", counter)
			break
		} else if err != nil {
			return err
		}

		tenantsMap[tenant.DisplayName] = tenant.ID

		guard <- struct{}{}
		go ts.iterateTenantUsers(ctx, tenant.ID, guard)
	}

	ts.tenantMap = tenantsMap
	ts.wg.Wait()

	return nil
}

func (ts *tenantIDSynchronizer) syncCustomerTenantIDs(ctx context.Context) error {
	customerSnaps, err := ts.fs.Collection("customers").Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	maxGoRoutines := 50
	guard := make(chan struct{}, maxGoRoutines)

	for _, customerSnap := range customerSnaps {
		var customer common.Customer
		if err := customerSnap.DataTo(&customer); err != nil {
			ts.l.Errorf("error getting customer data for customer %s: %s", customerSnap.Ref.ID, err.Error())
			continue
		}

		suffix := customerSnap.Ref.ID
		if len(customerSnap.Ref.ID) > 18 {
			suffix = customerSnap.Ref.ID[:18]
		}

		customerTenantDisplayName := fmt.Sprintf("t-%s", suffix)

		tenantID, ok := ts.tenantMap[customerTenantDisplayName]
		if !ok {
			guard <- struct{}{}

			ts.l.Warningf("couldn't find tenant for customer: %s ", customerSnap.Ref.ID)

			go ts.createTenant(ctx, customerSnap.Ref.ID, customerTenantDisplayName, guard)
		} else {
			ts.setTenantDocBatch(customerSnap.Ref.ID, tenantID)
		}
	}

	return nil
}

func (ts *tenantIDSynchronizer) iterateTenantUsers(ctx context.Context, tenantID string, guard chan struct{}) {
	ts.wg.Add(1)

	defer func() {
		ts.wg.Done()
		<-guard
	}()

	tenantClient, _ := ts.fbAuth.TenantManager.AuthForTenant(tenantID)
	userIt := tenantClient.Users(ctx, "")
	emails := make([]string, 0)
	counter := 0

	for {
		userRecord, err := userIt.Next()
		if err == iterator.Done {
			ts.l.Infof("Finsihed iterating users of tenant: %s, num of emails: %d", tenantID, counter)
			break
		} else if err != nil {
			ts.l.Errorf("%v", err)
			continue
		}

		counter++

		emails = append(emails, userRecord.Email)
	}
	ts.emailMap.Store(tenantID, emails)
}

func (ts *tenantIDSynchronizer) createTenant(ctx context.Context, customerID, displayName string, guard chan struct{}) {
	ts.wg.Add(1)

	defer func() {
		ts.wg.Done()
		<-guard
	}()
	ts.l.Infof("creating tenant for customer: %s ", customerID)

	tenantToCreate := (&auth.TenantToCreate{}).AllowPasswordSignUp(true).DisplayName(displayName)

	tenant, err := ts.fbAuth.TenantManager.CreateTenant(ctx, tenantToCreate)
	if err != nil {
		ts.l.Errorf("couldn't create tenant for customer: %s ", customerID)
		ts.l.Error(err.Error())

		return
	}

	ts.l.Infof("created tenant for customer: %s; tenantId: %s ", customerID, tenant.ID)
	ts.setTenantDocBatch(customerID, tenant.ID)
}

func (ts *tenantIDSynchronizer) setTenantDocBatch(customerID, tenantID string) {
	tcd := customerToTenantDoc{
		TenantID: tenantID,
	}
	customerToTenantCollectionID := fmt.Sprintf("tenants/%s/customerToTenant", ts.projectID)
	customerDocRef := ts.fs.Collection(customerToTenantCollectionID).Doc(customerID)
	ts.wb.Set(customerDocRef, tcd)

	tIDd := tenantToCustomerDoc{
		CustomerID: customerID,
	}
	tenantToCustomerCollectionId := fmt.Sprintf("tenants/%s/tenantToCustomer", ts.projectID)
	tenantIDDocRef := ts.fs.Collection(tenantToCustomerCollectionId).Doc(tenantID)
	ts.wb.Set(tenantIDDocRef, tIDd)
}

func (ts *tenantIDSynchronizer) syncPartnertenantIDs(ctx context.Context) error {
	partnersRef := ts.fs.Collection("app").Doc("partner-access")

	partnerSnap, err := partnersRef.Get(ctx)
	if err != nil {
		return err
	}

	var partners common.Partners
	if err := partnerSnap.DataTo(&partners); err != nil {
		ts.l.Errorf("error getting partners doc")
	}

	for _, partner := range partners.Partners {
		partnerDisplayName := fmt.Sprintf("t-%s", strings.Replace(partner.Name, ".", "-", -1))

		tenantID, ok := ts.tenantMap[partnerDisplayName]
		if !ok {
			ts.l.Warningf("couldn't find tenant for partner %s: ", partner.Name)
			continue
		}

		ts.setTenantDocBatch(partner.Name, tenantID)
	}

	return nil
}

func (ts *tenantIDSynchronizer) syncEmailToTenantId(ctx context.Context) {
	ts.emailMap.Range(func(tenantID, tenantEmails interface{}) bool {
		tenantEmailsSlice := tenantEmails.([]string)
		projectID := common.ProjectID

		emailCollectionID := ts.fs.Collection(fmt.Sprintf("tenants/%s/emailToTenant", projectID))
		for _, email := range tenantEmailsSlice {
			emailToTenantDocRef := emailCollectionID.Doc(email)
			emailToTenantDocdata := emailToTenantDoc{
				TenantID: tenantID.(string),
			}
			ts.wb.Set(emailToTenantDocRef, emailToTenantDocdata)
		}

		return true
	})
}

func syncCustomerTenantIDs(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	projectID := common.ProjectID
	l.Infof("Starting syncing customer tenantIDs for project: %s", projectID)

	ts, err := newTenantIDSynchronizer(ctx, l, projectID)
	if err != nil {
		return []error{err}
	}

	ts.getTenants(ctx)

	if err := ts.syncCustomerTenantIDs(ctx); err != nil {
		return []error{err}
	}

	if err := ts.syncPartnertenantIDs(ctx); err != nil {
		return []error{err}
	}

	ts.syncEmailToTenantId(ctx)
	ts.wg.Wait()

	errors := ts.wb.Commit(ctx)
	if len(errors) > 0 {
		return errors
	}

	return nil
}
