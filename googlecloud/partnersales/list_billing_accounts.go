package partnersales

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/channel/apiv1/channelpb"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"golang.org/x/time/rate"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
)

const (
	bucketSize int = 20
)

// BillingAccountsList fetches all PartnerSales Console entitelments (billing accounts)
// using Cloud Channel API. Cloud task is being schduled per every bucket of 20 accounts.
// Each scheduled task executes BillingAccountsPageHandler
func (s *GoogleChannelService) BillingAccountsList(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	gcpOffer, err := s.selectGCPOffer(ctx)
	if err != nil {
		return err
	}

	go func() {
		if err := s.handleBillingAccountsList(ctx, gcpOffer); err != nil {
			l.Errorf("failed to handle billing accounts list: %s", err)
		}
	}()

	return nil
}

// handleBillingAccountsList
func (s *GoogleChannelService) handleBillingAccountsList(ctx context.Context, gcpOffer *channelpb.Offer) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)
	client := s.cloudChannel

	limiter := rate.NewLimiter(0.5, 1) // 30 requests per minute
	bucketNum := 1
	billingAccounts := make([]googlecloud.AssetUpdateRequest, 0)

	cIterator := client.ListCustomers(ctx, &channelpb.ListCustomersRequest{
		Parent:   partnerAccountName,
		PageSize: 50,
	})

	for {
		channelCustomer, err := cIterator.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			l.Errorf("failed to get next customer: %s", err)
			return err
		}

		var customerID string

		docSnap, err := fs.Collection("integrations").
			Doc("google-cloud").
			Collection("googlePartnerSalesCustomers").
			Doc(s.getChannelCustomerID(channelCustomer)).Get(ctx)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				l.Errorf("failed to get customer doc: %s", err)
				return err
			}

			customerID = fb.OrphanID
		} else {
			var customerDoc ChannelServicesCustomer
			if err := docSnap.DataTo(&customerDoc); err != nil {
				l.Errorf("failed to convert customer doc: %s", err)
				return err
			}

			customerID = customerDoc.Customer.ID
		}

		eIterator := client.ListEntitlements(ctx, &channelpb.ListEntitlementsRequest{
			Parent:   channelCustomer.Name,
			PageSize: 100,
		})

		for {
			entitlement, err := eIterator.Next()
			if err == iterator.Done {
				break
			}

			if err != nil {
				l.Errorf("failed to get next entitlement: %s", err)
				return err
			}

			if entitlement.GetOffer() != gcpOffer.GetName() {
				continue
			}

			provisioningState := entitlement.GetProvisioningState()
			switch provisioningState {
			case channelpb.Entitlement_ACTIVE, channelpb.Entitlement_SUSPENDED:
			default:
				continue
			}

			channelBillingAccount, err := s.newChannelBillingAccount(entitlement)
			if err != nil {
				return err
			}

			billingAccount, err := channelBillingAccount.ToGCPBillingAccount()
			if err != nil {
				l.Errorf("failed to convert channel billing account to gcp billing account: %s", err)
				continue
			}

			billingAccounts = append(billingAccounts, googlecloud.AssetUpdateRequest{
				CustomerID:     customerID,
				BillingAccount: *billingAccount,
			})

			if len(billingAccounts) >= bucketSize {
				if err := s.scheduleAssetsUpdateTask(ctx, &billingAccounts, bucketNum); err != nil {
					l.Errorf("failed to schedule assets update task: %s", err)
				}

				bucketNum++
				billingAccounts = nil
			}
		}

		if err := limiter.Wait(ctx); err != nil {
			l.Errorf("failed to wait for rate limiter: %s", err)
		}
	}

	if len(billingAccounts) > 0 {
		if err := s.scheduleAssetsUpdateTask(ctx, &billingAccounts, bucketNum); err != nil {
			l.Errorf("failed to schedule assets update task: %s", err)
		}
	}

	return nil
}

func (s *GoogleChannelService) scheduleAssetsUpdateTask(ctx context.Context, billingAccounts *[]googlecloud.AssetUpdateRequest, bucketNum int) error {
	taskBody, err := json.Marshal(googlecloud.AssetsUpdateRequest{
		Assets: *billingAccounts,
		Type:   common.Assets.GoogleCloud,
	})
	if err != nil {
		return err
	}

	scheduleTime := time.Now().Add(time.Second * time.Duration(10*bucketNum))

	config := common.CloudTaskConfig{
		Method:       cloudtaskspb.HttpMethod_POST,
		Path:         "/tasks/assets/google-cloud",
		Queue:        common.TaskQueueAssetsGCP,
		Body:         taskBody,
		ScheduleTime: common.TimeToTimestamp(scheduleTime),
	}

	if _, err = common.CreateCloudTask(ctx, &config); err != nil {
		return err
	}

	return nil
}
