//go:generate mockery --output=../mocks --all

package iface

import (
	"context"

	labels "github.com/doitintl/hello/scheduled-tasks/labels/domain"
	"github.com/doitintl/hello/scheduled-tasks/labels/service"
)

type LabelsIface interface {
	CreateLabel(ctx context.Context, req service.CreateLabelRequest) (*labels.Label, error)
	UpdateLabel(ctx context.Context, req service.UpdateLabelRequest) (*labels.Label, error)
	DeleteLabel(ctx context.Context, labelID string) error
	AssignLabels(ctx context.Context, req service.AssignLabelsRequest) error
}
