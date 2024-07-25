package domain

import (
	"context"

	"cloud.google.com/go/firestore"
)

type MergeDataHubMetricTypesDocFunc func(
	ctx context.Context,
	tx *firestore.Transaction,
	customerRef *firestore.DocumentRef,
	metricTypes map[string]bool,
) (*firestore.DocumentRef, map[string]interface{}, error)
