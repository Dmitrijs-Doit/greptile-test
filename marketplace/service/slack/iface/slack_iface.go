//go:generate mockery --output=../mocks --all
package iface

import (
	"context"
)

type ISlackService interface {
	PublishEntitlementCancelledMessage(
		ctx context.Context,
		domain string,
		billingAccountID string,
	) error
}
