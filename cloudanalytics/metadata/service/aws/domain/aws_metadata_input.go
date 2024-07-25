package domain

import (
	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type UpdateAccountMetadataInput struct {
	CustomerRef           *firestore.DocumentRef
	Organizations         []*common.Organization
	CloudhealthCustomerID string
	IsCSP                 bool
	IsStandalone          bool
	IsRecalculated        bool
}
