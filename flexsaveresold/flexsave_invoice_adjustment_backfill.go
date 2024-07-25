package flexsaveresold

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-multierror"

	"github.com/doitintl/errors"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

const (
	devFlexsaveGlobalTable  = "doitintl-cmp-global-data-dev.aws_custom_billing.aws_custom_billing_export_v1"
	prodFlexsaveGlobalTable = "doitintl-cmp-global-data.aws_custom_billing.aws_custom_billing_export_v1"
)

func (s *Service) UpdateInvoiceAdjustment(ctx *gin.Context, customerID string, month string, totalSavings float64, entities map[string]float64) error {
	d, err := time.Parse(times.YearMonthDayLayout, month)
	if err != nil {
		return err
	}

	monthDate := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC)

	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	if len(customer.Entities) > 1 {
		err := validateEntities(customer, entities, totalSavings)
		if err != nil {
			return errors.Wrap(err, "validateEntities()")
		}

		return s.handleMultipleEntities(ctx, customer, entities, monthDate)
	}

	return s.handleSingleEntity(ctx, customer, monthDate, totalSavings)
}

func (s *Service) CreateBillingRows(ctx *gin.Context, customerID, month string, amount float64) error {
	monthDate, err := time.Parse(times.YearMonthDayLayout, month)
	if err != nil {
		return err
	}

	firstOfMonth := time.Date(monthDate.Year(), monthDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

	err = s.deleteExistingRows(ctx, customerID, firstOfMonth, lastOfMonth)
	if err != nil {
		return err
	}

	return s.CreateBillingLineItems(ctx, amount, lastOfMonth, customerID, true)
}

func (s *Service) deleteExistingRows(ctx *gin.Context, customerID string, firstDay, lastDay time.Time) error {
	queryString := `DELETE
					FROM
					  %s
					WHERE
					  customer = @customerID
					  AND billing_account_id = @customerID
					  AND DATE(usage_date_time) BETWEEN DATE(@firstDay)
					  AND DATE(@lastDay)
					  AND invoice.month = @invoiceMonth
					  AND sku_description = "Flexsave for AWS"`

	formattedQuery := fmt.Sprintf(queryString, s.flexsaveGlobalTable)
	finalQuery := s.bigqueryClient.Query(formattedQuery)

	finalQuery.Labels = map[string]string{
		common.LabelKeyHouse.String():    common.HouseAdoption.String(),
		common.LabelKeyEnv.String():      common.GetEnvironmentLabel(),
		common.LabelKeyFeature.String():  "flexsave",
		common.LabelKeyModule.String():   "billing-backfill",
		common.LabelKeyCustomer.String(): strings.ToLower(customerID),
	}

	finalQuery.Parameters = []bigquery.QueryParameter{
		{
			Name:  "customerID",
			Value: customerID,
		},
		{
			Name:  "firstDay",
			Value: firstDay.Format(times.YearMonthDayLayout),
		},
		{
			Name:  "lastDay",
			Value: lastDay.Format(times.YearMonthDayLayout),
		},
		{
			Name:  "invoiceMonth",
			Value: fmt.Sprintf("%v%v", firstDay.Year(), int(firstDay.Month())),
		},
	}

	job, err := s.queryHandler.Run(ctx, finalQuery)
	if err != nil {
		return errors.Wrapf(err, "Run() customer %s", customerID)
	}

	status, err := s.jobHandler.Wait(ctx, job)
	if err != nil {
		return errors.Wrapf(err, "Wait() customer %s", customerID)
	}

	err = status.Err()
	if err != nil {
		return errors.Wrapf(err, "status.Err() customer %s", customerID)
	}

	return nil
}

func (s *Service) handleSingleEntity(ctx *gin.Context, customer *common.Customer, monthDate time.Time, savings float64) error {
	ref := customer.Snapshot.Ref.Collection("customerInvoiceAdjustments")

	snaps, err := ref.
		Where("description", "==", FlexSaveInvoiceDescription).
		Where("details", "==", FlexSaveInvoiceDetails).
		Where("type", "==", AWS).
		Where("invoiceMonths", "array-contains", monthDate).
		Where("entity", "==", customer.Entities[0]).
		Documents(ctx).GetAll()
	if err != nil {
		return errors.Wrapf(err, "GetAll() for %s", ref.Path)
	}

	if len(snaps) == 0 {
		return s.createAdjustment(ctx, ref, customer, monthDate, savings, customer.Entities[0])
	}

	if len(snaps) > 1 {
		return handleUnexpectedSnapshots(snaps)
	}

	return s.updateAdjustment(ctx, savings, snaps[0])
}

func (s *Service) handleMultipleEntities(ctx *gin.Context, customer *common.Customer, requestEntities map[string]float64, monthDate time.Time) error {
	fs := s.Firestore(ctx)

	var finalErr *multierror.Error

	for id, savings := range requestEntities {
		if savings != 0 {
			entityRef := fs.Doc("entities/" + id)
			ref := customer.Snapshot.Ref.Collection("customerInvoiceAdjustments")
			snaps, err := ref.
				Where("description", "==", FlexSaveInvoiceDescription).
				Where("details", "==", FlexSaveInvoiceDetails).
				Where("type", "==", AWS).
				Where("invoiceMonths", "array-contains", monthDate).
				Where("entity", "==", entityRef).Documents(ctx).GetAll()

			if err != nil {
				finalErr = multierror.Append(finalErr, errors.Wrapf(err, "GetAll() for %s", ref.Path))
				continue
			}

			if len(snaps) == 0 {
				err := s.createAdjustment(ctx, ref, customer, monthDate, savings, entityRef)
				if err != nil {
					finalErr = multierror.Append(finalErr, err)
				}

				continue
			}

			if len(snaps) > 1 {
				finalErr = multierror.Append(finalErr, handleUnexpectedSnapshots(snaps))
				continue
			}

			err = s.updateAdjustment(ctx, savings, snaps[0])
			if err != nil {
				finalErr = multierror.Append(finalErr, err)
			}
		}
	}

	if finalErr != nil {
		return finalErr
	}

	return nil
}

func (s *Service) updateAdjustment(ctx *gin.Context, amount float64, snap *firestore.DocumentSnapshot) error {
	log := s.Logger(ctx)
	now := time.Now()

	_, err := snap.Ref.Update(ctx, []firestore.Update{
		{Path: "amount", Value: amount},
		{Path: "timestamp", Value: &now},
		{Path: "finalized", Value: true},
	})
	if err != nil {
		return errors.Wrapf(err, "Update() for %s", snap.Ref.Path)
	}

	log.Infof("document updated %v", snap.Ref.Path)

	return nil
}

func (s *Service) createAdjustment(ctx *gin.Context, ref *firestore.CollectionRef, customer *common.Customer, monthDate time.Time, amount float64, entity *firestore.DocumentRef) error {
	log := s.Logger(ctx)
	now := time.Now()

	newDocRef, _, err := ref.Add(ctx, &common.InvoiceAdjustment{
		Description:   FlexSaveInvoiceDescription,
		Details:       FlexSaveInvoiceDetails,
		Type:          AWS,
		Customer:      customer.Snapshot.Ref,
		Entity:        entity,
		InvoiceMonths: []time.Time{monthDate},
		Currency:      "USD",
		Amount:        amount,
		Metadata: map[string]interface{}{
			"customer": map[string]interface{}{
				"primaryDomain": customer.PrimaryDomain,
				"name":          customer.Name,
			},
		},
		Timestamp: &now,
		Finalized: true,
	})
	if err != nil {
		return errors.Wrapf(err, "Add() for %s", ref.Path)
	}

	log.Infof("document created: %s", newDocRef.Path)

	return nil
}

func handleUnexpectedSnapshots(snaps []*firestore.DocumentSnapshot) error {
	var paths []string
	for _, s := range snaps {
		paths = append(paths, s.Ref.Path)
	}

	return fmt.Errorf("expected 1 doc, found %v docs with matching entity, type and details: %v", len(snaps), strings.Join(paths, ","))
}

func validateEntities(customer *common.Customer, requestEntities map[string]float64, totalSavings float64) error {
	var customerEntities []string

	var entitiesReceived []string

	var entitySavings float64

	for _, e := range customer.Entities {
		customerEntities = append(customerEntities, e.ID)
	}

	for r := range requestEntities {
		entitiesReceived = append(entitiesReceived, r)
	}

	if requestEntities == nil {
		return fmt.Errorf("customer has %v entities, please provide all in the request body with adjusted amount. Required format: %v", len(customer.Entities), `{"`+strings.Join(customerEntities, `": 0, "`)+`": 0}`)
	}

	if len(customer.Entities) != len(requestEntities) {
		return fmt.Errorf("customer %s expected %v entities in request and received %v, customer entities are: %v", customer.Snapshot.Ref.ID, len(customer.Entities), len(requestEntities), strings.Join(customerEntities, ","))
	}

	if !entitiesMatch(customerEntities, requestEntities) {
		sort.Strings(entitiesReceived)
		return fmt.Errorf("entity values do not match with existing for customer, customer has: %v, entities received in request :%v", strings.Join(customerEntities, ","), strings.Join(entitiesReceived, ","))
	}

	for _, savings := range requestEntities {
		if savings > 0 {
			return fmt.Errorf("only negative savings allowed, request contained positive numbers %+v", requestEntities)
		}

		entitySavings += savings
	}

	if entitySavings != totalSavings {
		return fmt.Errorf("customer %s, total savings (%v) and sum of entities savings received (%v) do not match : %v", customer.Snapshot.Ref.ID, totalSavings, entitySavings, requestEntities)
	}

	return nil
}

func entitiesMatch(s []string, e map[string]float64) bool {
	for _, id := range s {
		_, ok := e[id]
		if !ok {
			return false
		}
	}

	return true
}
