package service

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var (
	errInvalidStorageSavingsData = errors.New("invalid storage savings data")
	errScanPriceNotFound         = errors.New("no data received from total scan price per period query")
)

func wrapOperationError(operation, customerID string, err error) error {
	return fmt.Errorf("operation '%s' failed for customer '%s': %w", operation, customerID, err)
}

func wrapPermissionDeniedError(operation, customerID string, err error) error {
	return fmt.Errorf("permission denied error in %s for customer %s: %w", operation, customerID, err)
}

func handleReservationsError[returnType any](log logger.ILogger, operation string, customerID string, err error) []returnType {
	if st, ok := status.FromError(err); ok && st.Code() == codes.PermissionDenied {
		log.Warning(wrapPermissionDeniedError(operation, customerID, err).Error())

		return nil
	}

	log.Errorf(wrapOperationError(operation, customerID, err).Error())

	return nil
}
