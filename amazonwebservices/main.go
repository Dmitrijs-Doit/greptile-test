package amazonwebservices

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudhealth"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

type Asset struct {
	AssetType  string                 `firestore:"type"`
	Properties *pkg.AWSProperties     `firestore:"properties"`
	Bucket     *firestore.DocumentRef `firestore:"bucket"`
	Contract   *firestore.DocumentRef `firestore:"contract"`
	Customer   *firestore.DocumentRef `firestore:"customer"`
	Entity     *firestore.DocumentRef `firestore:"entity"`
	Discovery  string                 `firestore:"discovery"`
	Tags       []string               `firestore:"tags"`

	Snapshot *firestore.DocumentSnapshot `firestore:"-"`
}

type Sauron struct {
	AccountNumber string      `json:"account_number"`
	Name          string      `json:"name"`
	Email         string      `json:"email"`
	Owner         SauronOwner `json:"owner"`
	Payer         string      `json:"payer,omitempty"`
	HasRole       bool        `json:"has_role,omitempty"`
	ErrorDetail   bool        `json:"detail,omitempty"`
}

type SauronOwner struct {
	Name       string `json:"name"`
	DomainName string `json:"domain_name"`
}

const (
	devSauronURL  = "https://api.sauron-dev.leepackham.com/api/cmp/"
	prodSauronURL = "https://api.sauron.doit-intl.com/api/cmp/"
)

var APIKey string

func init() {
	ctx := context.Background()

	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretSauronAPIKey)
	if err != nil {
		return
	}

	APIKey = string(secret)
}

func (a Asset) GetCacheKey() string {
	return fmt.Sprintf("%s-%s-%s", a.AssetType, a.Entity.ID, a.Properties.AccountID)
}

func (a Asset) ModifyContractQuery(query *firestore.Query) firestore.Query {
	return query.Where("type", "==", a.AssetType)
}

func (a Asset) GetCloudHealthCustomerID() int64 {
	if a.Properties.CloudHealth != nil {
		return a.Properties.CloudHealth.CustomerID
	}

	return 0
}

type httpError struct {
	statusCode int
	msg        string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("http error: [%d] %s", e.statusCode, e.msg)
}

func (a Asset) ContractPredicate(contract *common.Contract) (bool, bool) {
	if len(contract.Assets) > 0 {
		docID := fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, a.Properties.AccountID)
		for _, ref := range contract.Assets {
			if ref != nil && ref.ID == docID {
				return true, false
			}
		}

		return false, false
	}

	return true, true
}

func (s *AWSService) UpdateAssetsSharedPayers(ctx context.Context) error {
	fs := s.conn.Firestore(ctx)
	logger := s.loggerProvider(ctx)
	assetType := common.Assets.AmazonWebServices

	// Fetch CHT customers
	customers := make(map[int64]*cloudhealth.Customer)
	if err := cloudhealth.ListCustomers(1, customers); err != nil {
		return err
	}

	assetSettings := make(map[string]*pkg.AWSAssetSettings)

	assetSettingsIter := fs.Collection("assetSettings").Where("type", "==", assetType).Documents(ctx)
	defer assetSettingsIter.Stop()

	logger.Infof("Asset Discovery - shared - processing:")

	for {
		docSnap, err := assetSettingsIter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return err
		}

		var asDoc pkg.AWSAssetSettings
		if err := docSnap.DataTo(&asDoc); err != nil {
			log.Printf("Error: %s", err.Error())
		} else {
			assetSettings[docSnap.Ref.ID] = &asDoc
		}
	}

	masterPayerAccounts, err := dal.GetMasterPayerAccounts(ctx, fs)
	if err != nil {
		logger.Errorf("failed to fetch master payer accounts: %s", err)
		return err
	}

	flexsaveAccountIDs, err := s.flexsaveAPI.ListFlexsaveAccounts(ctx)
	if err != nil {
		return err
	}

	for _, chtCustomer := range customers {
		logger.Debugf("Asset Discovery - shared - customer name %s (%d)", chtCustomer.Name, chtCustomer.ID)

		batch := fb.NewAutomaticWriteBatch(fs, 100)

		priorityID := strings.TrimSpace(chtCustomer.Address.ZipCode)

		// Skip customers that are marked to be skipped using "cmp_skip_customer" tag key.
		if value := chtCustomer.GetTagValue("cmp_skip_customer"); value != nil {
			continue
		}

		docSnaps, err := fs.Collection("entities").
			Where("priorityId", "==", priorityID).
			Limit(1).
			Select("customer", "active").
			Documents(ctx).GetAll()
		if err != nil {
			logger.Errorf("[aws assets] failed to fetch entity with error: %s", err)
			continue
		}

		if len(docSnaps) <= 0 {
			logger.Errorf("[aws assets] entity not found for priority id %s", priorityID)
			continue
		}

		docSnap := docSnaps[0]

		var entityRef *firestore.DocumentRef

		var customerRef = fb.Orphan

		var entity common.Entity
		if err := docSnap.DataTo(&entity); err != nil {
			logger.Error(err)
			continue
		}

		customerRef = entity.Customer

		if entity.Active {
			entityRef = docSnap.Ref
		}

		awsAccounts := make(map[int64]*cloudhealth.AwsAccount)
		if err := cloudhealth.ListAccounts(1, awsAccounts, chtCustomer); err != nil {
			logger.Errorf("[aws assets] failed to fetch aws accounts for %s with error: %s", chtCustomer.Name, err)
			continue
		}

		var allAccounts []string

		for _, awsAccount := range awsAccounts {
			if awsAccount.OwnerID == "" {
				continue
			}
			// append all accounts
			allAccounts = append(allAccounts, awsAccount.OwnerID)
			// Do not create assets for Flexsave accounts
			if slice.Contains(flexsaveAccountIDs, awsAccount.OwnerID) {
				continue
			}

			docID := fmt.Sprintf("%s-%s", assetType, awsAccount.OwnerID)
			accountOrg := GetAccountOrganization(ctx, fs, docID)
			assetRef := fs.Collection("assets").Doc(docID)
			assetSettingsRef := fs.Collection("assetSettings").Doc(docID)
			paths := []firestore.FieldPath{[]string{"type"}, []string{"properties"}, []string{"customer"}}

			if accountOrg != nil && accountOrg.PayerAccount != nil {
				if mpa, ok := masterPayerAccounts.Accounts[accountOrg.PayerAccount.AccountID]; ok {
					if mpa.IsDedicatedPayer() {
						continue
					}
				} else {
					logger.Errorf("[aws assets] master payer account not found for %s", accountOrg.PayerAccount.AccountID)
					continue
				}
			}

			var as *pkg.AWSAssetSettings
			if v, prs := assetSettings[docID]; prs {
				as = v
			} else {
				as = &pkg.AWSAssetSettings{
					BaseAsset: pkg.BaseAsset{
						AssetType: assetType,
					},
				}
			}

			var supportSettings *pkg.AWSSettingsSupport

			if as.Settings != nil {
				supportSettings = &as.Settings.Support
			}

			props := &pkg.AWSProperties{
				AccountID:    awsAccount.OwnerID,
				Name:         awsAccount.AmazonName,
				FriendlyName: awsAccount.Name,
				CloudHealth: &pkg.CloudHealthAccountInfo{
					CustomerName: chtCustomer.Name,
					CustomerID:   chtCustomer.ID,
					AccountID:    awsAccount.ID,
					ExternalID:   awsAccount.Authentication.AssumeRoleExternalID,
					Status:       awsAccount.Status.Level,
				},
				Support: supportSettings,
			}

			if accountOrg != nil {
				props.SauronRole = GetSauronRole(ctx, accountOrg, customerRef)
				props.OrganizationInfo = &pkg.OrganizationInfo{
					PayerAccount: accountOrg.PayerAccount,
					Status:       accountOrg.Status,
					Email:        accountOrg.Email,
				}
			}

			asset := Asset{
				AssetType:  assetType,
				Customer:   customerRef,
				Properties: props,
			}

			if customerRef.ID != fb.Orphan.ID {
				var bucketRef *firestore.DocumentRef

				assetSettingsUpdate := map[string]interface{}{}

				if as.Customer != nil && as.Customer.ID == customerRef.ID && as.Entity != nil {
					entityRef = as.Entity
					bucketRef = as.Bucket
				} else {
					assetSettingsUpdate["type"] = assetType
					assetSettingsUpdate["customer"] = customerRef
					assetSettingsUpdate["entity"] = entityRef
					assetSettingsUpdate["bucket"] = bucketRef
					assetSettingsUpdate["contract"] = nil
					assetSettingsUpdate["timeCreated"] = firestore.ServerTimestamp
				}

				asset.Tags = as.Tags
				asset.Entity = entityRef
				asset.Bucket = bucketRef

				paths = append(paths, []string{"entity"}, []string{"bucket"}, []string{"tags"})

				if contractRef, update := common.GetAssetContract(ctx, fs, asset, customerRef, entityRef, nil); update {
					assetSettingsUpdate["contract"] = contractRef
					asset.Contract = contractRef

					paths = append(paths, []string{"contract"})
				}

				if len(assetSettingsUpdate) > 0 {
					batch.Set(assetSettingsRef, assetSettingsUpdate, firestore.MergeAll)
				}
			} else {
				// Could not find customer, update settings to reference orphan customer and reset entity, contract
				batch.Set(assetSettingsRef, map[string]interface{}{
					"customer":    customerRef,
					"entity":      nil,
					"contract":    nil,
					"bucket":      nil,
					"tags":        firestore.Delete,
					"type":        assetType,
					"timeCreated": firestore.ServerTimestamp,
				}, firestore.MergeAll)

				asset.Entity = nil
				asset.Contract = nil
				asset.Bucket = nil

				paths = append(paths, []string{"entity"}, []string{"contract"}, []string{"bucket"})
			}

			batch.Set(assetRef, asset, firestore.Merge(paths...))
			batch.Set(assetRef.Collection("assetMetadata").Doc("metadata"), map[string]interface{}{
				"lastUpdated": firestore.ServerTimestamp,
				"type":        asset.AssetType,
			})
		}

		logger.Infof("Asset Discovery - shared - customer ID %v, fetched assets %v", customerRef.ID, allAccounts)

		if errs := batch.Commit(ctx); len(errs) > 0 {
			logger.Errorf("batch.Commit err: %v", errs[0])
		}

		time.Sleep(time.Millisecond * time.Duration(50))
	}

	return nil
}

func GetSauronRole(ctx context.Context, accountOrg *Account, customerRef *firestore.DocumentRef) bool {
	url := fmt.Sprintf("%s%s", GetSauronURL(), accountOrg.ID)

	sauron, err := makeSauronReq(ctx, url, "GET", nil)
	if sauron == nil {
		// No account - create
		sauron, err = createSauronAccount(ctx, accountOrg, customerRef)
		if err != nil {
			return false
		}
	}

	return sauron.HasRole
}

func createSauronAccount(ctx context.Context, accountOrg *Account, customerRef *firestore.DocumentRef) (*Sauron, error) {
	docSnap, err := customerRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var customer common.Customer
	if err := docSnap.DataTo(&customer); err != nil {
		return nil, err
	}

	url := GetSauronURL()

	if accountOrg != nil && accountOrg.PayerAccount != nil {
		sauron := &Sauron{
			AccountNumber: accountOrg.ID,
			Email:         accountOrg.Email,
			Name:          accountOrg.Name,
			Owner: SauronOwner{
				Name:       customer.Name,
				DomainName: customer.PrimaryDomain,
			},
			Payer: accountOrg.PayerAccount.AccountID,
		}

		return makeSauronReq(ctx, url, "POST", sauron)
	}

	return nil, errors.New("accountOrg is empty")
}

func makeSauronReq(ctx context.Context, url string, method string, sauronData *Sauron) (*Sauron, error) {
	client := http.DefaultClient

	var req *http.Request

	if method == "POST" {
		body, _ := json.Marshal(sauronData)
		req, _ = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)

	defer resp.Body.Close()

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		httpErr := &httpError{resp.StatusCode, string(respBody)}
		return nil, httpErr
	}

	sauron := &Sauron{}
	if err := json.Unmarshal(respBody, sauron); err != nil {
		return nil, err
	}

	return sauron, nil
}

func GetSauronURL() string {
	if common.Production {
		return prodSauronURL
	}

	return devSauronURL
}

func GetSupportRole(ctx *gin.Context) {
	accountID := ctx.Request.URL.Query()["account"]
	url := fmt.Sprintf("%s%s", GetSauronURL(), accountID)
	sauron, err := makeSauronReq(ctx, url, "GET", nil)

	if httpErr, ok := err.(*httpError); ok {
		if httpErr.statusCode == http.StatusNotFound {
			ctx.JSON(http.StatusOK, false)
			return
		}

		ctx.AbortWithError(httpErr.statusCode, httpErr)

		return
	}

	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if sauron == nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	ctx.JSON(http.StatusOK, sauron.HasRole)
}
