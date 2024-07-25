package service

import "errors"

var (
	ErrMissingSubscriptionID        = errors.New("missing subscription id")
	ErrMissingCustomerID            = errors.New("missing customer id")
	ErrWrongAssetType               = errors.New("asset type is not office-365")
	ErrWrongQuantity                = errors.New("quantity is wrong")
	ErrInternalServer               = errors.New("internal server error")
	ErrBadRequest                   = errors.New("statusBadRequest")
	ErrNotFound                     = errors.New("statusNotFound")
	ErrUnauthorized                 = errors.New("errUnauthorized")
	ErrForbidden                    = errors.New("statusForbidden")
	ErrDecreasePlanQuantity         = errors.New("cannot decrease commitment plan quantity")
	ErrOperationOnSubFailed         = errors.New("operation on subscription failed")
	ErrQuantity                     = errors.New("quantity is less than 0")
	ErrNoAvailability               = errors.New("there is not availability present")
	ErrSubscriptionPending          = errors.New("subscriptions are still pending")
	ErrSubscriptionHasPrerequisites = errors.New("this subscription is an add-on and requires additional base subscriptions to be purchased first")
)
