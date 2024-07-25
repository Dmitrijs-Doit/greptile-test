package user

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/firebase/tenant"
)

type ImpersonateRequest struct {
	UserID   string `json:"userId"`
	TenantID string `json:"tenantId,omitempty"`
}

type ImpersonateStopResponse struct {
	Token             string `json:"token"`
	RequesterTenantID string `json:"requesterTenantId"`
}

type Impersonate struct {
	Customer  *firestore.DocumentRef `firestore:"customer"`
	StartTime *time.Time             `firestore:"startTime"`
	EndTime   *time.Time             `firestore:"endTime"`
	Subject   ImpersonateParty       `firestore:"subject"`
	Requester ImpersonateParty       `firestore:"requester"`
}

type ImpersonateParty struct {
	Email    string `firestore:"email"`
	UID      string `firestore:"uid"`
	TenantID string `firestore:"tenantId"`
}

const (
	accountAssumptionSettings   = "accountAssumptionSettings"
	accountAssumptionCollection = "accountAssumption"
	customerCollection          = "customers"
	doitOwner                   = "doitOwner"
)

func StartImpersonate(ctx *gin.Context) {
	var r ImpersonateRequest

	if err := ctx.ShouldBindJSON(&r); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	fs := common.GetFirestoreClient(ctx)

	docSnap, err := fs.Collection("users").Doc(r.UserID).Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if !docSnap.Exists() {
		err := errors.New("user not found")
		ctx.AbortWithError(http.StatusNotFound, err)

		return
	}

	var user common.User
	if err := docSnap.DataTo(&user); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	isDoitOwner := ctx.GetBool(doitOwner)
	if !isDoitOwner {
		sufficientPermissionToImpersonate, err := sufficientPermissionsToImpersonate(ctx, fs, user.Customer.Ref.ID)
		if err != nil && status.Code(err) != codes.NotFound {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if !sufficientPermissionToImpersonate {
			err = fmt.Errorf("unable to impersonate user")
			ctx.AbortWithError(http.StatusForbidden, err)

			return
		}
	}

	var tenantAuth *auth.TenantClient

	if r.TenantID == "" {
		customerTenantCustomer := user.Domain
		if user.Customer.Ref != nil {
			customerTenantCustomer = user.Customer.Ref.ID
		}

		tenantAuth2, err := tenant.GetTenantAuthClientByCustomer(ctx, fs, customerTenantCustomer)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		tenantAuth = tenantAuth2
	} else {
		tenantAuth2, err := tenant.GetTenantAuthClientByTenantID(ctx, r.TenantID)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		tenantAuth = tenantAuth2
	}

	subject, err := tenantAuth.GetUserByEmail(ctx, user.Email)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if userID, ok := subject.CustomClaims["userId"].(string); !ok {
		err := errors.New("invalid impersonate subject")
		ctx.AbortWithError(http.StatusBadRequest, err)

		return
	} else if userID != r.UserID {
		err := errors.New("impersonate subject conflict")
		ctx.AbortWithError(http.StatusConflict, err)

		return
	}

	impersonateRef := docSnap.Ref.Collection("impersonation").NewDoc()
	subject.CustomClaims["impersonate"] = map[string]interface{}{
		"active": true,
		"id":     impersonateRef.ID,
		"uid":    ctx.GetString("uid"),
		"email":  ctx.GetString("email"),
	}

	token, err := tenantAuth.CustomTokenWithClaims(ctx, subject.UID, subject.CustomClaims)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	now := time.Now().UTC()

	data := Impersonate{
		Customer:  user.Customer.Ref,
		StartTime: &now,
		EndTime:   nil,
		Subject: ImpersonateParty{
			Email:    subject.Email,
			UID:      subject.UID,
			TenantID: tenantAuth.TenantID(),
		},
		Requester: ImpersonateParty{
			UID:      ctx.GetString("uid"),
			Email:    ctx.GetString("email"),
			TenantID: ctx.GetString("tenantId"),
		},
	}
	if _, err := impersonateRef.Set(ctx, data); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ctx.String(http.StatusOK, token)
}

func StopImpersonate(ctx *gin.Context) {
	claims := ctx.GetStringMap("claims")

	userID, ok := claims["userId"].(string)
	if !ok || userID == "" {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if v, prs := claims["impersonate"]; prs {
		impersonate := v.(map[string]interface{})
		impersonateID := impersonate["id"].(string)
		requesterUID := impersonate["uid"].(string)

		fs := common.GetFirestoreClient(ctx)

		impersonateRef := fs.Collection("users").Doc(userID).Collection("impersonation").Doc(impersonateID)

		docSnap, err := impersonateRef.Get(ctx)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		var i Impersonate
		if err := docSnap.DataTo(&i); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if requesterUID != i.Requester.UID {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}

		tenantAuth, err := tenant.GetTenantAuthClientByTenantID(ctx, i.Requester.TenantID)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		subject, err := tenantAuth.GetUser(ctx, requesterUID)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		token, err := tenantAuth.CustomToken(ctx, subject.UID)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		impersonateRef.Update(ctx, []firestore.Update{
			{FieldPath: []string{"endTime"}, Value: time.Now().UTC()},
		})

		r := ImpersonateStopResponse{
			Token:             token,
			RequesterTenantID: i.Requester.TenantID,
		}
		ctx.JSON(http.StatusOK, r)
	}
}

type AccountAssumptionSettings struct {
	TTL       *time.Time             `firestore:"accountAssumptionUntil"`
	Customer  *firestore.DocumentRef `firestore:"customer"`
	Requester string                 `firestore:"requestedBy"`
}

func checkAssumptionEnabled(ctx *gin.Context, fs *firestore.Client, customerID string) (bool, error) {
	assumptionEnabled := false
	customerDocRef := fs.Collection(customerCollection).Doc(customerID)
	accountAssumptionDocRef := customerDocRef.Collection(accountAssumptionCollection).Doc(accountAssumptionSettings)

	docSnap, err := accountAssumptionDocRef.Get(ctx)
	if err != nil {
		return assumptionEnabled, err
	}

	var assumption AccountAssumptionSettings

	err = docSnap.DataTo(&assumption)
	if err != nil {
		return assumptionEnabled, err
	}

	if assumption.TTL == nil {
		assumptionEnabled = true
	} else {
		assumptionEnabled = assumption.TTL.After(time.Now())
	}

	return assumptionEnabled, err
}

func sufficientPermissionsToImpersonate(ctx *gin.Context, fs *firestore.Client, customerID string) (bool, error) {
	assumptionEnabled := false

	var err error

	assumptionEnabled, err = checkAssumptionEnabled(ctx, fs, customerID)
	if err != nil {
		return assumptionEnabled, err
	}

	allowed := assumptionEnabled

	return allowed, err
}
