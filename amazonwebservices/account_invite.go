package amazonwebservices

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/organizations"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type InviteAccountBody struct {
	Target         string `json:"target"`
	PayerAccountID string `json:"payerAccountId"`
	Notes          string `json:"notes"`
}

var regexpAccountID = regexp.MustCompile("^\\d{12}$")

func (s *AWSService) InviteAccount(ctx context.Context, customerID, entityID, email string, body *InviteAccountBody) (int, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	doitEmployee := s.doitEmployeeService.IsDoitEmployee(ctx)

	var (
		customer common.Customer
		entity   common.Entity
	)

	customerRef := fs.Collection("customers").Doc(customerID)
	entityRef := fs.Collection("entities").Doc(entityID)
	refs := []*firestore.DocumentRef{customerRef, entityRef}

	docs, err := fs.GetAll(ctx, refs)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if err1, err2 := docs[0].DataTo(&customer), docs[1].DataTo(&entity); err1 != nil {
		return http.StatusInternalServerError, err
	} else if err2 != nil {
		return http.StatusInternalServerError, err
	}

	if customerRef.ID != entity.Customer.ID {
		return http.StatusBadRequest, err
	}

	customer.Snapshot = docs[0]
	entity.Snapshot = docs[1]

	payerAccount, err := isValidPayer(ctx, fs, doitEmployee, customerRef, body.PayerAccountID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if payerAccount == nil {
		return http.StatusBadRequest, errors.New("invalid payer account")
	}

	masterPayerAccount, err := dal.ToMasterPayerAccount(ctx, payerAccount, fs)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	creds, err := masterPayerAccount.NewCredentials("doit-navigator-invite-account")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(endpoints.UsEast1RegionID),
		Credentials: creds,
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	svc := organizations.New(sess)

	// Prepare and validate handshake input
	var (
		input  organizations.InviteAccountToOrganizationInput
		target organizations.HandshakeParty
	)

	target.SetId(body.Target)

	if regexpAccountID.MatchString(body.Target) {
		target.SetType(organizations.HandshakePartyTypeAccount)
	} else {
		target.SetType(organizations.HandshakePartyTypeEmail)
	}

	input.SetTarget(&target)

	if body.Notes != "" {
		input.SetNotes(body.Notes)
	}

	if err := input.Validate(); err != nil {
		return http.StatusInternalServerError, err
	}

	resp, err := svc.InviteAccountToOrganization(&input)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	handshake := resp.Handshake

	parties := make([]*HandshakeParty, len(handshake.Parties))
	for i, p := range handshake.Parties {
		parties[i] = &HandshakeParty{ID: *p.Id, Type: *p.Type}
	}

	handshakeRef := fs.Collection("integrations").Doc("amazon-web-services").Collection("handshakes").Doc(*handshake.Id)
	if _, err := handshakeRef.Set(ctx, &Handshake{
		ID:                  *handshake.Id,
		Target:              body.Target,
		PayerAccount:        payerAccount,
		Arn:                 *handshake.Arn,
		State:               *handshake.State,
		Visible:             true,
		Action:              *handshake.Action,
		ExpirationTimestamp: *handshake.ExpirationTimestamp,
		RequestedTimestamp:  *handshake.RequestedTimestamp,
		Parties:             parties,
		Customer:            customerRef,
		Entity:              entityRef,
		Requester:           email,
	}); err != nil {
		l.Errorf("failed to save aws organizations handshake to firestore with error: %s", err)
	}

	if err := publishAccountInvitedSlackNotification(ctx, handshake, body, email, payerAccount, &customer, &entity); err != nil {
		l.Errorf("failed to publish aws account invite to slack with error: %s", err)
	}

	return http.StatusOK, nil
}

func publishAccountInvitedSlackNotification(
	ctx context.Context,
	handshake *organizations.Handshake,
	body *InviteAccountBody,
	email string,
	payerAccount *domain.PayerAccount,
	customer *common.Customer,
	entity *common.Entity,
) error {
	if customer == nil {
		return errors.New("invalid nil customer")
	}

	fields := []map[string]interface{}{
		{
			"title": "Customer",
			"value": fmt.Sprintf("<https://console.doit.com/customers/%s|%s>", customer.Snapshot.Ref.ID, customer.Name),
			"short": true,
		},
		{
			"title": "Priority ID",
			"value": entity.PriorityID,
			"short": true,
		},
		{
			"title": "Payer Account",
			"value": payerAccount.DisplayName,
			"short": true,
		},
		{
			"title": "State",
			"value": strings.Title(strings.ToLower(*handshake.State)),
			"short": true,
		},
		{
			"title": "Account",
			"value": body.Target,
			"short": true,
		},
		{
			"title": "Notes",
			"value": body.Notes,
			"short": true,
		},
	}

	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":          time.Now().Unix(),
				"color":       "#4CAF50",
				"author_name": fmt.Sprintf("<mailto:%s|%s>", email, email),
				"title":       fmt.Sprintf("Invitation Sent"),
				"title_link":  fmt.Sprintf("https://console.doit.com/customers/%s/assets/amazon-web-services", customer.Snapshot.Ref.ID),
				"thumb_url":   "https://storage.googleapis.com/hello-static-assets/logos/amazon-web-services.png",
				"fields":      fields,
			},
		},
	}
	if _, err := common.PublishToSlack(ctx, message, slackChannel); err != nil {
		return err
	}

	return nil
}

func isValidPayer(
	ctx context.Context,
	fs *firestore.Client,
	doitEmployee bool,
	customerRef *firestore.DocumentRef,
	payerAccountID string,
) (*domain.PayerAccount, error) {
	if doitEmployee {
		return service.GetPayerAccount(ctx, fs, payerAccountID)
	}

	// Search customer's existing AWS assets with specified payerAccountID
	assetDocSnaps, err := fs.Collection("assets").
		Where("type", "==", common.Assets.AmazonWebServices).
		Where("customer", "==", customerRef).
		Where("properties.organization.payerAccount.id", "==", payerAccountID).
		Limit(1).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	if len(assetDocSnaps) != 0 {
		return service.GetPayerAccount(ctx, fs, payerAccountID)
	}

	// Search customer's existing open or accepted AWS invites with specified payerAccountID
	handshakesDocSnaps, err := fs.Collection("integrations/amazon-web-services/handshakes").
		Where("customer", "==", customerRef).
		Where("state", common.In, []string{"ACCEPTED", "OPEN"}).
		Where("payerAccount.id", "==", payerAccountID).
		Limit(1).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	if len(handshakesDocSnaps) != 0 {
		return service.GetPayerAccount(ctx, fs, payerAccountID)
	}

	return nil, fmt.Errorf("invalid payer")
}
