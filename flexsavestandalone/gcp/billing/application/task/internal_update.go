package task

import (
	"context"
	"fmt"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type InternalAccountBillingUpdateTask struct {
	loggerProvider logger.Provider
	*connection.Connection
	dataCopier *service.BillingDataCopierService
	metadata   service.Metadata
	tQuery     service.TableQuery
}

func NewInternalAccountBillingUpdateTask(log logger.Provider, conn *connection.Connection) *InternalAccountBillingUpdateTask {
	return &InternalAccountBillingUpdateTask{
		log,
		conn,
		service.NewBillingDataCopierService(log, conn),
		service.NewMetadata(log, conn),
		service.NewTableQuery(log, conn),
	}
}

/*
var setInternalAccountUpdateStateToRunningFn = func(docSnap *firestore.DocumentSnapshot, iteration interface{}) (interface{}, error) {
	var md dataStructures.InternalTaskMetadata
	if err := docSnap.DataTo(&md); err != nil {
		return nil, err
	}
	if md.State != dataStructures.InternalTaskStatePending {
		return &md, common.ErrTaskStateNotPending
	}
	if md.Iteration != iteration.(int64) {
		return &md, common.ErrInvalidIteration
	}
	md.State = dataStructures.InternalTaskStateRunning
	return &md, nil
}*/

func (s *InternalAccountBillingUpdateTask) RunInternalTask(ctx context.Context, body *dataStructures.UpdateRequestBody) (err error) {
	logger := s.loggerProvider(ctx)
	logger.Infof("**** STARTING INTERNAL TASK FOR BA %s #%d ****", body.BillingAccountID, body.Iteration)

	defer func() {
		if err != nil {
			logger.Errorf("unable to execute internal task. Caused by %s", err)
			logger.Errorf("**** DONE INTERNAL TASK FOR BA %s #%d UNSUCCESSFULLY****", body.BillingAccountID, body.Iteration)
		} else {
			logger.Infof("**** DONE INTERNAL TASK FOR BA %s #%d SUCCESSFULLY****", body.BillingAccountID, body.Iteration)
		}
	}()

	itm, err := s.metadata.SetInternalTaskMetadata(ctx, body.BillingAccountID, func(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
		if itm.State != dataStructures.InternalTaskStatePending {
			return common.ErrTaskStateNotPending
		}

		if itm.Iteration != body.Iteration {
			return common.ErrInvalidIteration
		}

		itm.State = dataStructures.InternalTaskStateRunning

		return nil
	})

	if err != nil {
		return err
	}

	if itm.InternalTaskJobs == nil || itm.InternalTaskJobs.FromLocalTableToTmpTable == nil || itm.InternalTaskJobs.FromLocalTableToTmpTable.WaitToStartTimeout == nil || itm.InternalTaskJobs.FromLocalTableToTmpTable.WaitToFinishTimeout == nil {
		err = fmt.Errorf("invalid MD state. MD found InternalTaskJobs %v. Terminating task", itm.InternalTaskJobs)
		logger.Error(err)

		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Until(*itm.InternalTaskJobs.FromLocalTableToTmpTable.WaitToStartTimeout))
	defer cancel()

	logger.Infof("external task ToBucket for BA %s iteration %d started. Remaining time to run %v", itm.BillingAccount, itm.Iteration, time.Until(*itm.InternalTaskJobs.FromLocalTableToTmpTable.WaitToStartTimeout))

	job, err := s.tQuery.CopyFromLocalToTmpTable(ctx, itm)
	if err != nil {
		logger.Errorf("unable to CopyFromLocalToTmpTable for BA %s. Caused by %s", itm.BillingAccount, err)

		_, updateErr := s.metadata.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, oitm *dataStructures.InternalTaskMetadata) error {
			oitm.State = dataStructures.InternalTaskStateFailed
			return nil
		})
		if updateErr != nil {
			logger.Errorf("unable to SetInternalTaskMetadata for BA %s. Caused by %s", itm.BillingAccount, err)
		}

		return err
	}

	logger.Infof("job %s created for BA %s.", job.ID(), itm.BillingAccount)
	itm, err = s.metadata.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, oitm *dataStructures.InternalTaskMetadata) error {
		if itm.Iteration != oitm.Iteration {
			return fmt.Errorf("task running on invalid iterartion %d instead of %d", itm.Iteration, oitm.Iteration)
		}

		oitm.InternalTaskJobs.FromLocalTableToTmpTable.JobID = job.ID()
		oitm.InternalTaskJobs.FromLocalTableToTmpTable.JobStatus = dataStructures.JobCreated

		return nil
	})

	return err
}
