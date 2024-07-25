package iface

import (
	"context"
)

type Service interface {
	UpdateKnownIssues(ctx context.Context) error
	UpdateAwsKnownIssues(ctx context.Context) error
	UpdateGcpKnownIssues(ctx context.Context) error
}
