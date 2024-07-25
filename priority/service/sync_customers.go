package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/priority"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
)

type entityDetails struct {
	customer          *priorityDomain.Customer
	accountReceivable *priorityDomain.AccountReceivable
}

const (
	pageSize = 300
	channel  = "#sales-ops"
)

// Paths of the fields needs to be updated from Priority. !Edit carefuly!
var mergePaths = []firestore.FieldPath{
	[]string{"name"},
	[]string{"_name"},
	[]string{"country"},
	[]string{"currency"},
	[]string{"active"},
	[]string{"contact"},
	[]string{"billingAddress"},
}

func (s *service) SyncCustomers(ctx context.Context) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	bw := fs.BulkWriter(ctx)
	defer bw.End()

	for _, company := range priority.Companies {
		var (
			i          = 0
			startAfter *firestore.DocumentSnapshot
		)

		customers, err := s.priorityReaderWriter.GetCustomers(ctx, company)
		if err != nil {
			l.Errorf("Failed to get customers from Priority for company '%s' with error: %s", company, err)
			return err
		}

		accountsReceivable, err := s.priorityReaderWriter.GetAccountsReceivables(ctx, company)
		if err != nil {
			l.Errorf("Failed to get accounts receivable from Priority for company '%s' with error: %s", company, err)
			return err
		}

		entitiesDetails := make(map[string]*entityDetails)

		for _, customer := range customers.Value {
			entitiesDetails[customer.ID] = &entityDetails{
				customer: customer,
			}
		}

		for _, v := range accountsReceivable.Value {
			if ed, prs := entitiesDetails[v.ID]; prs {
				ed.accountReceivable = v
			}
		}

		query := fs.Collection("entities").
			Where("priorityCompany", "==", company).
			Limit(pageSize).
			SelectPaths([]string{"priorityId"}, []string{"name"}, []string{"customer"})

		for {
			docs, err := query.StartAfter(startAfter).Documents(ctx).GetAll()
			if err != nil {
				l.Errorf("Failed to get entities from Firestore for company '%s' with error: %s", company, err)
				return err
			}

			l.Infof("Company: %s, Page: %d: %d entities", company, i, len(docs))

			if len(docs) > 0 {
				startAfter = docs[len(docs)-1]
				i++
			} else {
				break
			}

			for _, docSnap := range docs {
				var entity common.Entity

				if err := docSnap.DataTo(&entity); err != nil {
					l.Errorf("Failed to unmarshal entity from Firestore with error: %s", err)
					continue
				}

				if entity.PriorityID == "999999" || entity.PriorityID == "US000001" || entity.PriorityID == "UK000001" {
					continue
				}

				if entityDetails, ok := entitiesDetails[entity.PriorityID]; ok {
					if entityDetails.customer == nil || entityDetails.accountReceivable == nil {
						l.Infof("entity %s customer: %+v, accountReceivable: %+v\n",
							entity.PriorityID, entityDetails.customer, entityDetails.accountReceivable)
						continue
					}

					entity.Name = entityDetails.customer.Name
					entity.LowerName = strings.ToLower(entityDetails.customer.Name)
					entity.Country = entityDetails.customer.CountryName
					entity.Active = entityDetails.customer.InactiveFlag == nil
					entity.Currency = entityDetails.accountReceivable.Code
					entity.BillingAddress = common.BillingAddress{
						Address:     entityDetails.customer.Address,
						Address2:    entityDetails.customer.Address2,
						Address3:    entityDetails.customer.Address3,
						State:       entityDetails.customer.State,
						StateA:      entityDetails.customer.Statea,
						StateCode:   entityDetails.customer.StateCode,
						StateName:   entityDetails.customer.StateName,
						CountryName: entityDetails.customer.CountryName,
						Zip:         entityDetails.customer.Zip,
					}

					if entityDetails.customer.Personnel != nil && len(entityDetails.customer.Personnel) > 0 {
						var contact = entityDetails.customer.Personnel[0]
						entity.Contact = &common.EntityContact{
							Name:      contact.Name,
							FirstName: contact.FirstName,
							LastName:  contact.LastName,
							Email:     contact.Email,
							Phone:     contact.Phone,
						}
					} else {
						entity.Contact = nil
					}

					if _, err := bw.Set(docSnap.Ref, entity, firestore.Merge(mergePaths...)); err != nil {
						l.Errorf("Failed to update entity in Firestore for company '%s' with error: %s", company, err)
					}

					if _, err := bw.Set(docSnap.Ref.Collection("entityMetadata").Doc("account-receivables"), map[string]interface{}{
						"balance": entityDetails.accountReceivable.Balance,
						"payTerm": entityDetails.customer.PayTermDesc,
						"entity":  docSnap.Ref,
					}, firestore.MergeAll); err != nil {
						l.Errorf("Failed to update account receivable in Firestore for company '%s' with error: %s", company, err)
					}
				} else {
					l.Infof("Entity %s (%s) was not found in priority", entity.Name, entity.PriorityID)

					message := getSlackMessage(docSnap.Ref.ID, entity, company)

					_, err := common.PublishToSlack(ctx, message, channel)
					if err != nil {
						l.Errorf("Failed to publish to Slack with error: %s", err)
					}
				}
			}
		}

		bw.Flush()
	}

	return nil
}

func getSlackMessage(entityID string, entity common.Entity, company priority.CompanyCode) map[string]interface{} {
	return map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":         time.Now().Unix(),
				"color":      "#F44336",
				"title":      "Billing Profile not found in Priority",
				"title_link": fmt.Sprintf("https://console.doit.com/customers/%s/entities/%s", entity.Customer.ID, entityID),
				"fields": []map[string]interface{}{
					{
						"title": "Billing Profile Name",
						"value": entity.Name,
						"short": false,
					},
					{
						"title": "Priority ID",
						"value": entity.PriorityID,
						"short": true,
					},
					{
						"title": "Company",
						"value": company,
						"short": true,
					},
				},
			},
		},
	}
}
