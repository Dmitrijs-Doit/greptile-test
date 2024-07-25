package domain

import (
	"context"

	"cloud.google.com/go/firestore"

	metadataMetadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
)

type UpdateMetadataDocsPostTxFunc func() error

type MergeMetadataDocFunc func(
	ctx context.Context,
	tx *firestore.Transaction,
	mdField metadataMetadataDomain.MetadataField,
	customerRef *firestore.DocumentRef,
	key string,
	values []string,
) (string, map[string]interface{}, error)
