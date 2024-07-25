package dal

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (m *Metadata) GetMetadataDocRef(ctx context.Context) *firestore.DocumentRef {
	return m.Firestore(ctx).Collection(consts.IntegrationsCollection).Doc(consts.GCPFlexsaveStandaloneDoc)
}

// Internal Manager Metadata funcs

func (m *Metadata) GetInternalManagerMetadataDocRef(ctx context.Context) *firestore.DocumentRef {
	return m.GetMetadataDocRef(ctx).Collection(consts.InternalUpdateManagerCollection).Doc(consts.InternalUpdateManagerDoc)
}

func (m *Metadata) CreateInternalManagerMetadata(ctx context.Context, internalManagerMetadata *dataStructures.InternalManagerMetadata) error {
	logger := m.Logger(ctx)
	_, err := m.GetInternalManagerMetadataDocRef(ctx).Create(ctx, internalManagerMetadata)

	if err != nil && status.Code(err) != codes.AlreadyExists {
		logger.Errorf("unable to create %s. Caused by %s", consts.InternalUpdateManagerDoc, err.Error())
		return err
	}

	return nil
}

func (m *Metadata) GetInternalManagerMetadata(ctx context.Context) (internalManagerMetadata *dataStructures.InternalManagerMetadata, err error) {
	var managerMetadata dataStructures.InternalManagerMetadata

	ref, err := m.GetInternalManagerMetadataDocRef(ctx).Get(ctx)
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

func (m *Metadata) GetAllInternalTasksMetadata(ctx context.Context) (mdArr []*dataStructures.InternalTaskMetadata, err error) {
	refIterator := m.GetMetadataDocRef(ctx).Collection(consts.InternalTasksCollection).Documents(ctx)

	for {
		ref, err := refIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		var md dataStructures.InternalTaskMetadata

		err = ref.DataTo(&md)
		if err != nil {
			//TODO handle error
			return nil, err
		}

		mdArr = append(mdArr, &md)
	}

	return mdArr, nil
}

func (m *Metadata) DeleteInternalManagerMetadata(ctx context.Context) (err error) {
	_, err = m.GetInternalManagerMetadataDocRef(ctx).Delete(ctx)
	return err
}

func (m *Metadata) DeleteAllInternalTasksMetadata(ctx context.Context) (err error) {
	mdArr, err := m.GetAllInternalTasksMetadata(ctx)
	if err != nil {
		return err
	}

	for _, md := range mdArr {
		err = m.DeleteInternalTaskMetadata(ctx, md.BillingAccount)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Metadata) SetInternalManagerMetadata(ctx context.Context, updateFunc func(ctx context.Context, managerMetadata *dataStructures.InternalManagerMetadata) error) (updatedManagerMetadata *dataStructures.InternalManagerMetadata, err error) {
	fs := m.Firestore(ctx)
	now := time.Now()

	err = fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) (err error) {
		var managerMetadata dataStructures.InternalManagerMetadata

		docRef, err := tx.Get(m.GetInternalManagerMetadataDocRef(ctx))
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

		managerMetadata.LastUpdate = &now

		err = tx.Set(m.GetInternalManagerMetadataDocRef(ctx), managerMetadata)
		if err != nil {
			//TODO handle error
			err = fmt.Errorf("unable to set the document. Caused by %s", err)
			m.Logger(ctx).Error(err)

			return err
		} else {
			updatedManagerMetadata = &managerMetadata
		}

		return err
	}, firestore.MaxAttempts(20))
	if err != nil {
		m.Logger(ctx).Errorf("unable to set %s. Caused by %s", consts.ExternalUpdateManagerDoc, err.Error())
	}

	return updatedManagerMetadata, err
}

// External Manager Metadata funcs

func (m *Metadata) GetExternalManagerDocRef(ctx context.Context) *firestore.DocumentRef {
	return m.GetMetadataDocRef(ctx).Collection(consts.ExternalUpdateManagerCollection).Doc(consts.ExternalUpdateManagerDoc)
}

func (m *Metadata) CreateExternalManagerMetadata(ctx context.Context, externalManagerMetadata *dataStructures.ExternalManagerMetadata) error {
	logger := m.Logger(ctx)
	_, err := m.GetExternalManagerDocRef(ctx).Create(ctx, externalManagerMetadata)

	if err != nil && status.Code(err) != codes.AlreadyExists {
		logger.Errorf("unable to create %s. Caused by %s", consts.ExternalUpdateManagerDoc, err.Error())
		return err
	}

	return nil
}

func (m *Metadata) GetExternalManagerMetadata(ctx context.Context) (*dataStructures.ExternalManagerMetadata, error) {
	var externalManagerMetadata dataStructures.ExternalManagerMetadata

	ref, err := m.GetExternalManagerDocRef(ctx).Get(ctx)
	if err != nil {
		//TODO handle error
		return nil, err
	}

	if ref.DataTo(&externalManagerMetadata) != nil {
		//TODO handle error
		return nil, err
	}

	return &externalManagerMetadata, nil
}

func (m *Metadata) DeleteExternalManagerMetadata(ctx context.Context) (err error) {
	_, err = m.GetExternalManagerDocRef(ctx).Delete(ctx)
	return err
}

func (m *Metadata) SetExternalManagerMetadata(ctx context.Context, updateFunc func(ctx context.Context, originalExternalManagerMetadata *dataStructures.ExternalManagerMetadata) error) (updatedExternalManagerMetadata *dataStructures.ExternalManagerMetadata, err error) {
	fs := m.Firestore(ctx)
	now := time.Now()
	err = fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) (err error) {
		var externalManagerMetadata dataStructures.ExternalManagerMetadata

		docRef, err := tx.Get(m.GetExternalManagerDocRef(ctx))
		if err != nil {
			//TODO handle error
			return err
		}

		if docRef.DataTo(&externalManagerMetadata) != nil {
			//TODO handle error
			return err
		}

		err = updateFunc(ctx, &externalManagerMetadata)
		if err != nil {
			//TODO handle error
			m.Logger(ctx).Error("unable to execute update func. Caused by %s", err.Error())
			return err
		}

		externalManagerMetadata.LastUpdate = &now

		err = tx.Set(m.GetExternalManagerDocRef(ctx), externalManagerMetadata)
		if err != nil {
			m.Logger(ctx).Error("unable to set %s value. Caused by %s", consts.ExternalUpdateManagerCollection, err.Error())
		} else {
			updatedExternalManagerMetadata = &externalManagerMetadata
		}

		return err
	}, firestore.MaxAttempts(20))

	return updatedExternalManagerMetadata, err
}

// Internal Task Metadata funcs

func (m *Metadata) GetInternalTaskCollectionRef(ctx context.Context) *firestore.CollectionRef {
	return m.GetMetadataDocRef(ctx).Collection(consts.InternalTasksCollection)
}

func (m *Metadata) CreateInternalTaskMetadata(ctx context.Context, internalTaskMetadata *dataStructures.InternalTaskMetadata) error {
	logger := m.Logger(ctx)
	_, err := m.GetInternalTaskCollectionRef(ctx).Doc(internalTaskMetadata.BillingAccount).Create(ctx, internalTaskMetadata)

	if err != nil && status.Code(err) != codes.AlreadyExists {
		logger.Errorf("unable to create %s for %s. Caused by %s", consts.InternalTasksCollection, internalTaskMetadata.BillingAccount, err.Error())
		return err
	}

	return nil
}

func (m *Metadata) CreateInternalTasksMetadata(ctx context.Context, internalTasksMetadata []*dataStructures.InternalTaskMetadata) error {
	logger := m.Logger(ctx)

	for _, internalTaskMetadata := range internalTasksMetadata {
		err := m.CreateInternalTaskMetadata(ctx, internalTaskMetadata)
		if err != nil && status.Code(err) != codes.AlreadyExists {
			logger.Errorf("unable to create %s. Caused by %s", consts.InternalTasksCollection, err.Error())
			return err
		}
	}

	return nil
}

func (m *Metadata) GetInternalTasksMetadata(ctx context.Context) (internalTasksMetadata []*dataStructures.InternalTaskMetadata, err error) {
	refIterator := m.GetInternalTaskCollectionRef(ctx).Documents(ctx)

	for {
		ref, err := refIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		var managerMetadata dataStructures.InternalTaskMetadata

		err = ref.DataTo(&managerMetadata)
		if err != nil {
			//TODO handle error
			return nil, err
		}

		internalTasksMetadata = append(internalTasksMetadata, &managerMetadata)
	}

	return internalTasksMetadata, nil
}

func (m *Metadata) GetInternalTasksMetadataByLifeCycleState(ctx context.Context, lifeCycleStage dataStructures.LifeCycleStage) (internalTasksMetadata []*dataStructures.InternalTaskMetadata, err error) {
	refIterator := m.GetInternalTaskCollectionRef(ctx).Where("lifeCycleStage", "==", lifeCycleStage).Documents(ctx)

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

		var internalTaskMetadata dataStructures.InternalTaskMetadata

		err = ref.DataTo(&internalTaskMetadata)
		if err != nil {
			//TODO handle error
			return nil, err
		}

		internalTasksMetadata = append(internalTasksMetadata, &internalTaskMetadata)
	}

	return internalTasksMetadata, nil
}

func (m *Metadata) GetAllInternalTasksMetadataByParams(ctx context.Context, iteration int64, possibleStates []dataStructures.InternalTaskState) (mdArr []*dataStructures.InternalTaskMetadata, err error) {
	refIterator := m.GetInternalTaskCollectionRef(ctx).Where("iteration", "==", iteration).Where("state", "in", possibleStates).Documents(ctx)

	for {
		ref, err := refIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		var md dataStructures.InternalTaskMetadata

		err = ref.DataTo(&md)
		if err != nil {
			//TODO handle error
			return nil, err
		}

		mdArr = append(mdArr, &md)
	}

	return mdArr, nil
}

func (m *Metadata) GetInternalTaskMetadata(ctx context.Context, billingAccount string) (internalTaskMetadata *dataStructures.InternalTaskMetadata, err error) {
	var taskMetadata dataStructures.InternalTaskMetadata

	ref, err := m.GetInternalTaskCollectionRef(ctx).Doc(billingAccount).Get(ctx)
	if err != nil {
		//TODO handle error
		return nil, err
	}

	if ref.DataTo(&taskMetadata) != nil {
		//TODO handle error
		return nil, err
	}

	return &taskMetadata, nil
}

func (m *Metadata) DeleteInternalTasksMetadata(ctx context.Context, internalTasksMetadata []*dataStructures.InternalTaskMetadata) (err error) {
	for _, internalTasksMetadata := range internalTasksMetadata {
		err = m.DeleteInternalTaskMetadata(ctx, internalTasksMetadata.BillingAccount)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Metadata) DeleteInternalTaskMetadata(ctx context.Context, billingAccount string) (err error) {
	logger := m.Logger(ctx)

	_, err = m.GetInternalTaskCollectionRef(ctx).Doc(billingAccount).Delete(ctx)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			logger.Errorf("unable to delete %s. Caused by %s", billingAccount, err.Error())
			return err
		}
	}

	return nil
}
func (m *Metadata) SetInternalTasksMetadata(ctx context.Context, updateFunc func(ctx context.Context, internalTaskMetadata *dataStructures.InternalTaskMetadata) error) error {
	internalTasks, err := m.GetInternalTasksMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to GetInternalTasksMetadata. Caused by %s", err)
		return err
	}

	for _, internalTask := range internalTasks {
		_, err = m.SetInternalTaskMetadata(ctx, internalTask.BillingAccount, updateFunc)
		if err != nil {
			err = fmt.Errorf("unable to SetInternalTaskMetadata. Caused by %s", err)
			return err
		}
	}

	return nil
}

func (m *Metadata) SetInternalTaskMetadata(ctx context.Context, billingAccount string, updateFunc func(ctx context.Context, internalTaskMetadata *dataStructures.InternalTaskMetadata) error) (updatedInternalTaskMetadata *dataStructures.InternalTaskMetadata, err error) {
	fs := m.Firestore(ctx)
	now := time.Now()

	err = fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) (err error) {
		var taskMetadata dataStructures.InternalTaskMetadata

		ref := m.GetInternalTaskCollectionRef(ctx).Doc(billingAccount)

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
			m.Logger(ctx).Error("unable to execute update func. Caused by %s", err.Error())
			return err
		}

		taskMetadata.LastUpdate = &now

		err = tx.Set(ref, taskMetadata)
		if err != nil {
			m.Logger(ctx).Error("unable to set %s value. Caused by %s", consts.InternalTasksCollection, err.Error())
		} else {
			updatedInternalTaskMetadata = &taskMetadata
		}

		return err
	}, firestore.MaxAttempts(20))
	if err != nil {
		m.Logger(ctx).Error(err)
	}

	return updatedInternalTaskMetadata, err
}

// External Task Metadata funcs

func (m *Metadata) CreateExternalTaskMetadata(ctx context.Context, externalTaskMetadata *dataStructures.ExternalTaskMetadata) error {
	logger := m.Logger(ctx)
	_, err := m.GetExternalTaskCollectionRef(ctx).Doc(externalTaskMetadata.BillingAccount).Create(ctx, externalTaskMetadata)

	if err != nil && status.Code(err) != codes.AlreadyExists {
		logger.Errorf("unable to create %s for %s. Caused by %s", consts.InternalTasksCollection, externalTaskMetadata.BillingAccount, err.Error())
		return err
	}

	return nil
}

func (m *Metadata) CreateExternalTasksMetadata(ctx context.Context, externalTasksMetadata []*dataStructures.ExternalTaskMetadata) error {
	logger := m.Logger(ctx)

	for _, externalTaskMetadata := range externalTasksMetadata {
		err := m.CreateExternalTaskMetadata(ctx, externalTaskMetadata)
		if err != nil {
			logger.Errorf("unable to create %s. Caused by %s", consts.InternalTasksCollection, err.Error())
			return err
		}
	}

	return nil
}

// External Task Metadata funcs

func (m *Metadata) GetExternalTaskCollectionRef(ctx context.Context) *firestore.CollectionRef {
	return m.GetMetadataDocRef(ctx).Collection(consts.ExternalTasksCollection)
}

func (m *Metadata) GetExternalTasksMetadata(ctx context.Context) (externalTasksMetadata []*dataStructures.ExternalTaskMetadata, err error) {
	refIterator := m.GetExternalTaskCollectionRef(ctx).Documents(ctx)

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

		var externalTaskMetadata dataStructures.ExternalTaskMetadata

		err = ref.DataTo(&externalTaskMetadata)
		if err != nil {
			//TODO handle error
			return nil, err
		}

		externalTasksMetadata = append(externalTasksMetadata, &externalTaskMetadata)
	}

	return externalTasksMetadata, nil
}

func (m *Metadata) GetExternalTasksMetadataByLifeCycleState(ctx context.Context, lifeCycleStage dataStructures.LifeCycleStage) (externalTasksMetadata []*dataStructures.ExternalTaskMetadata, err error) {
	refIterator := m.GetExternalTaskCollectionRef(ctx).Where("lifeCycleStage", "==", lifeCycleStage).Documents(ctx)

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

		var externalTaskMetadata dataStructures.ExternalTaskMetadata

		err = ref.DataTo(&externalTaskMetadata)
		if err != nil {
			//TODO handle error
			return nil, err
		}

		externalTasksMetadata = append(externalTasksMetadata, &externalTaskMetadata)
	}

	return externalTasksMetadata, nil
}

func (m *Metadata) GetExternalTaskMetadata(ctx context.Context, billingAccount string) (*dataStructures.ExternalTaskMetadata, error) {
	var externalTaskMetadata dataStructures.ExternalTaskMetadata

	ref, err := m.GetExternalTaskCollectionRef(ctx).Doc(billingAccount).Get(ctx)
	if err != nil {
		//TODO handle error
		return nil, err
	}

	if ref.DataTo(&externalTaskMetadata) != nil {
		//TODO handle error
		return nil, err
	}

	return &externalTaskMetadata, nil
}

func (m *Metadata) DeleteExternalTasksMetadata(ctx context.Context, externalTasksMetadata []*dataStructures.ExternalTaskMetadata) (err error) {
	for _, externalTasksMetadata := range externalTasksMetadata {
		err = m.DeleteInternalTaskMetadata(ctx, externalTasksMetadata.BillingAccount)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Metadata) DeleteExternalTaskMetadata(ctx context.Context, billingAccount string) (err error) {
	logger := m.Logger(ctx)

	_, err = m.GetExternalTaskCollectionRef(ctx).Doc(billingAccount).Delete(ctx)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			logger.Errorf("unable to delete %s. Caused by %s", billingAccount, err.Error())
			return err
		}
	}

	return nil
}

func (m *Metadata) DeleteAllExternalTasksMetadata(ctx context.Context) (err error) {
	mdArr, err := m.GetExternalTasksMetadata(ctx)
	if err != nil {
		return err
	}

	for _, md := range mdArr {
		err = m.DeleteExternalTaskMetadata(ctx, md.BillingAccount)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Metadata) SetExternalTaskMetadata(ctx context.Context, billingAccount string, updateFunc func(ctx context.Context, originalExternalTaskMetadata *dataStructures.ExternalTaskMetadata) error) (updatedExternalTaskMetadata *dataStructures.ExternalTaskMetadata, err error) {
	fs := m.Firestore(ctx)
	now := time.Now()

	m.Logger(ctx).Infof("INSIDE SetExternalTaskMetadata for BA %s", billingAccount)

	err = fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		m.Logger(ctx).Infof("starting transaction of SetExternalTaskMetadata for BA %s", billingAccount)

		var externalTaskMetadata dataStructures.ExternalTaskMetadata

		ref := m.GetExternalTaskCollectionRef(ctx).Doc(billingAccount)

		docRef, err := tx.Get(ref)
		if err != nil {
			//TODO handle error
			return err
		}

		if docRef.DataTo(&externalTaskMetadata) != nil {
			//TODO handle error
			return err
		}

		err = updateFunc(ctx, &externalTaskMetadata)
		if err != nil {
			//TODO handle error
			m.Logger(ctx).Error("unable to execute update func. Caused by %s", err.Error())
			return err
		}

		externalTaskMetadata.LastUpdate = &now

		err = tx.Set(ref, externalTaskMetadata)
		if err != nil {
			m.Logger(ctx).Error("unable to set %s value. Caused by %s", consts.ExternalTasksCollection, err.Error())
		} else {
			updatedExternalTaskMetadata = &externalTaskMetadata
		}

		return err
	}, firestore.MaxAttempts(20))
	if err != nil {
		m.Logger(ctx).Errorf("unable to SetExternalTaskMetadata. Caused by %s", err)
	}

	return updatedExternalTaskMetadata, err
}

func (m *Metadata) SetInternalAndExternalTasksMetadata(ctx context.Context, billingAccount string,
	updateFunc func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata,
		oitm *dataStructures.InternalTaskMetadata) error) (*dataStructures.InternalTaskMetadata, *dataStructures.ExternalTaskMetadata, error) {
	fs := m.Firestore(ctx)
	now := time.Now()

	var etm dataStructures.ExternalTaskMetadata

	var itm dataStructures.InternalTaskMetadata

	err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		extRef := m.GetExternalTaskCollectionRef(ctx).Doc(billingAccount)

		extDocRef, err := tx.Get(extRef)
		if err != nil {
			//TODO handle error
			return err
		}

		if extDocRef.DataTo(&etm) != nil {
			//TODO handle error
			return err
		}

		intRef := m.GetInternalTaskCollectionRef(ctx).Doc(billingAccount)

		intDocRef, err := tx.Get(intRef)
		if err != nil {
			//TODO handle error
			return err
		}

		if intDocRef.DataTo(&itm) != nil {
			//TODO handle error
			return err
		}

		err = updateFunc(ctx, &etm, &itm)
		if err != nil {
			//TODO handle error
			m.Logger(ctx).Error("unable to execute update func. Caused by %s", err.Error())
			return err
		}

		etm.LastUpdate = &now

		err = tx.Set(extRef, etm)
		if err != nil {
			m.Logger(ctx).Error("unable to set %s value. Caused by %s", consts.InternalTasksCollection, err.Error())
			return err
		}

		itm.LastUpdate = &now

		err = tx.Set(intRef, itm)
		if err != nil {
			m.Logger(ctx).Error("unable to set %s value. Caused by %s", consts.InternalTasksCollection, err.Error())
			return err
		}

		return nil
	}, firestore.MaxAttempts(20))

	return &itm, &etm, err
}
