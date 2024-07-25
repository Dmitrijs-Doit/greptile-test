package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	pkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/presentations/log"
)

const flexsaveUpdateErrorFmt = "failed to update Flexsave AWS savings for presentation customer %s: %v"

/*
Extracts oldest savings month start date from Flexsave savings history.
*/
func getSavingsStartTime(savingsHistory map[string]*pkg.FlexsaveMonthSummary) (*time.Time, error) {
	timeEnabled := time.Now()

	for month := range savingsHistory {
		dateParts := strings.Split(month, "_")

		month, err := strconv.Atoi(dateParts[0])
		if err != nil {
			return nil, err
		}

		year, err := strconv.Atoi(dateParts[1])
		if err != nil {
			return nil, err
		}

		savingsMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)

		if savingsMonth.Before(timeEnabled) {
			timeEnabled = savingsMonth
		}
	}

	return &timeEnabled, nil
}

/*
Creates Flexsave config and updates savings history and summary for each presentation customer with AWS assets.
Calculates and updates `timeEnabled` field based on savings history.
Sets `hasActiveResold` field to true.
Updates `/integrations/flexsave/configuration/{customerID}` documents.
*/
func (p *PresentationService) UpdateFlexsaveAWSSavings(ctx *gin.Context) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "flexsave")

	docSnaps, err := p.customersDAL.GetPresentationCustomersWithAssetType(ctx, common.Assets.AmazonWebServices)
	if err != nil {
		return fmt.Errorf(FetchCustomerErr, err)
	}

	for _, docSnap := range docSnaps {
		customerID := docSnap.Ref.ID

		l.Infof("Flexsave AWS savings update for customer: %s", customerID)

		if err := p.awsStandaloneService.UpdateStandaloneCustomerSpendSummary(ctx, customerID, 2); err != nil {
			return fmt.Errorf(flexsaveUpdateErrorFmt, customerID, err)
		}

		fsc, err := p.integrationsDAL.GetFlexsaveConfigurationCustomer(ctx, customerID)
		if err != nil {
			return fmt.Errorf(flexsaveUpdateErrorFmt, customerID, err)
		}

		timeEnabled, err := getSavingsStartTime(fsc.AWS.SavingsHistory)
		if err != nil {
			return fmt.Errorf(flexsaveUpdateErrorFmt, customerID, err)
		}

		fsc.AWS.TimeEnabled = timeEnabled
		fsc.AWS.HasActiveResold = true

		if err := p.integrationsDAL.UpdateFlexsaveConfigurationCustomer(
			ctx,
			customerID,
			map[string]*pkg.FlexsaveSavings{
				"AWS": &fsc.AWS,
			}); err != nil {
			return fmt.Errorf(flexsaveUpdateErrorFmt, customerID, err)
		}
	}

	return nil
}
