package scripts

import (
	"errors"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/gin-gonic/gin"
)

type UpdateFlexsaveConfigurationCustomerInput struct {
	ProjectID string `json:"projectId"`
}

func UpdateFlexsaveConfigTimeEnabled(ctx *gin.Context) []error {
	var params UpdateFlexsaveConfigurationCustomerInput
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	if params.ProjectID == "" {
		err := errors.New("missing project id")
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, params.ProjectID)
	if err != nil {
		return []error{err}
	}

	standAloneConfigSnaps, err := fs.Collection("integrations").Doc("flexsave").Collection("flexsave-payer-configs").Where("type", "==", "aws-flexsave-standalone").Documents(ctx).GetAll()

	if err != nil {
		return []error{err}
	}

	for _, snap := range standAloneConfigSnaps {
		var payerConfig types.PayerConfig

		if err := snap.DataTo(&payerConfig); err != nil {
			return []error{err}
		}

		docRef := fs.Collection("integrations").Doc("flexsave").Collection("configuration").Doc(payerConfig.CustomerID)

		if _, err := docRef.Update(ctx, []firestore.Update{{FieldPath: []string{"AWS", "timeEnabled"}, Value: payerConfig.TimeEnabled}}); err != nil {
			return []error{err}
		}
	}

	return nil
}
