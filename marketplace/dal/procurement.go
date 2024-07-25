package dal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/cloudcommerceprocurement/v1"

	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	"github.com/doitintl/http"
	doitPubsubIface "github.com/doitintl/pubsub/iface"
)

type ProcurementDAL struct {
	procurementClient http.IClient
	pubsubHandler     doitPubsubIface.TopicHandler
}

type ApproveAccountPayload struct {
	Properties   map[string]string `json:"properties"`
	ApprovalName ApprovalName      `json:"approvalName"`
	Reason       string            `json:"reason"`
}

type RejectAccountPayload struct {
	ApprovalName ApprovalName `json:"approvalName"`
	Reason       string       `json:"reason"`
}

type RejectEntitlementPayload struct {
	Reason string `json:"reason"`
}

type ApprovalName string

const (
	ApprovalNameSignup ApprovalName = "signup"
)

type EntitlementFilterKey string

const (
	EntitlementFilterKeyAccount                EntitlementFilterKey = "account"
	EntitlementFilterKeyState                  EntitlementFilterKey = "state"
	EntitlementFilterKeyProduct                EntitlementFilterKey = "product"
	EntitlementFilterKeyCustomerBillingAccount EntitlementFilterKey = "customer_billing_account"

	ProcurementEventsTopic = "cmp-events"
)

type Filter struct {
	Key   EntitlementFilterKey
	Value string
}

const (
	basePath                       = "/v1"
	DevProcurementProjectID        = "doitintl-cmp-flexsave-gcp-mp-a"
	ProductionProcurementProjectID = "doitintl-cmp-flexsave-gcp-mp"
)

func GetProcurementProjectID(isProduction bool) string {
	if isProduction {
		return ProductionProcurementProjectID
	}

	return DevProcurementProjectID
}

func NewProcurementDAL(
	procurementClient http.IClient,
	topicHandler doitPubsubIface.TopicHandler,
) (*ProcurementDAL, error) {
	return &ProcurementDAL{
		procurementClient,
		topicHandler,
	}, nil
}

func (s *ProcurementDAL) ApproveAccount(ctx context.Context, accountID, reason string) error {
	payload := ApproveAccountPayload{
		Reason:       reason,
		ApprovalName: ApprovalNameSignup,
	}

	if _, err := s.procurementClient.Post(ctx, &http.Request{
		URL:     fmt.Sprintf("%s/accounts/%s/approve", basePath, accountID),
		Payload: payload,
	}); err != nil {
		return err
	}

	return nil
}

func (s *ProcurementDAL) RejectAccount(ctx context.Context, accountID, reason string) error {
	payload := RejectAccountPayload{
		Reason:       reason,
		ApprovalName: ApprovalNameSignup,
	}

	if _, err := s.procurementClient.Post(ctx, &http.Request{
		URL:     fmt.Sprintf("%s/accounts/%s/reject", basePath, accountID),
		Payload: payload,
	}); err != nil {
		return err
	}

	return nil
}

func (s *ProcurementDAL) GetEntitlement(
	ctx context.Context,
	entitlementID string,
) (*cloudcommerceprocurement.Entitlement, error) {
	var entitlement cloudcommerceprocurement.Entitlement

	if _, err := s.procurementClient.Get(ctx, &http.Request{
		URL:          fmt.Sprintf("%s/entitlements/%s", basePath, entitlementID),
		ResponseType: &entitlement,
	}); err != nil {
		return nil, err
	}

	return &entitlement, nil
}

func (s *ProcurementDAL) ApproveEntitlement(ctx context.Context, entitlementID string) error {
	if _, err := s.procurementClient.Post(ctx, &http.Request{
		URL: fmt.Sprintf("%s/entitlements/%s/approve", basePath, entitlementID),
	}); err != nil {
		return err
	}

	return nil
}

func (s *ProcurementDAL) RejectEntitlement(ctx context.Context, entitlementID, reason string) error {
	payload := RejectEntitlementPayload{
		Reason: reason,
	}

	if _, err := s.procurementClient.Post(ctx, &http.Request{
		URL:     fmt.Sprintf("%s/entitlements/%s/reject", basePath, entitlementID),
		Payload: payload,
	}); err != nil {
		return err
	}

	return nil
}

func (s *ProcurementDAL) ListEntitlements(ctx context.Context, filters ...Filter) ([]*cloudcommerceprocurement.Entitlement, error) {
	var queryStr string

	var keyPairs []string
	for _, filter := range filters {
		keyPairs = append(keyPairs, fmt.Sprintf("%s=%s", filter.Key, filter.Value))
	}

	queryStr = strings.Join(keyPairs, "&")

	url := fmt.Sprintf("%s/entitlements", basePath)

	if len(queryStr) > 0 {
		url = fmt.Sprintf("%s?%s", url, queryStr)
	}

	var entitlements []*cloudcommerceprocurement.Entitlement

	if _, err := s.procurementClient.Get(ctx, &http.Request{
		URL:          url,
		ResponseType: &entitlements,
	}); err != nil {
		return nil, err
	}

	return entitlements, nil
}

func (s *ProcurementDAL) PublishAccountApprovalRequestEvent(
	ctx context.Context,
	payload domain.SubscribePayload,
) error {
	event, err := domain.NewAccountApproveRequestEvent(payload.ProcurementAccountID)
	if err != nil {
		return err
	}

	dataBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	attributes := map[string]string{
		"email":      payload.Email,
		"uid":        payload.UID,
		"customerId": payload.CustomerID,
	}

	message := &pubsub.Message{
		Data:       dataBytes,
		Attributes: attributes,
	}

	if err := s.pubsubHandler.Publish(ctx, message); err != nil {
		return err
	}

	return nil
}
