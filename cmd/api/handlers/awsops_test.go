package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager"
	state "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/compute/mocks"
	mockpayermanager "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/mocks"
	mockrdsstate "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/rds/mocks"
	mocksagemakerstate "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/sagemaker/mocks"
	payermanagerutils "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

func Test_handler_ProcessComputeFormEntry(t *testing.T) {
	var (
		payerID    = "1233455"
		customerID = "xxjijefiji"

		statusPending  = "pending"
		statusDisabled = "disabled"
		managedType    = "auto"
		reason         = "just because"

		body                       = `{"managed":"auto","status":"pending","rdsStatus":"pending","sagemakerStatus":"pending"}`
		bodyMissingField           = `{"managed":"auto","status":"","rdsStatus":"pending","sagemakerStatus":"pending"}`
		bodyWithStatusChangeReason = `{"managed":"auto","status":"pending","statusChangeReason":"just because","rdsStatus":"pending","sagemakerStatus":"pending"}`

		someErr = errors.New("something went wrong")
	)

	type fields struct {
		payerManager         mockpayermanager.Service
		payerStateController state.Service
		rdsState             mockrdsstate.Service
		sagemakerState       mocksagemakerstate.Service
	}

	type args struct {
		body io.Reader
	}

	tests := []struct {
		name           string
		on             func(f *fields, ctx *gin.Context)
		args           args
		assert         func(t *testing.T, ctx *gin.Context)
		wantStatusCode int
	}{
		{
			name: "happy path",
			on: func(f *fields, ctx *gin.Context) {
				config := types.PayerConfig{
					CustomerID:      customerID,
					AccountID:       payerID,
					Status:          statusPending,
					Managed:         managedType,
					RDSStatus:       statusDisabled,
					SageMakerStatus: statusDisabled,
				}

				form := payermanager.FormEntry{
					Status:          statusPending,
					Managed:         managedType,
					RDSStatus:       &statusPending,
					SagemakerStatus: &statusPending,
				}

				f.payerManager.On("GetPayer", ctx, payerID).Return(config, nil)

				f.payerManager.On("UpdateNonStatusPayerConfigFields", ctx, config, form).Return(nil)

				f.payerStateController.On("ProcessPayerStatusTransition", ctx, payerID, customerID, statusPending, statusPending).Return(nil)

				f.rdsState.On("ProcessPayerStatusTransition", ctx, payerID, customerID, statusDisabled, statusPending).Return(nil)

				f.sagemakerState.On("ProcessPayerStatusTransition", ctx, payerID, customerID, statusDisabled, statusPending).Return(nil)
			},
			args: args{
				body: strings.NewReader(body),
			},
			wantStatusCode: http.StatusOK,
			assert: func(t *testing.T, ctx *gin.Context) {
				assert.Equal(t, nil, ctx.Value(utils.StatusChangeReasonContextKey))
			},
		},
		{
			name: "with status change reason",
			on: func(f *fields, ctx *gin.Context) {
				config := types.PayerConfig{
					CustomerID:      customerID,
					AccountID:       payerID,
					Status:          statusPending,
					Managed:         managedType,
					RDSStatus:       statusDisabled,
					SageMakerStatus: statusDisabled,
				}

				form := payermanager.FormEntry{
					Status:             statusPending,
					Managed:            managedType,
					StatusChangeReason: &reason,
					RDSStatus:          &statusPending,
					SagemakerStatus:    &statusPending,
				}

				f.payerManager.On("GetPayer", ctx, payerID).Return(config, nil)

				f.payerManager.On("UpdateNonStatusPayerConfigFields", ctx, config, form).Return(nil)

				f.payerStateController.On("ProcessPayerStatusTransition", ctx, payerID, customerID, statusPending, statusPending).Return(nil)

				f.rdsState.On("ProcessPayerStatusTransition", ctx, payerID, customerID, statusDisabled, statusPending).Return(nil)

				f.sagemakerState.On("ProcessPayerStatusTransition", ctx, payerID, customerID, statusDisabled, statusPending).Return(nil)
			},
			args: args{
				body: strings.NewReader(bodyWithStatusChangeReason),
			},
			wantStatusCode: http.StatusOK,
			assert: func(t *testing.T, ctx *gin.Context) {
				assert.Equal(t, reason, ctx.Value(utils.StatusChangeReasonContextKey))
			},
		},
		{
			name: "bad request missing required field",
			on: func(f *fields, ctx *gin.Context) {
			},
			args: args{
				body: strings.NewReader(bodyMissingField),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "failed to get payer config",
			on: func(f *fields, ctx *gin.Context) {
				config := types.PayerConfig{
					CustomerID: customerID,
					AccountID:  payerID,
					Status:     statusPending,
					Managed:    managedType,
				}

				f.payerManager.On("GetPayer", ctx, payerID).Return(config, someErr)
			},
			args: args{
				body: strings.NewReader(body),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "failed to update payer config",
			on: func(f *fields, ctx *gin.Context) {
				config := types.PayerConfig{
					CustomerID: customerID,
					AccountID:  payerID,
					Status:     statusPending,
					Managed:    managedType,
				}

				form := payermanager.FormEntry{
					Status:          statusPending,
					Managed:         managedType,
					RDSStatus:       &statusPending,
					SagemakerStatus: &statusPending,
				}

				f.payerManager.On("GetPayer", ctx, payerID).Return(config, nil)

				f.payerManager.On("UpdateNonStatusPayerConfigFields", ctx, config, form).Return(someErr)
			},
			args: args{
				body: strings.NewReader(body),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "failed during compute status transition",
			on: func(f *fields, ctx *gin.Context) {
				config := types.PayerConfig{
					CustomerID: customerID,
					AccountID:  payerID,
					Status:     statusPending,
					Managed:    managedType,
				}

				form := payermanager.FormEntry{
					Status:          statusPending,
					Managed:         managedType,
					RDSStatus:       &statusPending,
					SagemakerStatus: &statusPending,
				}

				f.payerManager.On("GetPayer", ctx, payerID).Return(config, nil)

				f.payerManager.On("UpdateNonStatusPayerConfigFields", ctx, config, form).Return(nil)

				f.payerStateController.On("ProcessPayerStatusTransition",
					ctx,
					payerID,
					customerID,
					statusPending,
					statusPending,
				).Return(someErr)
			},
			args: args{
				body: strings.NewReader(body),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "failed on rds status transition",
			on: func(f *fields, ctx *gin.Context) {
				config := types.PayerConfig{
					CustomerID:      customerID,
					AccountID:       payerID,
					Status:          statusPending,
					Managed:         managedType,
					RDSStatus:       statusPending,
					SageMakerStatus: statusPending,
				}

				form := payermanager.FormEntry{
					Status:          statusPending,
					Managed:         managedType,
					RDSStatus:       &statusPending,
					SagemakerStatus: &statusPending,
				}

				f.payerManager.On("GetPayer", ctx, payerID).Return(config, nil)

				f.payerManager.On("UpdateNonStatusPayerConfigFields", ctx, config, form).Return(nil)

				f.payerStateController.On("ProcessPayerStatusTransition", ctx, payerID, customerID, statusPending, statusPending).Return(nil)

				f.rdsState.On("ProcessPayerStatusTransition", ctx, payerID, customerID, statusPending, statusPending).Return(someErr)
			},
			args: args{
				body: strings.NewReader(body),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "failed on sagemaker status transition",
			on: func(f *fields, ctx *gin.Context) {
				config := types.PayerConfig{
					CustomerID:      customerID,
					AccountID:       payerID,
					Status:          statusPending,
					Managed:         managedType,
					RDSStatus:       statusPending,
					SageMakerStatus: statusPending,
				}

				form := payermanager.FormEntry{
					Status:          statusPending,
					Managed:         managedType,
					RDSStatus:       &statusPending,
					SagemakerStatus: &statusPending,
				}

				f.payerManager.On("GetPayer", ctx, payerID).Return(config, nil)

				f.payerManager.On("UpdateNonStatusPayerConfigFields", ctx, config, form).Return(nil)

				f.payerStateController.On("ProcessPayerStatusTransition", ctx, payerID, customerID, statusPending, statusPending).Return(nil)

				f.rdsState.On("ProcessPayerStatusTransition", ctx, payerID, customerID, statusPending, statusPending).Return(nil)

				f.sagemakerState.On("ProcessPayerStatusTransition", ctx, payerID, customerID, statusPending, statusPending).Return(someErr)
			},
			args: args{
				body: strings.NewReader(body),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)

			req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("/payers/%s/ops-update", payerID), tt.args.body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			ctx.Request = req
			ctx.Params = []gin.Param{{Key: "payerId", Value: payerID}}

			fields := fields{}
			if tt.on != nil {
				tt.on(&fields, ctx)
			}

			h := &handler{
				payerManager:          &fields.payerManager,
				computeStateService:   &fields.payerStateController,
				rdsStateService:       &fields.rdsState,
				sagemakerStateService: &fields.sagemakerState,
			}

			err = h.ProcessOpsUpdates(ctx)
			if err == nil {
				assert.Equal(t, tt.wantStatusCode, w.Code)
			} else {
				var reqErr *web.Error
				if errors.As(err, &reqErr) {
					assert.Equal(t, tt.wantStatusCode, reqErr.Status)
				} else {
					t.Fatalf("Unexpected error type: %v", err)
				}
			}

			if tt.assert != nil {
				tt.assert(t, ctx)
			}
		})
	}
}

func Test_validateTransitions(t *testing.T) {
	type args struct {
		compute   string
		rds       string
		sagemaker string
	}

	tests := []struct {
		name      string
		args      args
		wantError bool
	}{
		{
			"in compute active all can be active",
			args{
				compute:   payermanagerutils.ActiveState,
				rds:       payermanagerutils.ActiveState,
				sagemaker: payermanagerutils.ActiveState,
			},
			false,
		},

		{
			"in compute active, pending allowed",
			args{
				compute:   payermanagerutils.ActiveState,
				rds:       payermanagerutils.PendingState,
				sagemaker: payermanagerutils.PendingState,
			},
			false,
		},

		{
			"in compute active, disabled are allowed",
			args{
				compute:   payermanagerutils.ActiveState,
				rds:       payermanagerutils.DisabledState,
				sagemaker: payermanagerutils.DisabledState,
			},
			false,
		},

		{
			"in compute pending others cannot be active",
			args{
				compute:   payermanagerutils.PendingState,
				rds:       payermanagerutils.ActiveState,
				sagemaker: payermanagerutils.DisabledState,
			},
			true,
		},
		{
			"in compute disabled, others cant be active",
			args{
				compute:   payermanagerutils.DisabledState,
				rds:       payermanagerutils.ActiveState,
				sagemaker: payermanagerutils.PendingState,
			},
			true,
		},

		{
			"in compute pending others can be also pending",
			args{
				compute:   payermanagerutils.PendingState,
				rds:       payermanagerutils.PendingState,
				sagemaker: payermanagerutils.PendingState,
			},
			false,
		},

		{
			"in compute disabled others can be also disabled",
			args{
				compute:   payermanagerutils.DisabledState,
				rds:       payermanagerutils.DisabledState,
				sagemaker: payermanagerutils.DisabledState,
			},
			false,
		},

		{
			"if others are disabled, compute can be pending",
			args{
				compute:   payermanagerutils.PendingState,
				rds:       payermanagerutils.DisabledState,
				sagemaker: payermanagerutils.DisabledState,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTransitions(tt.args.compute, tt.args.rds, tt.args.sagemaker)

			if err == nil {
				assert.False(t, tt.wantError)
			} else {
				assert.True(t, tt.wantError)
			}
		})
	}
}
