package service

//type BillingAccount struct {
//	loggerProvider logger.Provider
//	*connection.Connection
//	billingAccountsDAL fsdal.FlexsaveStandaloneAutomation
//	metadataDal        *dal.Metadata
//}

//func NewBillingAccount(log logger.Provider, conn *connection.Connection) *BillingAccount {
//	return &BillingAccount{
//		loggerProvider:     log,
//		Connection:         conn,
//		billingAccountsDAL: fsdal.NewFlexsaveStandaloneAutomationDALWithClient(conn.Firestore(context.Background())),
//		metadataDal:        dal.NewMetadata(log, conn),
//	}
//}

//func (ba *BillingAccount) CreateBillingAccounts(ctx context.Context, aom *dataStructures.AutomationOrchestratorMetadata) error {
//	logger := ba.loggerProvider(ctx)
//	createBillingWG := sync.WaitGroup{}
//	createBillingWG.Add(aom.NumOfDummyUsers)
//	for i := 0; i < aom.NumOfDummyUsers; i++ {
//		go func(index int) {
//			defer createBillingWG.Done()
//			tableName := utils.GetDummyTableName(aom.Version, index)
//			billingAccount := utils.GetDummyBillingAccount(aom.Version, index)
//			atm, err := ba.metadataDal.GetAutomationTaskMetadata(ctx, billingAccount)
//			if err != nil {
//				err = fmt.Errorf("unable to GetAutomationTaskMetadata for BA %s. Caused by %s", billingAccount, err)
//				logger.Error(err)
//				return
//			}
//
//			err = ba.billingAccountsDAL.SetGCPServiceAccountConfig(ctx, billingAccount, &pkg.ServiceAccount{
//				ServiceAccountEmail: atm.ServiceAccount,
//				Billing: &pkg.BillingTableInfo{
//					ProjectID: consts.DummyBQProjectName,
//					DatasetID: consts.DummyBQDatasetName,
//					TableID:   tableName,
//				},
//				BillingAccountID: billingAccount,
//			})
//			if err != nil {
//				err = fmt.Errorf("unable to create billing account %s. Caused by %s", billingAccount, err)
//				logger.Error(err)
//			}
//		}(i)
//	}
//	createBillingWG.Wait()
//	return nil
//}

//func (ba *BillingAccount) DeleteBillingAccounts(ctx context.Context, atms []*dataStructures.AutomationTaskMetadata) error {
//	logger := ba.loggerProvider(ctx)
//	deleteBillingWG := sync.WaitGroup{}
//	deleteBillingWG.Add(len(atms))
//	for _, atm := range atms {
//		go func(billingAccount string) {
//			defer deleteBillingWG.Done()
//			err := ba.billingAccountsDAL.DeleteGCPServiceAccountConfig(ctx, billingAccount)
//			if err != nil {
//				err = fmt.Errorf("unable to DeleteGCPServiceAccountConfig %s. Caused by %s", billingAccount, err)
//				logger.Error(err)
//			}
//		}(atm.BillingAccountID)
//	}
//	deleteBillingWG.Wait()
//	return nil
//}
