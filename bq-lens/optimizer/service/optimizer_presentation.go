package service

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	optimizerDal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	firestoremodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	presentationDomain "github.com/doitintl/hello/scheduled-tasks/presentations/domain"
)

type DetailedTable struct {
	MaxAvgDailySlots           float64 `firestore:"maxAvgDailySlots"`
	MaxSlotsUsed               float64 `firestore:"maxSlotsUsed"`
	ObservationEnd             string  `firestore:"observationEnd"`
	ObservationStart           string  `firestore:"observationStart"`
	OptimalSlotsMonthlyCost    float64 `firestore:"optimalSlotsMonthlyCost"`
	OptimalSlotsRecommendation float64 `firestore:"optimalSlotsRecommendation"`
	TotalSpend                 float64 `firestore:"totalSpend"`
}

type DetailedRecommendationDoc struct {
	firestoremodels.RecommendationsDocument
}

func isPresentationCustomer(customer *common.Customer) bool {
	return customer.PresentationMode.IsPredefined
}

func updateExplorerDocsTimeStamps(explorerDoc firestoremodels.ExplorerDocument) error {
	const timeFormat = "2006-01-02"

	length := len(explorerDoc.Day.XAxis)

	for i, timestampString := range explorerDoc.Day.XAxis {
		timestamp, err := time.Parse(timeFormat, timestampString)
		if err != nil {
			continue
		}

		timeDiff := time.Since(timestamp)
		// add the time difference to the timestamp minus 30 days + the index of the timestamp
		explorerDoc.Day.XAxis[i] = timestamp.Add(timeDiff).AddDate(0, 0, -length+i).Format(timeFormat)
	}

	return nil
}

func updateRecommendationsLimitingJobs(recommendationDoc DetailedRecommendationDoc) {
	const timeFormat = time.RFC3339Nano

	if recommendationDoc.LimitingJobs == nil {
		return
	}

	for i := range recommendationDoc.LimitingJobs.DetailedTable {
		detailedTable := &recommendationDoc.LimitingJobs.DetailedTable[i]
		firstExecutionTime, err := time.Parse(timeFormat, detailedTable.FirstExecution)
		if err != nil {
			continue
		}

		lastExecutionTime, err := time.Parse(timeFormat, detailedTable.LastExecution)
		if err != nil {
			continue
		}

		// time difference between the first and last execution time
		timeDiff := lastExecutionTime.Sub(firstExecutionTime)

		// change the last execution time to random time in the last few days
		randomTime := time.Now().AddDate(0, 0, -1).Add(time.Hour * time.Duration(24*7*rand.Intn(3)))
		detailedTable.LastExecution = randomTime.Format(timeFormat)

		// change the first execution time to the last execution time minus the time difference
		detailedTable.FirstExecution = randomTime.Add(-timeDiff).Format(timeFormat)
	}
}

func (p *OptimizerService) CreateSuperQuerySimulationRecommender(ctx context.Context, customer *common.Customer) error {
	fs := p.conn.Firestore(ctx)
	l := p.loggerProvider(ctx)

	l.Infof("Creating superquery simulation recommender for customer: %s", customer.ID)

	if !isPresentationCustomer(customer) {
		return errors.New("customer is not in presentation mode")
	}

	docData, err := p.dalFS.GetRecommendationDetails(ctx, presentationDomain.BudgetaoCustomerID)
	if err != nil {
		return err
	}

	if _, err := p.dalFS.SetRecommendationOutputDoc(ctx, customer.ID, docData); err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	errors := make([]error, 0)

	for _, mode := range optimizerDal.UsageModes {
		for _, period := range bqmodels.DataPeriods {
			wg.Add(1)

			go func(innerMode, innerPeriod string) {
				defer wg.Done()
				// Copy roll-ups
				sourceRollups := p.dalFS.GetRollupDoc(presentationDomain.BudgetaoCustomerID, innerMode, innerPeriod)
				destinationRollups := p.dalFS.GetRecommendationCustomerDoc(customer.ID, innerMode)

				customerDomain, err := customer.Snapshot.DataAt("primaryDomain")
				if err != nil {
					errors = append(errors, err)
					return
				}

				auxData := firestoremodels.RollUpsAuxData{
					PrimaryDomain: customerDomain.(string),
				}
				wb := doitFirestore.NewBatchProviderWithClient(fs, 250).Provide(ctx)

				if err = fb.CopyAllSubCollectionsDocuments(ctx, wb, sourceRollups, destinationRollups, []string{"rollUps", innerPeriod}, updateRollupsDoc, auxData); err != nil {
					errors = append(errors, err)
					return
				}

				if err := wb.Commit(ctx); err != nil {
					errors = append(errors, err)
					return
				}

				// Copy recommendations
				sourceRecommendation, err := p.dalFS.GetRecommendationDoc(presentationDomain.BudgetaoCustomerID, innerMode, innerPeriod).Get(ctx)
				if err != nil {
					errors = append(errors, err)
					return
				}

				if innerMode == string(bqmodels.FlatRate) {
					sourceRecommendationData := sourceRecommendation.Data()

					_, err = p.dalFS.SetRecommendationRecommenderDoc(ctx, customer.ID, innerMode, innerPeriod, sourceRecommendationData)
					if err != nil {
						errors = append(errors, err)
						return
					}
				} else if innerMode == string(bqmodels.OnDemand) {
					var sourceRecommendationData DetailedRecommendationDoc
					if err = sourceRecommendation.DataTo(&sourceRecommendationData); err != nil {
						errors = append(errors, err)
						return
					}

					updateRecommendationsLimitingJobs(sourceRecommendationData)

					_, err = p.dalFS.SetRecommendationRecommenderDoc(ctx, customer.ID, innerMode, innerPeriod, sourceRecommendationData)
					if err != nil {
						errors = append(errors, err)
						return
					}
				} else if innerMode == "output" {
					// No slots explorer data in output mode
					return
				}

				explorerRef := p.dalFS.GetRecommendationExplorerDoc(presentationDomain.BudgetaoCustomerID, innerMode, innerPeriod)
				explorerSnap, err := explorerRef.Get(ctx)
				if err != nil {
					errors = append(errors, err)
					return
				}

				var explorerData firestoremodels.ExplorerDocument
				if err = explorerSnap.DataTo(&explorerData); err != nil {
					errors = append(errors, err)
					return
				}

				if err = updateExplorerDocsTimeStamps(explorerData); err != nil {
					errors = append(errors, err)
					return
				}

				_, err = p.dalFS.SetRecommendationExplorerDoc(ctx, customer.ID, innerMode, innerPeriod, explorerData)
				if err != nil {
					errors = append(errors, err)
					return
				}
			}(mode, string(period))
		}
	}

	wg.Wait()

	if len(errors) > 0 {
		return errors[0]
	}

	l.Infof("Superquery simulation recommender created for customer: %s", customer.ID)

	return nil
}

func (p *OptimizerService) CreateSuperQuerySimulationOptimisation(ctx context.Context, customer *common.Customer) error {
	l := p.loggerProvider(ctx)

	l.Infof("Creating superquery simulation optimisation for customer: %s", customer.ID)

	if !isPresentationCustomer(customer) {
		return errors.New("customer is not in presentation mode")
	}

	docData, err := p.dalFS.GetSimulationDetails(ctx, presentationDomain.BudgetaoCustomerID)
	if err != nil {
		return err
	}

	docData.Customer = customer.Snapshot.Ref

	if _, err = p.dalFS.GetSimulationCustomerDoc(customer.ID).Set(ctx, docData); err != nil {
		return err
	}

	for _, period := range bqmodels.DataPeriods {
		period := string(period)

		docData, err := p.dalFS.GetCostFromTableType(ctx, presentationDomain.BudgetaoCustomerID, period)
		if err != nil {
			return err
		}

		if _, err = p.dalFS.SetSimulationOutputDoc(ctx, customer.ID, period, docData); err != nil {
			return err
		}
	}

	l.Infof("Superquery simulation optimisation created for customer: %s", customer.ID)

	return nil
}

func (p *OptimizerService) CreateSuperQueryBackFillData(ctx context.Context, customer *common.Customer) error {
	l := p.loggerProvider(ctx)

	l.Infof("Creating superquery backfill data for customer: %s", customer.ID)

	if !isPresentationCustomer(customer) {
		return errors.New("customer is not in presentation mode")
	}

	backfill := firestoremodels.BackfillDocument{
		Customer:         customer.Snapshot.Ref,
		BackfillDone:     true,
		BackfillProgress: 100,
	}

	if _, err := p.dalFS.SetJobsinkmetadata(ctx, customer.ID, backfill); err != nil {
		return err
	}

	l.Infof("Superquery backfill data created for customer: %s", customer.ID)

	return nil
}

func customizeUserID(id string, newDomain string) string {
	return strings.Split(id, "@")[0] + "@" + newDomain
}

func updateRollupsDoc(docSnap *firestore.DocumentSnapshot, auxData interface{}) (map[string]interface{}, error) {
	if docSnap.Ref.ID == "storagePrice" || docSnap.Ref.ID == "storageTB" {
		return docSnap.Data(), nil
	}

	aux := auxData.(firestoremodels.RollUpsAuxData)
	updatedData := make(map[string]interface{})

	var docData firestoremodels.RollUpsDocData
	if err := docSnap.DataTo(&docData); err != nil {
		return nil, err
	}

	for keyDoc, data := range docData {
		if data.UserID != "" {
			data.UserID = customizeUserID(data.UserID, aux.PrimaryDomain)
		}

		if data.TopQueries != nil {
			for key, scanData := range data.TopQueries {
				scanData.UserID = customizeUserID(scanData.UserID, aux.PrimaryDomain)
				data.TopQueries[key] = scanData
			}
		}

		if data.TopUsers != nil {
			newUsers := make(map[string]float64)

			for user, n := range data.TopUsers {
				newUserID := customizeUserID(user, aux.PrimaryDomain)
				newUsers[newUserID] = n
			}

			data.TopUsers = newUsers
		}

		updatedData[keyDoc] = data
	}

	return updatedData, nil
}
