package scripts

import (
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type assetsUpdateRecord struct {
	customerRef                  *firestore.DocumentRef
	assetRef                     *firestore.DocumentRef
	contractRef                  *firestore.DocumentRef
	PayerAccountId               string
	contractFlexRiDiscount       float64
	PayerAccountFlexSaveDiscount float64
}

func UpdateFlexSaveCustomerDiscount(ctx *gin.Context) []error {
	logging, err := logger.NewLogging(ctx)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	logger := logging.Logger(ctx)

	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	fs := conn.Firestore(ctx)

	//	batch := fb.NewAutomaticWriteBatch(fs, 420)

	payerAccounts, err := dal.GetMasterPayerAccounts(ctx, fs)
	if err != nil {
		logger.Errorf("can not update flexSave Customer Accounts, failed to fetch masterPayerAccounts, error occurred: %v", err)
		return []error{err}
	}

	contractAssetUpdates := make(map[string][]assetsUpdateRecord)

	assetDocSnaps, err := fs.Collection("assets").
		Where("type", "==", "amazon-web-services").
		Documents(ctx).GetAll()
	if err != nil {
		logger.Errorf("error while fetching: %v", err)
		return []error{err}
	}

	for _, assetDocSnap := range assetDocSnaps {
		var asset amazonwebservices.Asset
		if err := assetDocSnap.DataTo(&asset); err != nil {
			logger.Errorf("skipping asset %v, could not fetch assetDoc, error occurred: %v", assetDocSnap.Ref.ID, err)
			continue
		}

		assetRef := assetDocSnap.Ref
		customerRef := asset.Customer
		contractRef := asset.Contract

		customer, err := common.GetCustomer(ctx, customerRef)
		if err != nil {
			logger.Errorf("skipping asset %v, could not fetch customerRef %v, error occurred: %v", assetRef.ID, customerRef, err)
			continue
		}

		logger.Debugf("customer %v with customerRef %v fetched", customer.Name, customerRef.ID)

		contract, err := common.GetContract(ctx, contractRef) // asset must have valid contract
		if err != nil {
			logger.Errorf("skipping asset %v, could not fetch contractRef %v, error occurred: %v", assetRef.ID, contractRef, err)
			continue
		}

		if contract.Customer.ID != customerRef.ID { // make sure assetCustomer and contractCustomer are same, else log error, & ignore
			logger.Errorf("skipping asset %v, data mismatch, "+
				"contract.customerRef %v & asset.customerRef %v do Not match", assetDocSnap.Ref.ID, contract.Customer.ID, customerRef.ID)
			continue
		}

		contractFlexRi, _ := contract.GetFloatProperty("flexRI", 0.0)

		// asset must have payer account id
		var assetPayerAccountID string
		if asset.Properties != nil && asset.Properties.OrganizationInfo != nil && asset.Properties.OrganizationInfo.PayerAccount != nil {
			assetPayerAccountID = asset.Properties.OrganizationInfo.PayerAccount.AccountID
		} else {
			logger.Warningf("skipping asset %v, no PayerAccountID details found for assetRef...", assetRef.ID)
			continue
		}

		discountFromAssetPayerAccount := payerAccounts.Accounts[assetPayerAccountID].DefaultAwsFlexSaveDiscountRate

		mapKey := contractRef.ID // group various payerAccountIds, by contractID (ie. for given contract, catch multiple assets, having diff mpa
		if _, ok := contractAssetUpdates[mapKey]; !ok {
			contractAssetUpdates[mapKey] = make([]assetsUpdateRecord, 0)
		}

		contractAssetUpdates[mapKey] = append(contractAssetUpdates[mapKey], assetsUpdateRecord{
			customerRef,
			assetRef,
			contractRef,
			assetPayerAccountID,
			contractFlexRi,
			discountFromAssetPayerAccount,
		})
	}

	dvPayeraccountidtoassetsmaps := make([]map[string][]assetsUpdateRecord, 0)

	for contractRefID, contractAssetUpdate := range contractAssetUpdates {
		if len(contractAssetUpdates) == 0 {
			logger.Warningf("no valid asset identified for contractRef %v", contractRefID)
			continue
		}

		payerAccountIDToAssetsMap := make(map[string][]assetsUpdateRecord)

		for _, assetUpdate := range contractAssetUpdate {
			if _, ok := payerAccountIDToAssetsMap[assetUpdate.PayerAccountId]; !ok {
				payerAccountIDToAssetsMap[assetUpdate.PayerAccountId] = make([]assetsUpdateRecord, 0)
			}

			payerAccountIDToAssetsMap[assetUpdate.PayerAccountId] = append(payerAccountIDToAssetsMap[assetUpdate.PayerAccountId], assetUpdate)
			logger.Infof("customer %v, contract %v asset %v payerAccountId %v", assetUpdate.customerRef.ID, assetUpdate.contractRef.ID, assetUpdate.assetRef.ID, assetUpdate.PayerAccountId)
		}

		if len(payerAccountIDToAssetsMap) == 1 {
			for payerAcc, payerAccToAssetsList := range payerAccountIDToAssetsMap {
				if payerAccToAssetsList[0].contractFlexRiDiscount <= payerAccToAssetsList[0].PayerAccountFlexSaveDiscount {
					//batch.Update(payerAccToAssetsList[0].contractRef, []firestore.Update{
					//	{
					//		FieldPath: []string{"properties", "flexRI"},
					//		Value:     firestore.Delete,
					//	},
					//})
					logger.Infof("delete for contract %v payerAcc %v contractFlexRiDiscount %v ", payerAccToAssetsList[0].contractRef.ID, payerAcc, payerAccToAssetsList[0].contractFlexRiDiscount)
				} else {
					contractOverrideValue := payerAccToAssetsList[0].contractFlexRiDiscount // all items in list share same contract
					//batch.Update(payerAccToAssetsList[0].contractRef, []firestore.Update{
					//	{
					//		FieldPath: []string{"properties", "flexRI"},
					//		Value:     firestore.Delete,
					//	},
					//	{
					//		FieldPath: []string{"properties", "awsFlexSaveOverwrite"},
					//		Value:     contractOverrideValue,
					//	},
					//})
					logger.Infof("migrate-update for single, payerAcc %v contract %v payerAccountDiscountArr %v ", payerAccToAssetsList[0].contractRef.ID, payerAcc, contractOverrideValue)
				}
			}
		} else if len(payerAccountIDToAssetsMap) > 1 {
			logger.Infof("** Found Multiple MPA ==> contract %v ", contractRefID)

			payerAccountDiscountArr := make([]float64, 0)
			for payerAccId, payerAccToAssetsList := range payerAccountIDToAssetsMap {
				payerAccountDiscountArr = append(payerAccountDiscountArr, payerAccToAssetsList[0].PayerAccountFlexSaveDiscount)
				logger.Infof("manual migrate-update for multiple MPA per contract, for contract %v "+
					"payerAccId %v (one of %v assets->%v), payerAccountDiscountArr %v ", contractRefID, payerAccId, len(payerAccToAssetsList),
					payerAccToAssetsList[0].assetRef.ID, payerAccountDiscountArr)
			}
		}

		dvPayeraccountidtoassetsmaps = append(dvPayeraccountidtoassetsmaps, payerAccountIDToAssetsMap)
	}

	//if errs := batch.Commit(ctx); errs != nil {
	//	logger.Errorf("could not update discount rates. error %s", errs)
	//	return errs
	//}

	logger.Infof("done")

	return nil
}
