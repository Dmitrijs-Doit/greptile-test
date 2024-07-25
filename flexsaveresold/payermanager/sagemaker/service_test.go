package sagemakerstate

import (
	"context"
	"errors"
	"testing"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/utils"
	"github.com/stretchr/testify/assert"

	sagemakerMockactions "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/sagemaker/actions/mocks"
)

func Test_defineAction(t *testing.T) {
	type args struct {
		from string
		to   string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "active->pending",
			args: args{
				from: utils.ActiveState,
				to:   utils.PendingState,
			},
			want: utils.ActiveToPending,
		},
		{
			name: "active->disabled",
			args: args{
				from: utils.ActiveState,
				to:   utils.DisabledState,
			},
			want: utils.ActiveToDisabled,
		},
		{
			name: "pending->active",
			args: args{
				from: utils.PendingState,
				to:   utils.ActiveState,
			},
			want: utils.PendingToActive,
		},
		{
			name: "pending->disabled",
			args: args{
				from: utils.PendingState,
				to:   utils.DisabledState,
			},
			want: utils.PendingToDisabled,
		},
		{
			name: "disabled->pending",
			args: args{
				from: utils.DisabledState,
				to:   utils.PendingState,
			},
			want: utils.DisabledToPending,
		},
		{
			name: "disabled->pending",
			args: args{
				from: utils.DisabledState,
				to:   utils.ActiveState,
			},
			want: utils.DisabledToActive,
		},
		{
			name: "disabled->disabled",
			args: args{
				from: utils.DisabledState,
				to:   utils.DisabledState,
			},
			want: utils.StayWithinState,
		},
		{
			name: "invalid trigger misspelled status",
			args: args{
				from: "disable",
				to:   "pended",
			},
			want: utils.InvalidTrigger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defineAction(tt.args.from, tt.args.to); got != tt.want {
				t.Errorf("defineAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_service_ProcessPayerStatusTransition(t *testing.T) {
	var (
		ctx        = context.Background()
		customerID = "xxxx"
		accountID  = "123456"

		invalidTriggerErr = "stateless: No valid leaving transitions are permitted from state 'pended' for trigger 'invalidTrigger', consider ignoring the trigger"
	)

	type fields struct {
		payerManagement     sagemakerMockactions.Service
		OnDisabledToPending bool
		OnActiveToPending   bool
		OnToActive          bool
		OnPendingToDisabled bool
		OnActiveToDisabled  bool
	}

	type args struct {
		initialStatus string
		targetStatus  string
	}

	on := func(f *fields) {
		methods := []struct {
			name string
			fn   func(context.Context, ...any) error
		}{
			{"OnDisabledToPending", func(ctx context.Context, args ...any) error { f.OnDisabledToPending = true; return nil }},
			{"OnActiveToPending", func(ctx context.Context, args ...any) error { f.OnActiveToPending = true; return nil }},
			{"OnToActive", func(ctx context.Context, args ...any) error { f.OnToActive = true; return nil }},
			{"OnPendingToDisabled", func(ctx context.Context, args ...any) error { f.OnPendingToDisabled = true; return nil }},
			{"OnActiveToDisabled", func(ctx context.Context, args ...any) error { f.OnActiveToDisabled = true; return nil }},
		}

		for _, method := range methods {
			f.payerManagement.On(method.name, ctx, accountID, customerID).Return(method.fn)
		}
	}

	tests := []struct {
		name    string
		reset   func(*fields)
		assert  func(*fields)
		args    args
		wantErr error
	}{
		{
			name: "pending->active",
			args: args{
				initialStatus: utils.PendingState,
				targetStatus:  utils.ActiveState,
			},
			assert: func(f *fields) {
				assert.True(t, f.OnToActive)
				assert.False(t, f.OnActiveToPending)
				assert.False(t, f.OnDisabledToPending)
				assert.False(t, f.OnActiveToDisabled)
				assert.False(t, f.OnPendingToDisabled)
			},
			reset: func(f *fields) { f.OnToActive = false },
		},
		{
			name: "pending->disabled",
			args: args{
				initialStatus: utils.PendingState,
				targetStatus:  utils.DisabledState,
			},
			assert: func(f *fields) {
				assert.True(t, f.OnPendingToDisabled)
				assert.False(t, f.OnActiveToPending)
				assert.False(t, f.OnDisabledToPending)
				assert.False(t, f.OnActiveToDisabled)
				assert.False(t, f.OnToActive)
			},
			reset: func(f *fields) { f.OnPendingToDisabled = false },
		},
		{
			name: "disabled->active",
			args: args{
				initialStatus: utils.DisabledState,
				targetStatus:  utils.ActiveState,
			},
			reset: func(f *fields) { f.OnToActive = false },
			assert: func(f *fields) {
				assert.True(t, f.OnToActive)
				assert.False(t, f.OnActiveToPending)
				assert.False(t, f.OnDisabledToPending)
				assert.False(t, f.OnActiveToDisabled)
				assert.False(t, f.OnPendingToDisabled)
			},
		},
		{
			name: "disabled->pending",
			args: args{
				initialStatus: utils.DisabledState,
				targetStatus:  utils.PendingState,
			},
			reset: func(f *fields) { f.OnDisabledToPending = false },
			assert: func(f *fields) {
				assert.True(t, f.OnDisabledToPending)
				assert.False(t, f.OnActiveToPending)
				assert.False(t, f.OnActiveToDisabled)
				assert.False(t, f.OnPendingToDisabled)
				assert.False(t, f.OnToActive)
			},
		},
		{
			name: "active->pending",
			args: args{
				initialStatus: utils.ActiveState,
				targetStatus:  utils.PendingState,
			},
			reset: func(f *fields) { f.OnActiveToPending = false },
			assert: func(f *fields) {
				assert.True(t, f.OnActiveToPending)
				assert.False(t, f.OnDisabledToPending)
				assert.False(t, f.OnActiveToDisabled)
				assert.False(t, f.OnPendingToDisabled)
				assert.False(t, f.OnToActive)
			},
		},
		{
			name: "active->disabled",
			args: args{
				initialStatus: utils.ActiveState,
				targetStatus:  utils.DisabledState,
			},
			reset: func(f *fields) { f.OnActiveToDisabled = false },
			assert: func(f *fields) {
				assert.True(t, f.OnActiveToDisabled)
				assert.False(t, f.OnActiveToPending)
				assert.False(t, f.OnDisabledToPending)
				assert.False(t, f.OnPendingToDisabled)
				assert.False(t, f.OnToActive)
			},
		},
		{
			name: "active->active",
			args: args{
				initialStatus: utils.ActiveState,
				targetStatus:  utils.ActiveState,
			},
			assert: func(f *fields) {
				assert.False(t, f.OnActiveToPending)
				assert.False(t, f.OnDisabledToPending)
				assert.False(t, f.OnActiveToDisabled)
				assert.False(t, f.OnPendingToDisabled)
				assert.False(t, f.OnToActive)
			},
		},
		{
			name: "disabled->disabled",
			args: args{
				initialStatus: utils.DisabledState,
				targetStatus:  utils.DisabledState,
			},
			assert: func(f *fields) {
				assert.False(t, f.OnActiveToPending)
				assert.False(t, f.OnDisabledToPending)
				assert.False(t, f.OnActiveToDisabled)
				assert.False(t, f.OnPendingToDisabled)
				assert.False(t, f.OnToActive)
			},
		},
		{
			name: "pending->pending",
			args: args{
				initialStatus: utils.PendingState,
				targetStatus:  utils.PendingState,
			},
			assert: func(f *fields) {
				assert.False(t, f.OnActiveToPending)
				assert.False(t, f.OnDisabledToPending)
				assert.False(t, f.OnActiveToDisabled)
				assert.False(t, f.OnPendingToDisabled)
				assert.False(t, f.OnToActive)
			},
		},
		{
			name: "invalid trigger malformed state names",
			args: args{
				initialStatus: "pended",
				targetStatus:  "disable",
			},
			assert: func(f *fields) {
				assert.False(t, f.OnActiveToPending)
				assert.False(t, f.OnDisabledToPending)
				assert.False(t, f.OnActiveToDisabled)
				assert.False(t, f.OnPendingToDisabled)
				assert.False(t, f.OnToActive)
			},
			wantErr: errors.New(invalidTriggerErr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			on(&fields)

			if tt.reset != nil {
				tt.reset(&fields)
			}

			s := &service{
				transitionActions: &fields.payerManagement,
			}

			err := s.ProcessPayerStatusTransition(ctx, accountID, customerID, tt.args.initialStatus, tt.args.targetStatus)

			tt.assert(&fields)

			if err != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}
