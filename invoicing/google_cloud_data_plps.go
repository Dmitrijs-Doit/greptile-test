package invoicing

import (
	"context"
	"sort"
	"time"

	"cloud.google.com/go/firestore"

	contractDomain "github.com/doitintl/hello/scheduled-tasks/contract/domain"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
)

// We are interested in contracts that satisfy either of these conditions:
// - They are active and their start date needs to be before the end of the billing period.
// - They are inactive but their end date is after the begining of the invoicing period.
func (s *InvoicingService) getPLPSCharges(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	assetRef *firestore.DocumentRef,
	invoicingStartDate time.Time,
	invoicingEndDate time.Time,
) (domain.SortablePLPSCharges, error) {
	plpsCharges := domain.SortablePLPSCharges{}

	contracts, err := s.contractDAL.GetContractsByType(ctx, customerRef, contractDomain.ContractTypeGoogleCloudPLPS)
	if err != nil {
		return nil, err
	}

	for _, contract := range contracts {
		// Skip canceled contracts.
		if !contract.Active {
			continue
		}

		// Skip contracts that start after the invoicing end date.
		if contract.StartDate.After(invoicingEndDate) {
			continue
		}

		// Skip commitment contracts that ended before the invoicing start date.
		if !contract.EndDate.IsZero() && contract.EndDate.Before(invoicingStartDate) {
			continue
		}

		for _, ref := range contract.Assets {
			if ref.ID != assetRef.ID {
				continue
			}

			startDate := contract.StartDate

			var endDate time.Time

			if contract.EndDate.IsZero() {
				// On demand contracts have no end date.
				endDate = invoicingEndDate.AddDate(0, 0, 1)
			} else {
				endDate = contract.EndDate
			}

			plpsCharges = append(
				plpsCharges,
				&domain.PLPSCharge{
					StartDate:   startDate,
					EndDate:     endDate,
					PLPSPercent: contract.PLPSPercent,
				},
			)
		}
	}

	sort.Sort(plpsCharges)

	return plpsCharges, nil
}

func (s *InvoicingService) calculateNewPLPSCost(
	row *QueryProjectRow,
	gcpPLPSChargePercent float64,
	plpsCharges domain.SortablePLPSCharges,
) (float64, error) {
	rowDate := time.Date(row.Date.Year, row.Date.Month, row.Date.Day, 0, 0, 0, 0, time.UTC)
	onePercent := row.Cost / gcpPLPSChargePercent

	for _, c := range plpsCharges {
		if rowDate.Before(c.StartDate) || !rowDate.Before(c.EndDate) {
			continue
		}

		newCost := onePercent * c.PLPSPercent
		return newCost, nil
	}

	return row.Cost, ErrNoSuitablePLPSContractFound
}
