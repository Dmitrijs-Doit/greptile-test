package dispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/zapier/dal"
	"github.com/doitintl/hello/scheduled-tasks/zapier/domain"
)

var (
	errZapGone           = errors.New("zap gone")
	errRateLimitExceeded = errors.New("rate limit exceeded")
)

//go:generate mockery --name Dispatcher --output=./mocks
type Dispatcher interface {
	Dispatch(ctx context.Context, data any, customer *firestore.DocumentRef, itemID string, event domain.EventType) error
}

type EventDispatcher struct {
	dal dal.WebhookSubscriptionDAL
	c   *http.Client
	l   logger.Provider
}

func NewEventDispatcher(logger logger.Provider, fun connection.FirestoreFromContextFun) *EventDispatcher {
	return &EventDispatcher{
		dal: dal.NewWebhookSubscriptionsFirestoreWithClient(logger, fun),
		c:   &http.Client{Timeout: 10 * time.Second},
		l:   logger,
	}
}

// Dispatch dispatches an event, checking whether there is a matching webhook subscription, then sends the data to the
// matching target URLs
func (e *EventDispatcher) Dispatch(
	ctx context.Context,
	data any,
	customer *firestore.DocumentRef,
	entityID string,
	event domain.EventType,
) error {
	// Get all matching subscriptions
	subs, err := e.dal.GetForDispatch(ctx, customer, entityID, event)
	if err != nil {
		return err
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	sem := make(chan struct{}, 50) // limit goroutines used to dispatch with semaphore

	for _, sub := range subs {
		sem <- struct{}{}

		go func(s *domain.WebhookSubscription) {
			defer func() { <-sem }()

			if err := e.dispatchToSubscription(ctx, body, s); err != nil {
				e.l(ctx).Warningf("error when dispatching subscription %s: %s", s.ID, err)
			}
		}(sub)
	}

	return nil
}

// dispatchToSubscription handles the errors that are returned from sendToTarget
func (e *EventDispatcher) dispatchToSubscription(ctx context.Context, data []byte, sub *domain.WebhookSubscription) error {
	if err := e.sendToTarget(ctx, data, sub.TargetURL); err != nil {
		switch err {
		case errZapGone:
			err := e.dal.Delete(ctx, sub.ID)
			if err != nil {
				return fmt.Errorf("there was an error deleting subscription")
			}
		case errRateLimitExceeded:
			return fmt.Errorf("rate limit exceeded for subscription")
		default:
			return err
		}
	}

	return nil
}

// sendToTarget attempts to send the data to the targetURL and returns errors depending on the responses status code
func (e *EventDispatcher) sendToTarget(ctx context.Context, data []byte, targetURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		return nil
	case resp.StatusCode == http.StatusGone: // Zap removed from zapier
		return errZapGone
	case resp.StatusCode == http.StatusTooManyRequests:
		return errRateLimitExceeded
	default:
		return fmt.Errorf("webhook request failed with status code %d", resp.StatusCode)
	}
}
