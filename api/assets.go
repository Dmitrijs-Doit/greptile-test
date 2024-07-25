package api

import (
	"context"
	"errors"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type CreateAwsAsset struct {
	// Asset type e.g. amazon-web-services
	// default: "amazon-web-services"
	Type string `json:"type"`
	// Asset mode e.g. New
	// default: "New"
	Mode string `json:"mode"`
	// The desired name of the account
	// default: "Account name"
	AccountName string `json:"accountName"`
	// The root account email
	RootEmail string `json:"rootEmail"`
}

type AssetResponse struct {
	AccountID string `json:"accountID"`
}

// CreateAsset creates a new asset
func (s *APIV1Service) CreateAsset(ctx *gin.Context) {
	l := s.loggerProvider(ctx)
	fs := s.Connection.Firestore(ctx)

	params, err := alignParams(ctx)
	if err != nil {
		l.Errorf("alignParams error: %v", err)
		AbortMsg(ctx, http.StatusBadRequest, err, ErrorBadRequest)

		return
	}

	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	email := ctx.GetString("email")

	entityID, payerAccountID, err := getEntityAndPayerID(ctx, fs, customerID)
	if err != nil {
		AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)
		return
	}

	params.PayerAccountID = payerAccountID

	result, err := s.awsService.CreateAccount(ctx, customerID, entityID, email, &params)
	if err != nil {
		// AWS API Error
		switch err {
		case amazonwebservices.ErrAccountAlreadyExist:
			message := "An account already exists for this root email address"
			AbortMsg(ctx, http.StatusConflict, err, message)
		case amazonwebservices.ErrEmailIsNotValid:
			message := "The provided root email address is not valid"
			AbortMsg(ctx, http.StatusBadRequest, err, message)
		default:
			message := "Create account operation failed, try again later"
			AbortMsg(ctx, http.StatusInternalServerError, err, message)
		}

		return
	}

	l.Info(result)

	var response AssetResponse
	response.AccountID = result
	ctx.JSON(http.StatusInternalServerError, response)
}

func alignParams(ctx *gin.Context) (amazonwebservices.CreateAccountBody, error) {
	modes := map[string]string{
		"NEW":    "new",
		"INVITE": "invite",
	}

	var params CreateAwsAsset

	var awsCAB amazonwebservices.CreateAccountBody
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return awsCAB, err
	}

	if params.RootEmail == "" || params.AccountName == "" || params.Mode == "" || params.Type == "" {
		return awsCAB, errors.New("Some fields are empty")
	}

	if params.Type != common.Assets.AmazonWebServices {
		return awsCAB, errors.New("Type not supported")
	}

	if modes[params.Mode] == "" {
		return awsCAB, errors.New("Mode not supported")
	}

	awsCAB.Email = params.RootEmail
	awsCAB.Name = params.AccountName

	return awsCAB, nil
}

func getEntityAndPayerID(ctx context.Context, fs *firestore.Client, customerID string) (string, string, error) {
	customerRef := fs.Collection("customers").Doc(customerID)

	docSnaps, err := fs.Collection("assets").
		Where("customer", "==", customerRef).
		Where("type", "==", common.Assets.AmazonWebServices).
		Documents(ctx).GetAll()
	if err != nil {
		return "", "", err
	}

	if len(docSnaps) > 0 {
		for _, docSnap := range docSnaps {
			var asset *amazonwebservices.Asset
			if err := docSnap.DataTo(&asset); err != nil {
				continue
			}

			if asset.Entity != nil &&
				asset.Properties.OrganizationInfo != nil &&
				asset.Properties.OrganizationInfo.PayerAccount != nil {
				return asset.Entity.ID, asset.Properties.OrganizationInfo.PayerAccount.AccountID, nil
			}
		}
	} else {
		// currently results in error if there are no AWS assets,
		// until there is a decision on how to handle this case
	}

	return "", "", errors.New("could not select entity and payer account")
}
