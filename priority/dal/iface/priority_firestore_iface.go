//go:generate mockery --output=../mocks --name=PriorityFirestore --filename=priority_firestore_iface.go
package iface

import (
	"context"
)

type PriorityFirestore interface {
	HandleAvalaraStatus(ctx context.Context) (bool, bool, error)
	SetAvalaraHealthyStatus(ctx context.Context, healthy bool) error
}
