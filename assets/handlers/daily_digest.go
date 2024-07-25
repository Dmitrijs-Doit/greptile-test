package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/gsuite"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

type asset struct {
	ID             string               `json:"id"`
	TypeLabel      string               `json:"type_label"`
	TypeID         string               `json:"type_id"`
	Label          string               `json:"label"`
	Customer       assetCustomer        `json:"customer,omitempty"`
	AccountManager *assetAccountManager `json:"accountManager,omitempty"`
}

type assetCustomer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type assetAccountManager struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type entity struct {
	ID         string `json:"id"`
	CustomerID string `json:"customerId"`
	Name       string `json:"name"`
	PriorityID string `json:"priorityId"`
	Reason     string `json:"reason"`
}

type baseAssetWithProperties struct {
	pkg.BaseAsset
	Properties map[string]interface{} `firestore:"properties"`
}

func DailyDigestHandler(ctx *gin.Context, conn *connection.Connection) error {
	l := logger.FromContext(ctx)
	fs := conn.Firestore(ctx)

	today := times.CurrentDayUTC()

	customers := make(map[string]*common.Customer)
	accountManagers := make(map[string]*common.AccountManager)
	orphanedAssets := make([]*asset, 0)
	unassignedAssets := make([]*asset, 0)
	uncontractedAssets := make([]*asset, 0)
	unassignedContracts := make([]*asset, 0)
	invalidEntities := make([]*entity, 0)

	accountManagersIterator := fs.Collection("accountManagers").
		Where("company", "==", common.AccountManagerCompanyDoit).
		Documents(ctx)
	defer accountManagersIterator.Stop()

	for {
		docSnap, err := accountManagersIterator.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			l.Errorf("accountManagers iterator failed with error: %s", err)
			return err
		}

		var amDoc common.AccountManager

		if err := docSnap.DataTo(&amDoc); err != nil {
			l.Errorf("failed to populate account manager doc with error: %s", err)
			return err
		}

		accountManagers[docSnap.Ref.ID] = &amDoc
	}

	orphan, err := common.GetCustomer(ctx, fb.Orphan)
	if err != nil {
		return err
	}

	assetsIterator := fs.Collection("assets").
		Where("type", "in", []string{
			common.Assets.GoogleCloud,
			common.Assets.AmazonWebServices,
			common.Assets.MicrosoftAzure,
			common.Assets.Office365,
			common.Assets.GSuite,
		}).
		Where("entity", "==", nil).
		Documents(ctx)
	defer assetsIterator.Stop()

	for {
		docSnap, err := assetsIterator.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			l.Errorf("asset iteartor failed with error: %s", err)
			break
		}

		var assetDoc baseAssetWithProperties

		if err := docSnap.DataTo(&assetDoc); err != nil {
			l.Errorf("failed to populate assetDoc with error: %s", err)
			continue
		}

		customerRef := assetDoc.Customer
		label := getAssetDescription(&assetDoc, docSnap.Ref.ID)
		assetType := assetDoc.AssetType
		assetTypeLabel := getProductTypeLabel(assetType)

		if customerRef.ID == fb.Orphan.ID {
			orphanedAssets = append(orphanedAssets, &asset{
				ID:        docSnap.Ref.ID,
				Label:     label,
				TypeID:    assetType,
				TypeLabel: assetTypeLabel,
				Customer: assetCustomer{
					ID:   fb.Orphan.ID,
					Name: orphan.Name,
				},
			})

			continue
		}

		// Skip msp.doit.com customer
		if customerRef.ID == "6QJMHMUaIYdEShihSweH" {
			continue
		}

		if customer, ok := customers[customerRef.ID]; ok {
			if customer.AccountManager != nil {
				if accountManager, ok := accountManagers[customer.AccountManager.ID]; ok {
					unassignedAssets = append(unassignedAssets, &asset{
						ID:        docSnap.Ref.ID,
						Label:     label,
						TypeID:    assetType,
						TypeLabel: assetTypeLabel,
						Customer: assetCustomer{
							ID:   customerRef.ID,
							Name: customer.Name,
						},
						AccountManager: &assetAccountManager{
							Name:  accountManager.Name,
							Email: accountManager.Email,
						},
					})

					continue
				}
			}

			unassignedAssets = append(unassignedAssets, &asset{
				ID:        docSnap.Ref.ID,
				Label:     label,
				TypeID:    assetType,
				TypeLabel: assetTypeLabel,
				Customer: assetCustomer{
					ID:   customerRef.ID,
					Name: customer.Name,
				},
			})
		} else {
			customer, err := common.GetCustomer(ctx, customerRef)
			if err != nil {
				unassignedAssets = append(unassignedAssets, &asset{
					ID:        docSnap.Ref.ID,
					Label:     label,
					TypeID:    assetType,
					TypeLabel: assetTypeLabel,
					Customer: assetCustomer{
						ID:   customerRef.ID,
						Name: "N/A",
					},
				})

				continue
			}

			customers[customerRef.ID] = customer

			if customer.AccountManager != nil {
				if accountManager, ok := accountManagers[customer.AccountManager.ID]; ok {
					unassignedAssets = append(unassignedAssets, &asset{
						ID:        docSnap.Ref.ID,
						Label:     label,
						TypeID:    assetType,
						TypeLabel: assetTypeLabel,
						Customer: assetCustomer{
							ID:   customerRef.ID,
							Name: customer.Name,
						},
						AccountManager: &assetAccountManager{
							Name:  accountManager.Name,
							Email: accountManager.Email,
						},
					})

					continue
				}
			}

			unassignedAssets = append(unassignedAssets, &asset{
				ID:        docSnap.Ref.ID,
				Label:     label,
				TypeID:    assetType,
				TypeLabel: assetTypeLabel,
				Customer: assetCustomer{
					ID:   customerRef.ID,
					Name: customer.Name,
				},
			})
		}
	}

	contractsIterator := fs.Collection("contracts").
		Where("active", "==", true).
		Where("entity", "==", nil).
		Where("type", common.NotIn, []string{
			common.Assets.GoogleCloudStandalone,
			common.Assets.AmazonWebServicesStandalone,
		}).
		Documents(ctx)

	defer contractsIterator.Stop()

	for {
		docSnap, err := contractsIterator.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			l.Errorf("contract iterator failed with error: %s", err)
			break
		}

		var contractDoc = make(map[string]interface{})

		if err := docSnap.DataTo(&contractDoc); err != nil {
			l.Errorf("failed to populate contractDoc with error: %s", err)
			continue
		}

		customerRef := contractDoc["customer"].(*firestore.DocumentRef)
		accountManagerRef := contractDoc["accountManager"].(*firestore.DocumentRef)
		contractType := contractDoc["type"].(string)
		contractTypeLabel := getProductTypeLabel(contractType)

		var accountManager *assetAccountManager

		if accountManagerRef != nil {
			if am, ok := accountManagers[accountManagerRef.ID]; ok {
				accountManager = &assetAccountManager{
					Name:  am.Name,
					Email: am.Email,
				}
			}
		}

		if customer, ok := customers[customerRef.ID]; ok {
			unassignedContracts = append(unassignedContracts, &asset{
				ID:        docSnap.Ref.ID,
				Label:     "",
				TypeID:    contractType,
				TypeLabel: contractTypeLabel,
				Customer: assetCustomer{
					ID:   customerRef.ID,
					Name: customer.Name,
				},
				AccountManager: accountManager,
			})
		} else {
			customer, err := common.GetCustomer(ctx, customerRef)
			if err != nil {
				l.Errorf("failed to get customer %s with error: %s", customerRef.ID, err)

				unassignedContracts = append(unassignedContracts, &asset{
					ID:        docSnap.Ref.ID,
					Label:     "",
					TypeID:    contractType,
					TypeLabel: contractTypeLabel, Customer: assetCustomer{
						ID:   customerRef.ID,
						Name: "N/A",
					},
					AccountManager: accountManager,
				})

				continue
			}

			customers[customerRef.ID] = customer
			unassignedContracts = append(unassignedContracts, &asset{
				ID:        docSnap.Ref.ID,
				Label:     "",
				TypeID:    contractType,
				TypeLabel: contractTypeLabel,
				Customer: assetCustomer{
					ID:   customerRef.ID,
					Name: customer.Name,
				},
				AccountManager: accountManager,
			})

			continue
		}
	}

	entitiesIterator := fs.Collection("entities").
		Where("invoicing.mode", "==", "CUSTOM").
		Where("invoicing.default", "==", nil).
		Documents(ctx)
	defer entitiesIterator.Stop()

	for {
		docSnap, err := entitiesIterator.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			l.Errorf("entity iterator failed with error: %s", err)
			break
		}

		var e common.Entity

		if err := docSnap.DataTo(&e); err != nil {
			l.Errorf("failed to populate entity doc with error: %s", err)
			continue
		}

		invalidEntities = append(invalidEntities, &entity{
			ID:         docSnap.Ref.ID,
			CustomerID: e.Customer.ID,
			Name:       e.Name,
			PriorityID: e.PriorityID,
			Reason:     "Invoice using custom bucketing missing default bucket",
		})
	}

	gsuiteAssetIter := fs.Collection("assets").
		Where("type", "==", common.Assets.GSuite).
		Where("contract", "==", nil).
		OrderBy("customer", firestore.Asc).
		Documents(ctx)
	defer gsuiteAssetIter.Stop()

	for {
		docSnap, err := gsuiteAssetIter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			l.Errorf("gsuite asset iterator failed with error: %s", err)
			break
		}

		var gsuiteAsset gsuite.Asset

		if err := docSnap.DataTo(&gsuiteAsset); err != nil {
			l.Errorf("failed to populate gsuite asset with error: %s", err)
			continue
		}

		if gsuiteAsset.Entity == nil {
			continue
		}

		contractDocs, err := fs.Collection("contracts").
			Where("type", "==", common.Assets.GSuite).
			Where("customer", "==", gsuiteAsset.Customer).
			Where("entity", "==", gsuiteAsset.Entity).
			Where("active", "==", true).
			Documents(ctx).GetAll()
		if err != nil {
			l.Errorf("failed to get contracts docs with error: %s", err)
			continue
		}

		for _, contractDocSnap := range contractDocs {
			var contract common.Contract

			if err := contractDocSnap.DataTo(&contract); err != nil {
				l.Errorf("failed to populate contract doc with error: %s", err)
				continue
			}

			if today.After(contract.StartDate) || today.Equal(contract.StartDate) {
				if !contract.IsCommitment || today.Before(contract.EndDate) {
					if _, prs := customers[gsuiteAsset.Customer.ID]; !prs {
						c, err := common.GetCustomer(ctx, gsuiteAsset.Customer)
						if err != nil {
							l.Errorf("failed to get customer %s with error: %s", gsuiteAsset.Customer.ID, err)
							break
						}

						customers[gsuiteAsset.Customer.ID] = c
					}

					customer := customers[gsuiteAsset.Customer.ID]
					uncontractedAssets = append(uncontractedAssets, &asset{
						ID:        docSnap.Ref.ID,
						Label:     fmt.Sprintf("%s (%s)", gsuiteAsset.Properties.Subscription.SkuName, gsuiteAsset.Properties.CustomerDomain),
						TypeID:    gsuiteAsset.AssetType,
						TypeLabel: "Google Workspace",
						Customer: assetCustomer{
							ID:   gsuiteAsset.Customer.ID,
							Name: customer.Name,
						},
					})

					break
				}
			}
		}
	}

	l.Infof("%d Orphaned assets", len(orphanedAssets))
	l.Infof("%d Unassigned assets", len(unassignedAssets))
	l.Infof("%d Unassigned contracts", len(unassignedContracts))
	l.Infof("%d Invalid entities", len(invalidEntities))
	l.Infof("%d Uncontracted assets", len(uncontractedAssets))

	sort.Slice(orphanedAssets, func(i, j int) bool {
		if orphanedAssets[i].TypeID < orphanedAssets[j].TypeID {
			return true
		}

		if orphanedAssets[i].TypeID > orphanedAssets[j].TypeID {
			return false
		}

		return orphanedAssets[i].Label < orphanedAssets[j].Label
	})

	sort.Slice(unassignedAssets, func(i, j int) bool {
		if unassignedAssets[i].Customer.ID < unassignedAssets[j].Customer.ID {
			return true
		}

		if unassignedAssets[i].Customer.ID > unassignedAssets[j].Customer.ID {
			return false
		}

		if unassignedAssets[i].TypeID < unassignedAssets[j].TypeID {
			return true
		}

		if unassignedAssets[i].TypeID > unassignedAssets[j].TypeID {
			return false
		}

		return unassignedAssets[i].Label < unassignedAssets[j].Label
	})

	sort.Slice(unassignedContracts, func(i, j int) bool {
		if unassignedContracts[i].Customer.ID < unassignedContracts[j].Customer.ID {
			return true
		}

		if unassignedContracts[i].Customer.ID > unassignedContracts[j].Customer.ID {
			return false
		}

		return unassignedContracts[i].TypeID < unassignedContracts[j].TypeID
	})

	if len(orphanedAssets) > 0 || len(unassignedAssets) > 0 || len(unassignedContracts) > 0 {
		m := mail.NewV3Mail()
		m.SetTemplateID(mailer.Config.DynamicTemplates.AssetsDailyDigest)

		var enable bool

		m.SetTrackingSettings(&mail.TrackingSettings{SubscriptionTracking: &mail.SubscriptionTrackingSetting{Enable: &enable}})
		m.SetFrom(mail.NewEmail("DoiT International", "noreply@doit-intl.com"))

		p := mail.NewPersonalization()
		p.AddTos(mail.NewEmail("", "cmp-ops-daily-digest@doit-intl.com"))

		var orphanedAssetsData []map[string]interface{}

		if bytes, err := json.Marshal(orphanedAssets); err != nil {
			l.Errorf("failed to marshal orphanedAssets with error: %s", err)
			return err
		} else if err := json.Unmarshal(bytes, &orphanedAssetsData); err != nil {
			l.Errorf("failed to unmarshal orphanedAssets with error: %s", err)
			return err
		}

		p.SetDynamicTemplateData("orphanedAssets", orphanedAssetsData)

		var unassignedAssetsData []map[string]interface{}

		if bytes, err := json.Marshal(unassignedAssets); err != nil {
			l.Errorf("failed to marshal unassignedAssets with error: %s", err)
			return err
		} else if err := json.Unmarshal(bytes, &unassignedAssetsData); err != nil {
			l.Errorf("failed to unmarshal unassignedAssets with error: %s", err)
			return err
		}

		p.SetDynamicTemplateData("unassignedAssets", unassignedAssetsData)

		var unassignedContractsData []map[string]interface{}

		if bytes, err := json.Marshal(unassignedContracts); err != nil {
			l.Errorf("failed to marshal unassignedContracts with error: %s", err)
			return err
		} else if err := json.Unmarshal(bytes, &unassignedContractsData); err != nil {
			l.Errorf("failed to unmarshal unassignedContracts with error: %s", err)
			return err
		}

		p.SetDynamicTemplateData("unassignedContracts", unassignedContractsData)
		p.SetDynamicTemplateData("invalidEntities", invalidEntities)
		p.SetDynamicTemplateData("uncontractedAssets", uncontractedAssets)
		m.AddPersonalizations(p)

		request := sendgrid.GetRequest(mailer.Config.APIKey, mailer.Config.MailSendPath, mailer.Config.BaseURL)
		request.Method = http.MethodPost
		request.Body = mail.GetRequestBody(m)

		_, err := sendgrid.MakeRequestRetry(request)
		if err != nil {
			l.Errorf("failed to send email with error: %s", err)
		} else {
			l.Infof("Email sent successfully")
		}
	}

	return nil
}

func getAssetDescription(assetDoc *baseAssetWithProperties, docID string) string {
	props := assetDoc.Properties

	if props == nil {
		return docID
	}

	switch assetDoc.AssetType {
	case common.Assets.GSuite:
		return fmt.Sprintf("%s (%s)", props["subscription"].(map[string]interface{})["skuName"], props["customerDomain"])

	case common.Assets.GoogleCloud:
		return fmt.Sprintf("%s (%s)", props["displayName"], props["billingAccountId"])

	case common.Assets.GoogleCloudProject:
		return fmt.Sprintf("%s (%s)", props["projectId"], props["billingAccountId"])

	case common.Assets.AmazonWebServices:
		return fmt.Sprintf("%s (%s)", props["friendlyName"], props["accountId"])

	case common.Assets.MicrosoftAzure:
		return fmt.Sprintf("%s (%s)", props["subscription"].(map[string]interface{})["displayName"], props["customerDomain"])

	case common.Assets.Office365:
		return fmt.Sprintf("%s (%s)", props["subscription"].(map[string]interface{})["offerName"], props["customerDomain"])

	default:
		return docID
	}
}

func getProductTypeLabel(assetType string) string {
	switch assetType {
	case common.Assets.GSuite:
		return "Google Workspace"

	case common.Assets.GoogleCloud:
		return "Google Cloud"

	case common.Assets.GoogleCloudProject:
		return "Google Cloud Project"

	case common.Assets.AmazonWebServices:
		return "Amazon Web Services"

	case common.Assets.MicrosoftAzure:
		return "Microsoft Azure"

	case common.Assets.Office365:
		return "Office 365"

	case common.Assets.BetterCloud:
		return "BetterCloud"

	case common.Assets.Zendesk:
		return "Zendesk"

	default:
		return assetType
	}
}
