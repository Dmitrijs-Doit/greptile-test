package amazonwebservices

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudhealth"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

const (
	IntegrationsCloudHealthCustomersCollection = "integrations/cloudhealth/cloudhealthCustomers"
	integrationsCloudHealthInstancesCollection = "integrations/cloudhealth/cloudhealthInstances"
)

type IntegrationCloudHealthCustomer struct {
	ID        int64                  `firestore:"id"`
	Name      string                 `firestore:"name"`
	Customer  *firestore.DocumentRef `firestore:"customer"`
	Disabled  bool                   `firestore:"disabled"`
	Timestamp time.Time              `firestore:"timestamp,serverTimestamp"`
}

func SyncCloudhealthCustomers(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	customers := make(map[int64]*cloudhealth.Customer)
	if err := cloudhealth.ListCustomers(1, customers); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	cloudhealthCustomersRef := fs.Collection(IntegrationsCloudHealthCustomersCollection)
	batch := fb.NewAutomaticWriteBatch(fs, 100)

	for _, customer := range customers {
		priorityID := strings.TrimSpace(customer.Address.ZipCode)

		docSnaps, err := fs.Collection("entities").
			Where("priorityId", "==", priorityID).
			Limit(1).
			SelectPaths([]string{"customer"}).
			Documents(ctx).GetAll()
		if err != nil {
			l.Error(err)
			continue
		}

		var customerRef *firestore.DocumentRef

		if len(docSnaps) <= 0 {
			l.Warningf("customer %s has no matching entity for '%s'", customer.Name, priorityID)

			customerRef = fb.Orphan
		} else {
			if p, err := docSnaps[0].DataAt("customer"); err != nil {
				l.Error(err)
				continue
			} else {
				customerRef = p.(*firestore.DocumentRef)
			}
		}

		chtCustomerID := strconv.FormatInt(customer.ID, 10)
		ref := cloudhealthCustomersRef.Doc(chtCustomerID)

		batch.Set(ref, IntegrationCloudHealthCustomer{
			ID:       customer.ID,
			Name:     customer.Name,
			Customer: customerRef,
			Disabled: false,
		})
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		for _, err := range errs {
			l.Errorf("batch.Commit err: %v", err)
		}
	}

	docs, err := cloudhealthCustomersRef.Documents(ctx).GetAll()
	if err != nil {
		l.Error(err)
		return
	}

	// mark all items that still exist in FS but not in CHT as disabled
	for _, doc := range docs {
		ID, err := strconv.ParseInt(doc.Ref.ID, 10, 64)
		if err != nil {
			l.Error(err)
			continue
		}

		if _, ok := customers[ID]; ok {
			continue
		}

		var customer IntegrationCloudHealthCustomer
		if err := doc.DataTo(&customer); err != nil {
			l.Error(err)
			continue
		} else if customer.Disabled {
			err = disableInstanceDoc(ctx, fs, l, doc.Ref.ID)
			if err != nil {
				l.Error(err)
			}

			continue
		}

		l.Infof("marking cloudhealth customer %v as disabled", ID)

		_, err = doc.Ref.Update(ctx, []firestore.Update{{FieldPath: []string{"disabled"}, Value: true}})
		if err != nil {
			l.Error(err)
			continue
		}

		err = disableInstanceDoc(ctx, fs, l, doc.Ref.ID)
		if err != nil {
			l.Error(err)
		}
	}
}

func disableInstanceDoc(ctx *gin.Context, fs *firestore.Client, l logger.ILogger, ID string) error {
	instanceDoc, err := fs.Collection(integrationsCloudHealthInstancesCollection).Doc(ID).Get(ctx)
	if err != nil {
		return err
	}

	var instance struct {
		Disabled bool `firestore:"disabled"`
	}

	if err := instanceDoc.DataTo(&instance); err != nil {
		return err
	} else if instance.Disabled {
		return nil
	}

	_, err = instanceDoc.Ref.Update(ctx, []firestore.Update{{FieldPath: []string{"disabled"}, Value: true}})
	if err != nil {
		return err
	}

	l.Infof("marked cloudhealth instance %v as disabled", ID)

	return nil
}
