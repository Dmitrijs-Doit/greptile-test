package scripts

import (
	"log"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

func AddExpireByToNonFinalInvoices(ctx *gin.Context) []error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	dry := ctx.Query("dry") == "true"
	timeNow := time.Now().UTC()
	ttlDate := time.Now().UTC().AddDate(0, 0, 45)
	ttlField := "expireBy"
	docsProcessed := 0
	batchLimit := 50
	startMonth := ctx.Query("startMonth")

	startMonthParsed, _ := time.Parse("2006-01-02", startMonth)

	log.Printf("AddExpireByToNonFinalInvoices execution started: %v, dry mode %v, starting from %v", timeNow, dry, startMonthParsed)

	defer func() {
		log.Printf("AddExpireByToNonFinalInvoices execution took: %v", time.Since(timeNow))
	}()

	fs, err := firestore.NewClient(ctx, common.ProjectID)

	if err != nil {
		log.Printf("An error has occurred with firestore client: %s", err)
		return []error{err}
	}

	defer fs.Close()

	monthDocSnaps, err := fs.Collection("billing").Doc("invoicing").Collection("invoicingMonths").Documents(ctx).GetAll()

	if err != nil {
		log.Printf("An error has occurred with monthDocSnaps: %s", err)
		return []error{err}
	}

	errCh := make(chan error, len(monthDocSnaps))

	update := []firestore.Update{{
		Path:  ttlField,
		Value: ttlDate,
	}}

	for _, monthDocSnap := range monthDocSnaps {
		month := monthDocSnap.Ref.ID
		invoiceMonth, _ := time.Parse("2006-01", month)

		if invoiceMonth.Before(startMonthParsed) {
			continue
		}

		entityDocSnaps, err := fs.Collection("billing").Doc("invoicing").Collection("invoicingMonths").Doc(month).Collection("monthInvoices").Documents(ctx).GetAll()
		if err != nil {
			log.Printf("An error has occurred with entityDocSnaps: %s", err)
			return []error{err}
		}

		for _, entityDocSnap := range entityDocSnaps {
			wg.Add(1)

			go func(month, entityID string) {
				defer wg.Done()

				var lastDoc *firestore.DocumentSnapshot

				var wb = fb.NewAutomaticWriteBatch(fs, batchLimit)

				for {
					iter := fs.Collection("billing").Doc("invoicing").Collection("invoicingMonths").Doc(month).Collection("monthInvoices").Doc(entityID).Collection("entityInvoices").Where("final", "==", false).Limit(batchLimit).StartAfter(lastDoc).Documents(ctx)
					lastDoc = nil

					if !dry && wb != nil {
						if errs := wb.Commit(ctx); len(errs) > 0 {
							for _, err := range errs {
								log.Printf("Commit error: %v", err)
								errCh <- err
							}
						}
					}

					for {
						doc, err := iter.Next()
						if err == iterator.Done {
							break
						}

						lastDoc = doc

						if err != nil {
							log.Printf("An error has occurred with entityDocSnaps: %s, month %s, entityID %s", err, month, entityID)
							errCh <- err

							return
						}

						docSnap, err := doc.Ref.Get(ctx)
						if err != nil {
							log.Printf("An error has occurred with docSnap: %s", err)
							continue
						}

						if _, err := docSnap.DataAt(ttlField); err != nil {
							wb.Update(doc.Ref, update)

							mu.Lock()
							docsProcessed++
							mu.Unlock()
						}
					}

					if lastDoc == nil {
						break
					}
				}
			}(month, entityDocSnap.Ref.ID)
		}

		wg.Wait()
		log.Printf("Month %s processed, Documents processed %d. errors %d, entity doc snaps %d\n", month, docsProcessed, len(errCh), len(entityDocSnaps))
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	log.Printf("Documents processed %d. errors %d\n", docsProcessed, len(errCh))

	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}
