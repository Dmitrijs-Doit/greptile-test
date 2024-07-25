package dal

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/billing-explainer/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type FirestoreDAL struct {
	client *firestore.Client
	logger logger.Logger
}

func NewFirestoreDAL(ctx context.Context, projectID string) *FirestoreDAL {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	return &FirestoreDAL{client: client}
}

func readDocument(ctx context.Context, docRef *firestore.DocumentRef) (map[string]interface{}, error) {
	if docRef == nil {
		return nil, nil
	}

	docSnapshot, err := docRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	return docSnapshot.Data(), nil
}

func (d *FirestoreDAL) UpdateEntityFirestoreDoc(ctx context.Context, isBackfill bool, yearMonth string, entityID string, invoicingMode string, summaryBqResults []domain.SummaryBQ, bucketName string, serviceBreakdownResults []domain.ServiceRecord, accountBreakdownResults []domain.AccountRecord) error {
	explainer := domain.MapResultsToExplainer(summaryBqResults, serviceBreakdownResults, accountBreakdownResults)

	collectionPath := "billing/invoicing/invoicingMonths/" + yearMonth + "/monthInvoices/" + entityID
	docRef := d.client.Doc(collectionPath)

	docSnap, err := docRef.Get(ctx)
	if err != nil {
		return err
	}

	// Filter the entityInvoice collection group by retrieving the timestamp of its last update
	// and using that timestamp to filter only documents updated since then.
	// This ensures that we only process the most recent updates to the documents in the collection.

	var invoiceLastProcessedTime time.Time

	timestamp, exists := docSnap.Data()["timestamp"]

	if exists {
		invoiceLastProcessedTime, _ = timestamp.(time.Time)
	}

	startTime := time.Date(invoiceLastProcessedTime.Year(), invoiceLastProcessedTime.Month(), invoiceLastProcessedTime.Day(), 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(24 * time.Hour)

	now := time.Now().UTC()

	// Decide if we are backfilling data
	firstDayOfInvoiceMonth, _ := time.Parse("2006-01", yearMonth) // e.g. 2024-03 gets parsed to 2024-03-01 00:00:00 +0000 UTC
	isBackfill = isBackfill || !(firstDayOfInvoiceMonth.Year() == now.Year() && firstDayOfInvoiceMonth.Month() == now.Month() && firstDayOfInvoiceMonth.Day() == now.Day())

	if isBackfill {
		// Look for relevant documents since the beginning of the month
		startTime = firstDayOfInvoiceMonth
		endTime = now
	}

	iter := d.client.Collection(collectionPath+"/entityInvoices").Where("timestamp", ">=", startTime).Where("timestamp", "<", endTime).OrderBy("timestamp", firestore.Desc).Documents(ctx)

	var docID string

	for {
		doc, err := iter.Next()

		if err == iterator.Done {
			break
		}

		if err != nil {
			break
		}

		data := doc.Data()

		rows, ok := data["rows"].([]interface{})

		if data["type"] != common.Assets.AmazonWebServices {
			continue
		}

		if !ok {
			continue
		}

		if invoicingMode == "CUSTOM" && bucketName != "" && bucketName != "bucket" {
			for _, row := range rows {
				rowMap, ok := row.(map[string]interface{})
				if !ok {
					continue // Skip if the row is not a map
				}
				// Case: invoicing bucket present
				if description, exists := rowMap["description"].(string); exists && description == "Invoice Bucket" && rowMap["details"] == bucketName {
					if data["issuedAt"] != nil {
						// If there is an invoice that has been issued, we should take that document since it is the latest
						docID = doc.Ref.ID
						break
					}
					// In case of backfill, we need to take the document with `issuedAt` not null even if it is not the latest doc
					if docID == "" && (isBackfill && data["issuedAt"] != nil || !isBackfill) {
						docID = doc.Ref.ID
					}
				}
			}
		} else {
			// Case: no invoicing bucket
			if data["issuedAt"] != nil {
				// If there is an invoice that has been issued, we should take that document since it is the latest
				docID = doc.Ref.ID
				break
			}
			// In case of backfill, we need to take the document with `issuedAt` not null even if it is not the latest doc
			if docID == "" && (isBackfill && data["issuedAt"] != nil || !isBackfill) {
				docID = doc.Ref.ID
			}
		}
	}

	if docID == "" {
		d.logger.Warning("no doc found to save billing explainer data")
		return err
	}

	err = d.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docRef := d.client.Doc(collectionPath + "/entityInvoices/" + docID)
		_, err := docRef.Get(ctx)
		if err != nil {
			return err
		}
		return tx.Set(docRef, map[string]interface{}{
			"explainer": explainer,
		}, firestore.MergeAll)
	})

	if err != nil {
		d.logger.Warning(fmt.Sprintf("failed to update entity document: %v", err))
		return err
	}

	return nil
}

func (d *FirestoreDAL) GetPayerAccountDoc(ctx context.Context, payerID string) (map[string]interface{}, error) {
	var m map[string]interface{}

	iter := d.client.Collection("app/master-payer-accounts/mpaAccounts").Where("accountNumber", "==", payerID).Documents(ctx)

	doc, err := iter.Next()

	if err == iterator.Done {
		return nil, nil
	}

	if err != nil {
		return m, err
	}

	data, err := readDocument(ctx, doc.Ref)
	if err != nil {
		d.logger.Warning(fmt.Sprintf("error reading document: %v", err))
		return m, err
	}

	return data, err
}
