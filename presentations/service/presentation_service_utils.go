package service

import (
	"sync"

	"github.com/gin-gonic/gin"
)

type FunctionToRun func(ctx *gin.Context, customerID string) error

func (p *PresentationService) runForEachPresentationCustomerWithAssetType(ctx *gin.Context, assetType string, functionToRun FunctionToRun) []error {
	docSnaps, err := p.customersDAL.GetPresentationCustomersWithAssetType(ctx, assetType)
	if err != nil {
		return []error{err}
	}

	var wg sync.WaitGroup

	errs := make(chan error, len(docSnaps))

	wg.Add(len(docSnaps))

	for _, docSnap := range docSnaps {
		go func(customerID string) {
			if err := functionToRun(ctx, customerID); err != nil {
				errs <- err
			}

			wg.Done()
		}(docSnap.Ref.ID)
	}

	wg.Wait()

	close(errs)

	var errors []error
	for err := range errs {
		errors = append(errors, err)
	}

	return errors
}
