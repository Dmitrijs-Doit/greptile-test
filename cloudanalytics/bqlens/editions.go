package bqlens

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
	optimizerDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	pricebookDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/domain"
)

type ReservationID string

type ReservationsCosts map[ReservationID]map[string]float64

type BQLensQueryArgs struct {
	ReservationAssignments       []optimizerDomain.ReservationAssignment
	CapacityCommitments          []optimizerDomain.CapacityCommitment
	Pricebooks                   pricebookDomain.PriceBooksByEdition
	ReservationMappingWithClause string
	FlatRateUsageTypes           []pricebookDomain.UsageType
	CustomerBQLogsTableID        string
	StartTime                    time.Time
	EndTime                      time.Time
}

const (
	defaultPipelineReservationID ReservationID = "default-pipeline"

	noReservationsSubQueryStr = "0.0\n"
)

// match strings formatted as: projects/reoptimize-io/locations/US/reservations/catch-all
var reservationIDRegex = regexp.MustCompile(`projects/(.*)/locations/(.*)/reservations/(.*)$`)

func GetBQEditionsAnalysisPrices(
	args *BQLensQueryArgs,
) (string, error) {
	// Customer doesn't run any jobs under reservations
	if len(args.ReservationAssignments) == 0 {
		return noReservationsSubQueryStr, nil
	}

	reservationCosts, err := costForReservationID(args)
	if err != nil {
		return "", err
	}

	subQuery := "CASE\n"

	for reservationID, regionPricing := range reservationCosts {
		for region, pricing := range regionPricing {
			subQuery = fmt.Sprintf("%s WHEN reservation = \"%s\" AND location = \"%s\" THEN %v\n",
				subQuery, reservationID, region, pricing)
		}
	}

	subQuery = fmt.Sprintf("%s ELSE 0.0\nEND\n", subQuery)

	return subQuery, nil
}

type editionUsage struct {
	edition   pricebookDomain.Edition
	usageType pricebookDomain.UsageType
}

func costForReservationID(
	args *BQLensQueryArgs,
) (ReservationsCosts, error) {
	if len(args.FlatRateUsageTypes) > 0 {
		return costForLegacyFlatRateReservationID(args)
	}

	return costForEditionsReservationID(args)
}

func costForEditionsReservationID(
	args *BQLensQueryArgs,
) (ReservationsCosts, error) {
	reservationCosts := make(ReservationsCosts)
	uniqueReservations := make(map[editionUsage]struct{})

	for _, reservasionAssignment := range args.ReservationAssignments {
		reservationID, err := parserReservationID(reservasionAssignment.Reservation.Name)
		if err != nil {
			return nil, err
		}

		pbEdition := reservasionAssignment.Reservation.Edition
		edition, err := protoToEdition(pbEdition)
		if err != nil {
			return nil, err
		}

		editionPricebook, ok := args.Pricebooks[edition]
		if !ok {
			return nil, fmt.Errorf("no pricebook found for edition %s", edition)
		}

		plan := findPlanForEdition(args.CapacityCommitments, pbEdition, args.StartTime, args.EndTime)

		var usageType pricebookDomain.UsageType

		switch plan {
		case reservationpb.CapacityCommitment_MONTHLY:
			usageType = pricebookDomain.OnDemand
		case reservationpb.CapacityCommitment_ANNUAL:
			usageType = pricebookDomain.Commit1Yr
		case reservationpb.CapacityCommitment_THREE_YEAR:
			usageType = pricebookDomain.Commit3Yr
		default:
			continue
		}

		uniqueReservations[editionUsage{
			edition:   edition,
			usageType: usageType,
		}] = struct{}{}

		pricebook := *editionPricebook
		reservationCosts[ReservationID(reservationID)] = pricebook[string(usageType)]

	}

	// Handle default-pipeline. For the majority of our users there will only be one
	// capacity commitment used for everything. For those few cases where there's
	// more, we pick one randomly as there is no mechanism to infer what reservation
	// the default-pipeline borrowed slots from. This will result in a cost calculation
	// for the default pipeline that is not 100% accurate, but should be a good enough
	// approximation.
	editionsUsage := []editionUsage{}
	for editionUsage := range uniqueReservations {
		editionsUsage = append(editionsUsage, editionUsage)
	}

	edition := editionsUsage[0].edition
	usageType := editionsUsage[0].usageType
	editionPricebook := args.Pricebooks[edition]
	pricebook := *editionPricebook
	reservationCosts[defaultPipelineReservationID] = pricebook[string(usageType)]

	return reservationCosts, nil
}

func findPlanForEdition(
	capacityCommitments []optimizerDomain.CapacityCommitment,
	pbEdition reservationpb.Edition,
	startTime time.Time,
	endTime time.Time,
) reservationpb.CapacityCommitment_CommitmentPlan {
	// Find the plan for this edition in the capacity commitment
	for _, capacityCommitment := range capacityCommitments {
		if capacityCommitment.Edition == pbEdition &&
			capacityCommitment.CommitmentStartTime.Before(startTime) &&
			capacityCommitment.CommitmentEndTime.After(endTime) {
			return capacityCommitment.Plan
		}
	}

	// A few customers use editions with Pay as you go
	// and don't have an explicit capacity commitment plan.
	return reservationpb.CapacityCommitment_MONTHLY
}

func protoToEdition(edition reservationpb.Edition) (pricebookDomain.Edition, error) {
	switch edition {
	case reservationpb.Edition_STANDARD:
		return pricebookDomain.Standard, nil
	case reservationpb.Edition_ENTERPRISE:
		return pricebookDomain.Enterprise, nil
	case reservationpb.Edition_ENTERPRISE_PLUS:
		return pricebookDomain.EnterprisePlus, nil
	default:
		return "", fmt.Errorf("unsupported edition %s", edition.String())
	}
}

func parserReservationID(s string) (string, error) {
	matches := reservationIDRegex.FindStringSubmatch(s)
	if len(matches) != 4 {
		return "", fmt.Errorf("reservation %s does not match regex", s)
	}

	formatted := fmt.Sprintf("%s:%s.%s", matches[1], matches[2], matches[3])

	return formatted, nil
}

func GetReservationMappingWithClause(
	customerBQLogsTableID string,
	startTime string,
	endTime string,
) string {

	const reservationsMappingWithClause = `
  reservation_mapping AS (
  SELECT
    *
  FROM (
    SELECT
      *,
      ROW_NUMBER() OVER(PARTITION BY project_id) AS _rnk
    FROM (
      SELECT
        protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.reservation AS RESERVATION,
        resource.labels.project_id,
        FORMAT_TIMESTAMP("%Y-%m-%dT%H", timestamp, "UTC") AS usage_hour
      FROM
        {table}
      WHERE
        DATE(timestamp) >= "{start_date}"
        AND DATE(timestamp) <= "{end_date}"
      GROUP BY
        1,
        2,
        3
      HAVING
        RESERVATION IS NOT NULL
        AND RESERVATION <> "unreserved"
        AND RESERVATION <> "default-pipeline" ) )
  WHERE
    _rnk = 1)
`

	return strings.NewReplacer(
		"{table}", customerBQLogsTableID,
		"{start_date}", startTime,
		"{end_date}", endTime,
	).Replace(reservationsMappingWithClause)
}
