//go:generate mockery --output=../mocks --all
package iface

import (
	"context"
)

type AuthService interface {
	Validate(ctx context.Context, customerID string) (string, error)
}
