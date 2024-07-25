package api

import (
	"errors"
	"time"
)

// maxRetries is the maximum number of retries before bailing.
var maxRetries = 20

var errMaxRetriesReached = errors.New("exceeded retry limit")

// fnc represents functions that can be retried.
type fnc func(attempt int) (retry bool, err error)

// Do keeps trying the function until the second argument
// returns false, or no error is returned.
func retry(fn fnc, overrideDelta *time.Duration) error {
	var (
		err     error
		cont    bool
		attempt               = 1
		delta   time.Duration = 1
	)

	sleepTime := time.Second * delta
	if overrideDelta != nil {
		sleepTime = time.Second * (*overrideDelta)
	}

	for {
		cont, err = fn(attempt)
		if !cont || err == nil {
			break
		}

		attempt++

		time.Sleep(sleepTime)

		if attempt > maxRetries {
			return errMaxRetriesReached
		}
	}

	return err
}
