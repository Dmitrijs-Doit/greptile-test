package iface

import "context"

//go:generate mockery --name OnboardService --output ../mocks --case=underscore
type OnboardService interface {
	HandleSpecificSink(ctx context.Context, sinkID string) error
	RemoveData(ctx context.Context, sinkID string) error
}
