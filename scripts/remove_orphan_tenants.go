// Script for removing orphaned tenants from GCP Identity Platform
//
// An orphaned tenant is a tenant which does not have an actual customer ID linked to it
// sourced from tenantsToCustomer collection

package scripts

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type TenantRemover struct {
	write     bool
	projectID string
	fs        *firestore.Client
	auth      *auth.Client
	l         logger.ILogger
}

func NewTenantRemover(ctx context.Context, l logger.ILogger, projectID string, write bool) (*TenantRemover, error) {
	fb, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("error creating firebase app: %w", err)
	}

	auth, err := fb.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting firebase auth: %w", err)
	}

	fs, err := fb.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting firestore client: %w", err)
	}

	return &TenantRemover{
		write:     write,
		projectID: projectID,
		fs:        fs,
		auth:      auth,
		l:         l,
	}, nil
}

type RemoveOrphanTenantsRequest struct {
	Write bool `json:"write"`
}

func RemoveOrphanTenants(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	var req RemoveOrphanTenantsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{fmt.Errorf("error binding json request: %w", err)}
	}

	tr, err := NewTenantRemover(ctx, l, common.ProjectID, req.Write)
	if err != nil {
		return []error{fmt.Errorf("error creating tenant remover: %w", err)}
	}

	l.Info("remove orphan tenants script starting...")

	tenants, err := tr.getTenantsFromIDP(ctx)
	if err != nil {
		return []error{err}
	}

	tenantsToCustomers, err := tr.getTenantToCustomerMapping(ctx)
	if err != nil {
		return []error{err}
	}

	numJobs := len(tenants)
	numWorkers := 250
	jobs := make(chan *auth.Tenant, numJobs)
	results := make(chan string)

	var wg sync.WaitGroup

	wg.Add(numWorkers)

	for _, t := range tenants {
		jobs <- t
	}

	close(jobs)

	for w := 0; w < numWorkers; w++ {
		go func() {
			defer wg.Done()

			for t := range jobs {
				tr.checkCustomerExists(ctx, t, tenantsToCustomers, results)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var remove []string
	for r := range results {
		remove = append(remove, r)
	}

	if !req.Write {
		ctx.JSON(http.StatusOK, gin.H{
			"message":   fmt.Sprintf("action would remove %d tenants", len(remove)),
			"tenantIds": remove,
		})

		l.Info("remove orphan tenants script complete")

		return nil
	}

	for _, id := range remove {
		if err := tr.auth.TenantManager.DeleteTenant(ctx, id); err != nil {
			return []error{fmt.Errorf("error deleting tenant with id %s: %w", id, err)}
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("removed %d tenants", len(remove)),
		"tenantIds": remove,
	})

	l.Info("remove orphan tenants script complete")

	return nil
}

func (tr *TenantRemover) getTenantsFromIDP(ctx context.Context) ([]*auth.Tenant, error) {
	var tenants []*auth.Tenant

	it := tr.auth.TenantManager.Tenants(ctx, "")

	for {
		tenant, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("error getting tenant from iterator: %w", err)
		}

		tenants = append(tenants, tenant)
	}

	return tenants, nil
}

func (tr *TenantRemover) getTenantToCustomerMapping(ctx context.Context) (map[string]string, error) {
	tenantToCustomer, err := tr.fs.Collection("tenants").Doc(tr.projectID).Collection("tenantToCustomer").Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("error getting tenantToCustomer collection: %w", err)
	}

	tm := make(map[string]string, len(tenantToCustomer))

	for _, t := range tenantToCustomer {
		id, err := t.DataAt("customerId")
		if err != nil {
			return nil, fmt.Errorf("error getting data at customerId: %w", err)
		}

		cid, ok := id.(string)
		if !ok {
			return nil, fmt.Errorf("error casting id to string")
		}

		tm[t.Ref.ID] = cid
	}

	return tm, nil
}

func (tr *TenantRemover) checkCustomerExists(
	ctx context.Context,
	tenant *auth.Tenant,
	tenantToCustomer map[string]string,
	result chan<- string,
) {
	if tenant == nil {
		return
	}

	cid, found := tenantToCustomer[tenant.ID]
	if !found {
		tr.l.Infof("tenant %s not found in firestore", tenant.ID)
		result <- tenant.ID

		return
	}

	_, err := tr.fs.Collection("customers").Doc(cid).Get(ctx)
	if status.Code(err) == codes.NotFound {
		result <- tenant.ID
		return
	}

	if err != nil {
		tr.l.Errorf("error getting customer %s from customers firestore: %v", cid, err)
		return
	}
}
