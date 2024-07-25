package dal

import "errors"

var (
	ErrLogIsNil                     = errors.New("log is nil")
	ErrPriorityClientIsNil          = errors.New("priority client is nil")
	ErrPriorityProcedureClientIsNil = errors.New("priority procedure client is nil")
	ErrPriorityUserNameIsEmpty      = errors.New("priority username is empty")
	ErrPriorityPasswordIsEmpty      = errors.New("priority password is empty")

	ErrUpdateInvoiceStatusFailed = errors.New("update invoice status failed")
)
