package dal

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/dataStructures"
	billingConsts "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Metadata struct {
	Logger logger.Provider
	*connection.Connection
}

func NewMetadata(loggerProvider logger.Provider, conn *connection.Connection) *Metadata {
	return &Metadata{
		Logger:     loggerProvider,
		Connection: conn,
	}
}

// Automation Manager Metadata funcs

func (m *Metadata) createAutomationManagerMetadata(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) error {
	logger := m.Logger(ctx)
	_, err := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationManagerCollection).Doc(consts.AutomationManagerDoc).Create(ctx, amm)

	if err != nil && status.Code(err) != codes.AlreadyExists {
		logger.Errorf("unable to create %s. Caused by %s", consts.AutomationManagerDoc, err.Error())
		return err
	}

	return nil
}

func (m *Metadata) GetAutomationManagerMetadata(ctx context.Context) (internalManagerMetadata *dataStructures.AutomationManagerMetadata, err error) {
	var managerMetadata dataStructures.AutomationManagerMetadata

	ref, err := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationManagerCollection).Doc(consts.AutomationManagerDoc).Get(ctx)
	if err != nil {
		//TODO handle error
		return nil, err
	}

	if ref.DataTo(&managerMetadata) != nil {
		//TODO handle error
		return nil, err
	}

	return &managerMetadata, nil
}

func (m *Metadata) DeleteAutomationManagerMetadata(ctx context.Context) (err error) {
	_, err = m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationManagerCollection).Doc(consts.AutomationManagerDoc).Delete(ctx)
	return err
}

func (m *Metadata) SetAutomationManagerMetadata(ctx context.Context, updateFunc func(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) error) (uamm *dataStructures.AutomationManagerMetadata, err error) {
	fs := m.Firestore(ctx)

	err = fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) (err error) {
		var managerMetadata dataStructures.AutomationManagerMetadata

		ref := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationManagerCollection).Doc(consts.AutomationManagerDoc)

		docRef, err := tx.Get(ref)
		if err != nil {
			//TODO handle error
			return err
		}

		if docRef.DataTo(&managerMetadata) != nil {
			//TODO handle error
			return err
		}

		err = updateFunc(ctx, &managerMetadata)
		if err != nil {
			//TODO handle error
			m.Logger(ctx).Error("unable to execute update func. Caused by %s", err)
			return err
		}

		err = tx.Set(ref, managerMetadata)
		if err != nil {
			//TODO handle error
			err = fmt.Errorf("unable to set the document. Caused by %s", err)
			m.Logger(ctx).Error(err)

			return err
		} else {
			uamm = &managerMetadata
		}

		return err
	}, firestore.MaxAttempts(20))
	if err != nil {
		m.Logger(ctx).Errorf("unable to set %s. Caused by %s", consts.AutomationManagerDoc, err.Error())
	}

	return uamm, err
}

// Automation Orchestrator Metadata funcs
func (m *Metadata) CreateAutomationOrchestratorMetadata(ctx context.Context, aom *dataStructures.AutomationOrchestratorMetadata) (uaom *dataStructures.AutomationOrchestratorMetadata, err error) {
	fs := m.Firestore(ctx)

	err = fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) (err error) {
		var orchestratorMetadata dataStructures.AutomationOrchestratorMetadata

		ref := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationManagerCollection).Doc(consts.AutomationOrchestratorDoc)

		docRef, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				err = fmt.Errorf("unable to get orchestrator metadata. Caused by %s", err)
				m.Logger(ctx).Error(err)

				return err
			}

			err = tx.Create(ref, aom)
			if err != nil {
				//TODO handle error
				err = fmt.Errorf("unable to create the document. Caused by %s", err)
				m.Logger(ctx).Error(err)

				return err
			}

			uaom = aom

			return nil
		}

		if docRef.DataTo(&orchestratorMetadata) != nil {
			err = fmt.Errorf("unable to desirialize orchestrator metadata. Caused by %s", err)
			m.Logger(ctx).Error(err)

			return err
		}

		if orchestratorMetadata.WriteTime != nil && time.Until(*orchestratorMetadata.WriteTime) > 0 {
			err = fmt.Errorf("an older orchestrator will run for other %v", time.Until(*orchestratorMetadata.WriteTime))
			m.Logger(ctx).Error(err)

			return err
		}

		aom.Version = orchestratorMetadata.Version + 1

		err = tx.Set(ref, aom)
		if err != nil {
			//TODO handle error
			err = fmt.Errorf("unable to set the document. Caused by %s", err)
			m.Logger(ctx).Error(err)

			return err
		}

		uaom = aom

		return err
	}, firestore.MaxAttempts(20))
	if err != nil {
		m.Logger(ctx).Errorf("unable to set %s. Caused by %s", consts.AutomationOrchestratorDoc, err.Error())
	}

	return uaom, err
}

func (m *Metadata) CreateDefaultAutomationManagerMetadata(ctx context.Context) error {
	return m.createAutomationManagerMetadata(ctx, &dataStructures.AutomationManagerMetadata{})
}

func (m *Metadata) CreateDefaultAutomationOrchestratorMetadata(ctx context.Context) error {
	return m.createAutomationOrchestratorMetadata(ctx, &dataStructures.AutomationOrchestratorMetadata{})
}

func (m *Metadata) createAutomationOrchestratorMetadata(ctx context.Context, aom *dataStructures.AutomationOrchestratorMetadata) error {
	logger := m.Logger(ctx)
	_, err := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationManagerCollection).Doc(consts.AutomationOrchestratorDoc).Create(ctx, aom)

	if err != nil && status.Code(err) != codes.AlreadyExists {
		logger.Errorf("unable to create %s. Caused by %s", consts.AutomationOrchestratorDoc, err.Error())
		return err
	}

	return nil
}

func (m *Metadata) GetAutomationOrchestratorMetadata(ctx context.Context) (uaom *dataStructures.AutomationOrchestratorMetadata, err error) {
	var aom dataStructures.AutomationOrchestratorMetadata

	ref, err := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationManagerCollection).Doc(consts.AutomationOrchestratorDoc).Get(ctx)
	if err != nil {
		//TODO handle error
		return nil, err
	}

	if ref.DataTo(&aom) != nil {
		//TODO handle error
		return nil, err
	}

	return &aom, nil
}

func (m *Metadata) DeleteAutomationOrchestratorMetadata(ctx context.Context) (err error) {
	_, err = m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationManagerCollection).Doc(consts.AutomationOrchestratorDoc).Delete(ctx)
	return err
}

func (m *Metadata) SetAutomationOrchestratorMetadata(ctx context.Context, updateFunc func(ctx context.Context, aom *dataStructures.AutomationOrchestratorMetadata) error) (uaom *dataStructures.AutomationOrchestratorMetadata, err error) {
	fs := m.Firestore(ctx)

	err = fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) (err error) {
		var orchestratorMetadata dataStructures.AutomationOrchestratorMetadata

		ref := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationManagerCollection).Doc(consts.AutomationOrchestratorDoc)

		docRef, err := tx.Get(ref)
		if err != nil {
			//TODO handle error
			return err
		}

		if docRef.DataTo(&orchestratorMetadata) != nil {
			//TODO handle error
			return err
		}

		err = updateFunc(ctx, &orchestratorMetadata)
		if err != nil {
			//TODO handle error
			m.Logger(ctx).Error("unable to execute update func. Caused by %s", err)
			return err
		}

		err = tx.Set(ref, orchestratorMetadata)
		if err != nil {
			//TODO handle error
			err = fmt.Errorf("unable to set the document. Caused by %s", err)
			m.Logger(ctx).Error(err)

			return err
		} else {
			uaom = &orchestratorMetadata
		}

		return err
	}, firestore.MaxAttempts(20))
	if err != nil {
		m.Logger(ctx).Errorf("unable to set %s. Caused by %s", consts.AutomationOrchestratorDoc, err.Error())
	}

	return uaom, err
}

// Automation Tasks Metadata funcs

func (m *Metadata) CreateAutomationTaskMetadata(ctx context.Context, atm *dataStructures.AutomationTaskMetadata) error {
	logger := m.Logger(ctx)
	_, err := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationTasksCollection).Doc(atm.BillingAccountID).Create(ctx, atm)

	if err != nil && status.Code(err) != codes.AlreadyExists {
		logger.Errorf("unable to create %s. Caused by %s", consts.AutomationManagerDoc, err.Error())
		return err
	}

	return nil
}

func (m *Metadata) GetAutomationTaskMetadata(ctx context.Context, billingAccount string) (uatm *dataStructures.AutomationTaskMetadata, err error) {
	var atm dataStructures.AutomationTaskMetadata

	ref, err := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationTasksCollection).Doc(billingAccount).Get(ctx)
	if err != nil {
		//TODO handle error
		return nil, err
	}

	if ref.DataTo(&atm) != nil {
		//TODO handle error
		return nil, err
	}

	return &atm, nil
}

func (m *Metadata) GetDeprecatedAutomationTasksMetadata(ctx context.Context, currVersion int64) (atms []*dataStructures.AutomationTaskMetadata, err error) {
	refIterator := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationTasksCollection).
		Where("version", "!=", currVersion).Documents(ctx)

	if err != nil {
		//TODO handle error
		return nil, err
	}

	for {
		ref, err := refIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		var atm dataStructures.AutomationTaskMetadata

		err = ref.DataTo(&atm)
		if err != nil {
			//TODO handle error
			return nil, err
		}

		atms = append(atms, &atm)
	}

	return atms, nil
}

func (m *Metadata) GetAutomationTasksMetadataByVersion(ctx context.Context, currVersion int64) (atms []*dataStructures.AutomationTaskMetadata, err error) {
	refIterator := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationTasksCollection).
		Where("version", "==", currVersion).Documents(ctx)

	if err != nil {
		//TODO handle error
		return nil, err
	}

	for {
		ref, err := refIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		var atm dataStructures.AutomationTaskMetadata

		err = ref.DataTo(&atm)
		if err != nil {
			//TODO handle error
			return nil, err
		}

		atms = append(atms, &atm)
	}

	return atms, nil
}

func (m *Metadata) GetAutomationTasksMetadataByIteration(ctx context.Context, currIteration int64) (atms []*dataStructures.AutomationTaskMetadata, err error) {
	refIterator := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationTasksCollection).
		Where("iteration", "==", currIteration).Documents(ctx)

	if err != nil {
		//TODO handle error
		return nil, err
	}

	for {
		ref, err := refIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		var atm dataStructures.AutomationTaskMetadata

		err = ref.DataTo(&atm)
		if err != nil {
			//TODO handle error
			return nil, err
		}

		atms = append(atms, &atm)
	}

	return atms, nil
}

func (m *Metadata) GetAutomationTasksMetadata(ctx context.Context) (atms []*dataStructures.AutomationTaskMetadata, err error) {
	refIterator := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationTasksCollection).Documents(ctx)

	if err != nil {
		//TODO handle error
		return nil, err
	}

	for {
		ref, err := refIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		var atm dataStructures.AutomationTaskMetadata

		err = ref.DataTo(&atm)
		if err != nil {
			//TODO handle error
			return nil, err
		}

		atms = append(atms, &atm)
	}

	return atms, nil
}

func (m *Metadata) DeleteAutomationTaskMetadata(ctx context.Context, billingAccount string) (err error) {
	_, err = m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationTasksCollection).Doc(billingAccount).Delete(ctx)
	return err
}

func (m *Metadata) DeleteAllAutomationTasksMetadata(ctx context.Context) (err error) {
	logger := m.Logger(ctx)

	atms, err := m.GetAutomationTasksMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to GetAutomationTasksMetadata. Caused by %s", err)
		logger.Error(err)

		return err
	}

	m.DeleteAutomationTasksMetadata(ctx, atms)

	return nil
}

func (m *Metadata) DeleteAutomationTasksMetadata(ctx context.Context, atms []*dataStructures.AutomationTaskMetadata) {
	logger := m.Logger(ctx)

	for _, atm := range atms {
		go func(atm *dataStructures.AutomationTaskMetadata) {
			err := m.DeleteAutomationTaskMetadata(ctx, atm.BillingAccountID)
			if err != nil {
				err = fmt.Errorf("unable to DeleteAutomationTaskMetadata %s. Caused by %s", atm.BillingAccountID, err)
				logger.Error(err)
			}
		}(atm)
	}
}

func (m *Metadata) DeleteDeprecatedAutomationTasksMetadata(ctx context.Context, currVersion int64) (err error) {
	logger := m.Logger(ctx)

	oldAtms, err := m.GetDeprecatedAutomationTasksMetadata(ctx, currVersion)
	if err != nil {
		err = fmt.Errorf("unable to GetDeprecatedAutomationTasksMetadata. Caused by %s", err)
		logger.Error(err)

		return err
	}

	for _, atm := range oldAtms {
		err = m.DeleteAutomationTaskMetadata(ctx, atm.BillingAccountID)
		if err != nil {
			err = fmt.Errorf("unable to DeleteAutomationTaskMetadata %s. Caused by %s", atm.BillingAccountID, err)
			logger.Error(err)

			return err
		}
	}

	return nil
}

func (m *Metadata) SetAutomationTaskMetadata(ctx context.Context, billingAccount string, updateFunc func(ctx context.Context, atm *dataStructures.AutomationTaskMetadata) error) (uatm *dataStructures.AutomationTaskMetadata, err error) {
	fs := m.Firestore(ctx)

	err = fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) (err error) {
		var taskMetadata dataStructures.AutomationTaskMetadata

		ref := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.AutomationTasksCollection).Doc(billingAccount)

		docRef, err := tx.Get(ref)
		if err != nil {
			//TODO handle error
			return err
		}

		if docRef.DataTo(&taskMetadata) != nil {
			//TODO handle error
			return err
		}

		err = updateFunc(ctx, &taskMetadata)
		if err != nil {
			//TODO handle error
			m.Logger(ctx).Error("unable to execute update func. Caused by %s", err)
			return err
		}

		err = tx.Set(ref, taskMetadata)
		if err != nil {
			//TODO handle error
			err = fmt.Errorf("unable to set the document. Caused by %s", err)
			m.Logger(ctx).Error(err)

			return err
		} else {
			uatm = &taskMetadata
		}

		return err
	}, firestore.MaxAttempts(20))
	if err != nil {
		m.Logger(ctx).Errorf("unable to set %s. Caused by %s", billingAccount, err.Error())
	}

	return uatm, err
}

func (m *Metadata) CreateServiceAccountsMetadata(ctx context.Context, sas []*iam.ServiceAccount) (err error) {
	logger := m.Logger(ctx)

	for _, sa := range sas {
		err = m.CreateServiceAccountMetadata(ctx, sa)
		if err != nil {
			err = fmt.Errorf("unable to create %s. Caused by %s", consts.ServiceAccountPoolCollection, err)
			logger.Error(err)

			return err
		}
	}

	return nil
}

func (m *Metadata) CreateServiceAccountMetadata(ctx context.Context, sa *iam.ServiceAccount) (err error) {
	logger := m.Logger(ctx)
	now := time.Now()

	_, err = m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.ServiceAccountPoolCollection).Doc(sa.Email).Create(ctx, &dataStructures.ServiceAccount{
		ServiceAccountID: sa.Email,
		Name:             sa.Name,
		TTL:              &now,
	})
	if err != nil && status.Code(err) != codes.AlreadyExists {
		err = fmt.Errorf("unable to create %s. Caused by %s", consts.ServiceAccountPoolCollection, err)
		logger.Error(err)

		return err
	}

	return nil
}

func (m *Metadata) GetNextNonFullServiceAccount(ctx context.Context, billingAccount string) (sa *dataStructures.ServiceAccount, err error) {
	logger := m.Logger(ctx)

	var sam dataStructures.ServiceAccount

	fs := m.Firestore(ctx)

	err = fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) (err error) {
		refIterator := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.ServiceAccountPoolCollection).
			Where("isFull", "==", false).Documents(ctx)

		if err != nil {
			//TODO handle error
			return err
		}

		snapshots, err := refIterator.GetAll()
		if err != nil {
			err = fmt.Errorf("unable to GetAll")
			logger.Error(err)

			return err
		}

		if len(snapshots) == 0 {
			return fmt.Errorf("there are no more available service accounts")
		}

		randomRef := snapshots[rand.Intn(len(snapshots))]

		docRef, err := tx.Get(randomRef.Ref)
		if err != nil {
			//TODO handle error
			return err
		}

		if docRef.DataTo(&sam) != nil {
			//TODO handle error
			return err
		}

		if sam.IsFull {
			err = fmt.Errorf("unable to add SA %s to BA %s. Caused by SA seems to be full", sam.ServiceAccountID, billingAccount)
			logger.Error(err)

			return err
		}

		sam.BillingAccounts = append(sam.BillingAccounts, billingAccount)
		if len(sam.BillingAccounts) > 9 {
			sam.IsFull = true
		}

		err = tx.Set(randomRef.Ref, sam)
		if err != nil {
			err = fmt.Errorf("unable to set the document. Caused by %s", err)
			m.Logger(ctx).Error(err)

			return err
		}

		return nil
	}, firestore.MaxAttempts(20))
	if err != nil {
		m.Logger(ctx).Errorf("unable to set %s. Caused by %s", billingAccount, err.Error())
	}

	return &sam, nil
}

func (m *Metadata) GetServiceAccountsMetadata(ctx context.Context) (sas []*dataStructures.ServiceAccount, err error) {
	refIterator := m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.ServiceAccountPoolCollection).Documents(ctx)

	if err != nil {
		//TODO handle error
		return nil, err
	}

	for {
		ref, err := refIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		var atm dataStructures.ServiceAccount

		err = ref.DataTo(&atm)
		if err != nil {
			//TODO handle error
			return nil, err
		}

		sas = append(sas, &atm)
	}

	return sas, nil
}

func (m *Metadata) DeleteServiceAccountMetadata(ctx context.Context, sa string) (err error) {
	_, err = m.Firestore(ctx).Collection(billingConsts.IntegrationsCollection).Doc(billingConsts.GCPFlexsaveStandaloneDoc).Collection(consts.ServiceAccountPoolCollection).Doc(sa).Delete(ctx)
	return err
}

func (m *Metadata) DeleteAllServiceAccountsMetadata(ctx context.Context) (err error) {
	logger := m.Logger(ctx)

	sas, err := m.GetServiceAccountsMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to GetAutomationTasksMetadata. Caused by %s", err)
		logger.Error(err)

		return err
	}

	for _, sa := range sas {
		err = m.DeleteServiceAccountMetadata(ctx, sa.ServiceAccountID)
		if err != nil {
			logger.Error(err)
		}
	}

	return nil
}
