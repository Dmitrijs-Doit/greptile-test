package amazonwebservices

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/concedefy"
	"github.com/doitintl/googleadmin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/gin-gonic/gin"
)

type AccessGroupEmail string

const (
	awsssoAdminsGroup            AccessGroupEmail = "awssso-admins@doit-intl.com"
	awsssoStrategicGroup         AccessGroupEmail = "awssso-strategic@doit-intl.com"
	awsssoBillingAndSupportGroup AccessGroupEmail = "awssso-cre@doit-intl.com"
)

var ErrUnauthorized = fmt.Errorf("Not authorized")
var ErrBadRequest = fmt.Errorf("Bad request")

type UserClaims struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	DoitEmployee bool   `json:"doitEmployee"`
	CustomerID   string `json:"customerId"`
}

type AWSAccountAccessor struct {
	fs          *firestore.Client
	googleAdmin googleadmin.GoogleAdmin
	transporter *concedefy.Transporter
}

func NewAWSAccountAccessor(ctx context.Context, fs *firestore.Client, googleAdmin googleadmin.GoogleAdmin) (*AWSAccountAccessor, error) {
	var tc concedefy.TransporterConfig
	if common.Production {
		tc = concedefy.ProdConfig
	} else {
		tc = concedefy.DevConfig
	}

	transporter, err := concedefy.NewTransporter(ctx, tc)
	if err != nil {
		return nil, err
	}

	return &AWSAccountAccessor{
		fs:          fs,
		googleAdmin: googleAdmin,
		transporter: transporter,
	}, nil
}

type GetCredsRequest struct {
	Flow         concedefy.Flow `json:"flow"`
	CustomerID   string         `json:"customerId"`
	AWSAccountID string         `json:"awsAccountId"`
}

func (a *AWSAccountAccessor) GetCreds(ctx *gin.Context, req GetCredsRequest) (*concedefy.Credentials, error) {
	u, err := UserClaimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := a.Authorize(ctx, *u, req); err != nil {
		return nil, err
	}

	creds, err := a.transporter.Send(ctx, concedefy.Session{
		Flow:             req.Flow,
		CSPType:          concedefy.CSPTypeAWS,
		CSPEnvironmentID: req.AWSAccountID,
		Email:            u.Email,
	})

	if err != nil {
		if errors.Is(err, concedefy.ErrUnauthorized) {
			return nil, fmt.Errorf("%w: Transporter unauthorized", ErrUnauthorized)
		}

		if errors.Is(err, concedefy.ErrBadRequest) {
			return nil, ErrBadRequest
		}

		return nil, err
	}

	return creds, nil
}

func (a *AWSAccountAccessor) Authorize(ctx context.Context, u UserClaims, req GetCredsRequest) error {
	switch req.Flow {
	case concedefy.FlowAWSMPAAdministrator:
		return a.AuthAdminOrSupport(ctx, u, req.CustomerID, awsssoAdminsGroup)
	case concedefy.FlowAWSMPABillingAndSupport:
		return a.AuthAdminOrSupport(ctx, u, req.CustomerID, awsssoBillingAndSupportGroup)
	case concedefy.FlowAWSMPAStrategic:
		return a.AuthStrategic(ctx, u, req.CustomerID)
	default:
		return fmt.Errorf("unknown flow: %s", req.Flow)
	}
}

type AllowedRoleResponse struct {
	Allowed          bool   `json:"allowed"`
	NotAllowedReason string `json:"notAllowedReason,omitempty"`
}

func (a *AWSAccountAccessor) GetRoles(ctx *gin.Context, customerID string) (map[concedefy.Flow]AllowedRoleResponse, error) {
	u, err := UserClaimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var res = make(map[concedefy.Flow]AllowedRoleResponse)

	adminErr := a.AuthAdminOrSupport(ctx, *u, customerID, awsssoAdminsGroup)
	if adminErr == nil {
		res[concedefy.FlowAWSMPAAdministrator] = AllowedRoleResponse{Allowed: true}
	} else if errors.Is(adminErr, ErrUnauthorized) {
		res[concedefy.FlowAWSMPAAdministrator] = AllowedRoleResponse{Allowed: false, NotAllowedReason: adminErr.Error()}
	} else {
		return nil, adminErr
	}

	billingAndSupportErr := a.AuthAdminOrSupport(ctx, *u, customerID, awsssoBillingAndSupportGroup)
	if billingAndSupportErr == nil {
		res[concedefy.FlowAWSMPABillingAndSupport] = AllowedRoleResponse{Allowed: true}
	} else if errors.Is(billingAndSupportErr, ErrUnauthorized) {
		res[concedefy.FlowAWSMPABillingAndSupport] = AllowedRoleResponse{Allowed: false, NotAllowedReason: billingAndSupportErr.Error()}
	} else {
		return nil, billingAndSupportErr
	}

	stategicErr := a.AuthStrategic(ctx, *u, customerID)
	if stategicErr == nil {
		res[concedefy.FlowAWSMPAStrategic] = AllowedRoleResponse{Allowed: true}
	} else if errors.Is(stategicErr, ErrUnauthorized) {
		res[concedefy.FlowAWSMPAStrategic] = AllowedRoleResponse{Allowed: false, NotAllowedReason: stategicErr.Error()}
	} else {
		return nil, stategicErr
	}

	return res, nil
}

func (a *AWSAccountAccessor) AuthAdminOrSupport(ctx context.Context, u UserClaims, accountCustomerID string, checkedGroup AccessGroupEmail) error {
	if u.DoitEmployee {
		isMember, err := a.isGoogleGroupMember(u.Email, string(checkedGroup))
		if err != nil {
			return err
		}

		if !isMember {
			if checkedGroup == awsssoBillingAndSupportGroup {
				return fmt.Errorf("%w: user must be a member of the billing and support group", ErrUnauthorized)
			}

			return fmt.Errorf("%w: user must be a member of the admins group", ErrUnauthorized)
		}

		return nil
	}

	if u.CustomerID != "" {
		if u.CustomerID != accountCustomerID {
			return fmt.Errorf("%w: account does not belong to the requesting customer", ErrUnauthorized)
		}

		hasRole, err := UserHasRoles(ctx, a.fs, u.ID, []common.PresetRole{common.PresetRoleAdmin})
		if err != nil {
			return err
		}

		if !hasRole {
			return fmt.Errorf("%w: user does not have the required roles", ErrUnauthorized)
		}

		return nil
	}

	return ErrUnauthorized
}

func (a *AWSAccountAccessor) AuthStrategic(ctx context.Context, u UserClaims, accountCustomerID string) error {
	if !u.DoitEmployee {
		return fmt.Errorf("%w: user is not a doit employee", ErrUnauthorized)
	}

	// user is the an account manager for the customer
	if accountCustomerID != "" {
		customer, err := common.GetCustomer(ctx, a.fs.Collection("customers").Doc(accountCustomerID))
		if err != nil {
			return err
		}

		AMs, err := common.GetCustomerAccountManagers(ctx, customer, common.AccountManagerCompanyDoit)
		if err != nil {
			return err
		}

		for _, AM := range AMs {
			if AM.Email == u.Email {
				return nil
			}
		}

		// fallback to google group membership
		isMember, err := a.isGoogleGroupMember(u.Email, string(awsssoStrategicGroup))
		if err != nil {
			return err
		}

		if isMember {
			return nil
		}

		return fmt.Errorf("%w: user must be an AM or a member of the strategic google group", ErrUnauthorized)
	}

	return ErrUnauthorized
}

func (a *AWSAccountAccessor) isGoogleGroupMember(email string, group string) (bool, error) {
	members, err := a.googleAdmin.ListGroupMembers(group)
	if err != nil {
		return false, err
	}

	for _, member := range members.Members {
		if member.Email == email {
			return true, nil
		}
	}

	return false, nil
}

func UserHasRoles(ctx context.Context, fs *firestore.Client, userID string, roles []common.PresetRole) (bool, error) {
	userDoc, err := fs.Collection("users").Doc(userID).Get(ctx)
	if err != nil {
		return false, err
	}

	user := &common.User{}
	if err := userDoc.DataTo(user); err != nil {
		return false, err
	}

	rolesIDsMap := make(map[string]bool)

	for _, role := range roles {
		roleRef := fs.Collection("roles").Doc(string(role))
		rolesIDsMap[roleRef.ID] = true
	}

	for _, r := range user.Roles {
		if rolesIDsMap[r.ID] {
			return true, nil
		}
	}

	return false, nil
}

// UserClaimsFromCtx extracts the user claims from the context as set by the auth middleware
func UserClaimsFromCtx(ctx *gin.Context) (*UserClaims, error) {
	claims := ctx.GetStringMap("claims")

	customerID, ok := claims["customerId"].(string)
	if !ok {
		customerID = ""
	}

	return &UserClaims{
		ID:           claims["userId"].(string),
		Email:        claims["email"].(string),
		DoitEmployee: ctx.GetBool("doitEmployee"),
		CustomerID:   customerID,
	}, nil
}
