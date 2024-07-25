package scripts

import (
	"fmt"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type setCustomClaimsRequest struct {
	Email   string                 `json:"email"`
	Claims  map[string]interface{} `json:"claims"`
	IsWrite *bool                  `json:"isWrite"`
}

type CustomClaimsSetter struct {
	ctx        *gin.Context
	fs         *firestore.Client
	fbAuth     *auth.Client
	tenantAuth *auth.TenantClient
	l          logger.ILogger
	isWrite    bool
	email      string
	claims     map[string]interface{}
}

func newCustomClaimsSetter(ctx *gin.Context, l logger.ILogger, r setCustomClaimsRequest) (*CustomClaimsSetter, error) {
	fbApp, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: common.ProjectID,
	})

	if err != nil {
		return nil, err
	}

	fbAuth, err := fbApp.Auth(ctx)
	if err != nil {
		return nil, err
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	isWrite := false
	if r.IsWrite != nil {
		isWrite = *r.IsWrite
	}

	c := &CustomClaimsSetter{
		ctx:     ctx,
		fs:      fs,
		fbAuth:  fbAuth,
		l:       l,
		isWrite: isWrite,
		email:   r.Email,
		claims:  r.Claims,
	}

	return c, nil
}

func (c *CustomClaimsSetter) setTenantAuthClient() error {
	collectionPath := fmt.Sprintf("tenants/%s/emailToTenant", common.ProjectID)

	emailToTenantDocSnap, err := c.fs.Collection(collectionPath).Doc(c.email).Get(c.ctx)
	if err != nil {
		return err
	}

	if !emailToTenantDocSnap.Exists() {
		return fmt.Errorf("%s was not found in email to tenant mapping", c.email)
	}

	tenantId, err := emailToTenantDocSnap.DataAt("tenantId")
	if err != nil {
		return err
	}

	tenantIdVal, ok := tenantId.(string)
	if !ok {
		return fmt.Errorf("cloudn't parse tenantId for email %s", c.email)
	}

	c.l.Infof("tenantId for email %s is %s", c.email, tenantIdVal)

	tenantAuth, err := c.fbAuth.TenantManager.AuthForTenant(tenantIdVal)
	if err != nil {
		return err
	}

	c.tenantAuth = tenantAuth

	return nil
}

func (c *CustomClaimsSetter) setCustomClaims() (map[string]interface{}, error) {
	user, err := c.tenantAuth.GetUserByEmail(c.ctx, c.email)
	if err != nil {
		return nil, err
	}

	c.l.Infof("previous claims %+v", user.CustomClaims)

	newClaims := make(map[string]interface{})
	for k, v := range user.CustomClaims {
		newClaims[k] = v
	}

	for k, v := range c.claims {
		newClaims[k] = v
	}

	c.l.Infof("new claims %+v", newClaims)

	userToUpdate := auth.UserToUpdate{}
	userToUpdate.CustomClaims(newClaims)

	if c.isWrite {
		c.l.Infof("writing custom claims for user %s", c.email)

		updatedUser, err := c.tenantAuth.UpdateUser(c.ctx, user.UID, &userToUpdate)
		if err != nil {
			return nil, err
		}

		return updatedUser.CustomClaims, nil
	}

	return user.CustomClaims, nil
}

// SetCustomClaims
// Set a user token customClaims properties.
// example payload
//
//	{
//	    "email": "e2e@doit-intl.com",
//	    "claims": {
//	        "doitDeveloper": true,
//	        "doitEmployee": true,
//	        "doitOwner": true,
//	        "domain": "doit-intl.com",
//	        "provider": "password",
//	        "tenantId": "t-EE8CtpzYiKp0dVAESV-cictw"
//	    },
//	    "isWrite": false // pass true to update
//	}
func SetCustomClaims(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	l.Infof("SetCustomClaims on project: %s", common.ProjectID)

	var r setCustomClaimsRequest
	if err := ctx.ShouldBindJSON(&r); err != nil {
		return []error{err}
	}

	c, err := newCustomClaimsSetter(ctx, l, r)
	if err != nil {
		return []error{err}
	}

	l.Infof("SetCustomClaims for user: %s", r.Email)

	if err := c.setTenantAuthClient(); err != nil {
		return []error{err}
	}

	updateClaims, err := c.setCustomClaims()
	if err != nil {
		return []error{err}
	}

	ctx.JSON(200, updateClaims)

	return nil
}
