// GCP Identity Platform tenants creation script
//
// It is used for one time creation of Identity Platform tenants for CMP customers
package scripts

import (
	"context"
	"fmt"

	// "strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	tenantIDPath string = "auth.tenantId"
)

type CreateTenantsRequest struct {
	Overwrite bool `json:"overwrite"`
}

// TenantsCreator creates a dedicated tenant for each customer & partner. Important to make sure
// that Partners struct populating all the data in '/app/partner-access' doc as it going to overwrite it
type TenantsCreator struct {
	fs        *firestore.Client
	fbAuth    *auth.Client
	overwrite bool
	l         logger.ILogger
}

func NewTenantsCreator(ctx context.Context, projectID string, overwrite bool, l logger.ILogger) (*TenantsCreator, error) {
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

	return &TenantsCreator{
		fs:        fs,
		fbAuth:    fbAuth,
		overwrite: overwrite,
		l:         l,
	}, nil
}

func CreateTenants(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	var errs []error

	var req CreateTenantsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return append(errs, err)
	}

	tc, err := NewTenantsCreator(ctx, common.ProjectID, req.Overwrite, l)
	if err != nil {
		return append(errs, err)
	}

	l.Info("tenant creation script start....")

	if tc.overwrite {
		errs = tc.DeleteAllTenants(ctx)
		if len(errs) > 0 {
			tc.l.Warningf("delete tenants errors: %v", errs)
		}
	}

	errs = tc.CreateCustomersTenants(ctx)
	errs = append(errs, tc.CreatePartenrsTenants(ctx)...)

	l.Info("tenant creation script finished")

	return errs
}

func (tc *TenantsCreator) createTenant(ctx context.Context, displayNameSuffix string) (*auth.Tenant, error) {
	var displayName string
	if len(displayNameSuffix) > 18 {
		displayName = fmt.Sprintf("t-%s", displayNameSuffix[:18])
	} else {
		displayName = fmt.Sprintf("t-%s", displayNameSuffix)
	}

	config := (&auth.TenantToCreate{}).DisplayName(displayName)
	config.AllowPasswordSignUp(true)

	return tc.fbAuth.TenantManager.CreateTenant(ctx, config)
}

func (tc *TenantsCreator) updateCustomerTenantID(ctx context.Context, ref *firestore.DocumentRef, tenantID string) error {
	_, err := ref.Update(ctx, []firestore.Update{{
		Path:  tenantIDPath,
		Value: tenantID,
	}})

	return err
}

func (tc *TenantsCreator) CreateCustomersTenants(ctx context.Context) []error {
	customersRefs, err := tc.fs.Collection("customers").DocumentRefs(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	var errs []error

	for _, customerRef := range customersRefs {
		customerSnap, err := customerRef.Get(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get customer's snap %q: %w", customerRef.ID, err))
			continue
		}

		if !tc.overwrite {
			tenantID, err := customerSnap.DataAt(tenantIDPath)
			if err == nil {
				tc.l.Infof("tenant already exists for customer: %s. tenantId: %v", customerRef.ID, tenantID)
				continue
			}
		}

		tenant, err := tc.createTenant(ctx, customerRef.ID)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create identity platform tenant for customer %q: %w", customerRef.ID, err))
			continue
		}

		if err := tc.updateCustomerTenantID(ctx, customerRef, tenant.ID); err != nil {
			errs = append(errs, fmt.Errorf("faield to add tenant-id %q to customer %q document: %w", tenant.ID, customerRef.ID, err))
		}
	}

	return errs
}

func (tc *TenantsCreator) setPartnersDoc(ctx context.Context, ref *firestore.DocumentRef, partners common.Partners) error {
	_, err := ref.Set(ctx, partners)

	return err
}

func (tc *TenantsCreator) CreatePartenrsTenants(ctx context.Context) []error {
	var errs []error

	partnersSnap, err := tc.fs.Collection("app").Doc("partner-access").Get(ctx)
	if err != nil {
		return append(errs, fmt.Errorf("failed fetching 'partner-access' doc: %w", err))
	}

	var partners common.Partners
	if err := partnersSnap.DataTo(&partners); err != nil {
		return append(errs, fmt.Errorf("failed to reading partners doc: %w", err))
	}

	// for _, partner := range partners.Partners {
	// if !tc.overwrite && len(partner.Auth.TenantID) > 0 {
	// 	tc.logger.Infof("tenant already exists for customer: %s. tenantId: %v", partner.Name, partner.Auth.TenantID)
	// 	continue
	// }

	// tenant, err := tc.createTenant(ctx, strings.Replace(partner.Name, ".", "-", -1))
	// if err != nil {
	// 	errs = append(errs, fmt.Errorf("failed to create identity platform tenant for customer %q: %w", "ref.ID", err))
	// 	continue
	// }

	// partner.Auth.TenantID = tenant.ID
	// }

	err = tc.setPartnersDoc(ctx, partnersSnap.Ref, partners)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to set partners doc: %v. error: %w", partners, err))
	}

	return errs
}

func (tc *TenantsCreator) DeleteAllTenants(ctx context.Context) []error {
	iter := tc.fbAuth.TenantManager.Tenants(ctx, "")

	var errs []error

	for {
		tenant, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = tc.fbAuth.TenantManager.DeleteTenant(ctx, tenant.ID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errs
}
