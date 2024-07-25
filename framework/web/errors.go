package web

import (
	"errors"
	"net/http"
)

// Set of error variables for returning on operations.
var (
	ErrBadRequest            = errors.New("bad request")
	ErrNotFound              = errors.New("not found")
	ErrAuthenticationFailure = errors.New("authentication failed")
	ErrForbidden             = errors.New("attempted action is not allowed")
	ErrInternalServerError   = errors.New("internal server error")
	ErrUnauthorized          = errors.New("user does not have required permissions for this action")
)

// ErrorResponse is the form used for API responses from failures in the API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// Error is used to pass an error during the request through the
// application with web specific context.
type Error struct {
	Err    error
	Status int
}

// NewRequestError wraps a provided error with an HTTP status code. This
// function should be used when handlers encounter expected errors.
func NewRequestError(err error, status int) error {
	return &Error{err, status}
}

// Error implements the error interface. It uses the default message of the
// wrapped error. This is what will be shown in the services' logs.
func (err *Error) Error() string {
	return err.Err.Error()
}

// shutdown is a type used to help with the graceful termination of the service.
type shutdown struct {
	Message string
}

// NewShutdownError returns an error that causes the framework to signal
// a graceful shutdown.
func NewShutdownError(message string) error {
	return &shutdown{message}
}

// Error is the implementation of the error interface.
func (s *shutdown) Error() string {
	return s.Message
}

// IsShutdown checks to see if the shutdown error is contained
// in the specified error value.
func IsShutdown(err error) bool {
	if _, ok := err.(*shutdown); ok {
		return true
	}

	return false
}

// TranslateError checks whether the error is defined in our errors set and return the related http status code.
// this function should be used only inside handlers.
func TranslateError(err error) error {
	if err != nil {
		switch err {
		case ErrBadRequest:
			return NewRequestError(err, http.StatusBadRequest)
		case ErrNotFound:
			return NewRequestError(err, http.StatusNotFound)
		case ErrAuthenticationFailure:
			return NewRequestError(err, http.StatusUnauthorized)
		case ErrForbidden:
			return NewRequestError(err, http.StatusForbidden)
		case ErrInternalServerError:
			return NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return nil
}
