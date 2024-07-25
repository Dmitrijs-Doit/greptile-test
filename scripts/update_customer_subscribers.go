package scripts

import (
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type SubscribersUpdaterRequest struct {
	IsWrite *bool
}

type SubscribersUpdater struct {
	ctx     *gin.Context
	fs      *firestore.Client
	l       logger.ILogger
	isWrite bool
	wb      *fb.AutomaticWriteBatch
}

func newSubscribersUpdater(ctx *gin.Context, r *SubscribersUpdaterRequest, l logger.ILogger) (*SubscribersUpdater, error) {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	isWrite := false
	if r.IsWrite != nil {
		isWrite = *r.IsWrite
	}

	wb := fb.NewAutomaticWriteBatch(fs, 500)

	s := &SubscribersUpdater{
		ctx:     ctx,
		fs:      fs,
		l:       l,
		isWrite: isWrite,
		wb:      wb,
	}

	return s, nil
}

func (s *SubscribersUpdater) setCustomersUpdateBatchOperations() error {
	docSnaps, err := s.fs.Collection("customers").Documents(s.ctx).GetAll()
	if err != nil {
		return err
	}

	for _, docSnap := range docSnaps {
		var c common.Customer
		if err := docSnap.DataTo(&c); err != nil {
			return err
		}

		if len(c.Subscribers) > 0 {
			u := s.getCustomerUpdateOperation(&c)
			if u != nil {
				s.wb.Update(docSnap.Ref, u)
			}
		}
	}

	return nil
}

func (s *SubscribersUpdater) getCustomerUpdateOperation(customer *common.Customer) []firestore.Update {
	newSubscribers := make([]string, 0)
	shouldUpdateSubscribers := false

	for _, subscriber := range customer.Subscribers {
		newSubscribers = append(newSubscribers, subscriber)

		if strings.Contains(subscriber, "@doit-intl.com") {
			sArray := strings.Split(subscriber, "@")
			newSubscriber := fmt.Sprintf("%s@doit.com", sArray[0])
			shouldAdd := true

			for _, s := range customer.Subscribers {
				if s == newSubscriber {
					shouldAdd = false
					break
				}
			}

			if shouldAdd {
				shouldUpdateSubscribers = true

				newSubscribers = append(newSubscribers, newSubscriber)
			}
		}
	}

	if shouldUpdateSubscribers {
		s.l.Infof("Updateing subscribers list for customer: %s|%s. New list: %s", customer.Name, customer.PrimaryDomain, newSubscribers)

		return []firestore.Update{{
			Path:  "subscribers",
			Value: newSubscribers,
		}}
	}

	return nil
}

func UpdateCustomerSubscribers(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	l.Infof("UpdateCustomerSubscribers on project: %s", common.ProjectID)

	var r SubscribersUpdaterRequest
	if err := ctx.ShouldBindJSON(&r); err != nil {
		return []error{err}
	}

	s, err := newSubscribersUpdater(ctx, &r, l)
	if err != nil {
		return []error{err}
	}

	if err := s.setCustomersUpdateBatchOperations(); err != nil {
		return []error{err}
	}

	if s.isWrite {
		return s.wb.Commit(ctx)
	}

	return nil
}
