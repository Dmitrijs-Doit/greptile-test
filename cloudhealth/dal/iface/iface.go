package iface

import (
	"context"

	"cloud.google.com/go/firestore"
)

//go:generate mockery --name CloudHealthDAL --output ../mocks
type CloudHealthDAL interface {
	GetCustomerCloudHealthID(ctx context.Context, customerRef *firestore.DocumentRef) (string, error)
}
