//go:generate mockery --output=../mocks --all
package iface

import "context"

type Jira interface {
	CreateInstance(ctx context.Context, customerID string, url string) error
}
