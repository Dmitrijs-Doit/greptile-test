package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/gin-gonic/gin"
)

func UpdateAlertsOrgs(ctx *gin.Context) []error {
	var errs []error

	fs, err := firestore.NewClient(ctx, common.ProjectID)

	if err != nil {
		return []error{err}
	}

	defer fs.Close()

	alertDocSnaps, err := fs.Collection("cloudAnalytics").Doc("alerts").Collection("cloudAnalyticsAlerts").Where("organization", "==", organizations.GetDoitOrgRef(fs)).Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	if len(alertDocSnaps) > 0 {
		var alert *domain.Alert

		for _, alertDocSnap := range alertDocSnaps {
			err = alertDocSnap.DataTo(&alert)
			if err != nil {
				errs = append(errs, err)
			}

			if _, err := alertDocSnap.Ref.Update(ctx, []firestore.Update{
				{Path: "organization", Value: alert.Customer.Collection("customerOrgs").Doc("root")},
			}); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}
