package bqlens

import (
	"fmt"
	"slices"
	"strings"

	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
	pricebookDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/domain"
)

/*
WITH editions_customers AS (
SELECT
discounts.customer,
-- project.id as project,
sku.description,
FROM
doitintl-cmp-gcp-data.gcp_billing.gcp_raw_billing billing
RIGHT JOIN (
SELECT
DISTINCT billing_account_id,
customer
FROM
doitintl-cmp-gcp-data.gcp_billing.gcp_discounts_v1
WHERE
DATE(_PARTITIONTIME) = CURRENT_DATE()) discounts
ON
billing.billing_account_id = discounts.billing_account_id
WHERE
DATE(export_time) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 90 DAY))
AND service.description like "%BigQuery%"
AND LOWER(sku.description) LIKE "%flat rate%" AND LOWER(sku.description) NOT LIKE "%bi engine%"
GROUP BY
1,2)

SELECT
customer,
(SELECT name from `me-doit-intl-com.customers.customer_list` WHERE customer_id = editions.customer) as name,
-- project,
-- description,
FROM editions_customers editions
group by 1,2
order by 2 asc
*/

var legacyFlatrateUsers = []string{
	"FgffPnB2CXOxpJNrenEY", // ADARA
	"sWBxBFXR8oKsrQfWutwy", // Algolia
	"dAxipYVeCzk6j69kIwad", // Apartment List
	"xxrSvBUz3cHUgJd0zJGK", // AppsFlyer
	"6t5ofgV7jPEuo5xL3L9Z", // Arpeely
	"BKuYrtL3FJ60wQmYnu3r", // Beryl
	"4c27g7jZQxws8EjK9ArJ", // CB4
	"wUmwj9Oc6ZFSWz5vrxg0", // CRED
	"9IjFSCQQCSqaubW6aNc6", // Cover Genius
	"9JJLCU5TqewnK8udHUfo", // Ding
	"EE8CtpzYiKp0dVAESVrB", // DoiT International
	"kwsZVx8gNUrnuJpC24Ay", // EDGE226 (prev. PLYmedia)
	"1hqfbivOJjj4c9DIsZZy", // Energyworx
	"KzDE3pXU4KeQ3b53ciDN", // Exabeam
	"K0wlFSMFieUaCXmbKXFY", // Freightos
	"d6j0IpuqjNdMzuk17HbK", // Guardio
	"gNZoYGk9PvPiojEtQsSQ", // Handshake
	"TbSMwrqYfjolouQAqpng", // Kiwi.com
	"8cDjJ8neBjTvOCzSGc68", // Knowunity
	"xAscz4jClPuw7yKB7CxW", // LendingPoint
	"VNYRPNgx0QIQKXzqoVcB", // Localis
	"MIvq180q34a73OCAMryA", // Mattress Firm
	"FLMkRTxRjCRvW266e9GL", // Money Gram International
	"2Gi0e4pPA3wsfJNOOohW", // Moon Active
	"o6P4wEOko4Df7UFehPfQ", // Narvar
	"DgCxIRosorS59fUGearB", // Nectar sleep
	"CJOOmZSPmzIFGQOZvNNh", // Northbeam
	"raBKSxmOC0A6FerlgoiX", // OfferFit
	"p4DylGGvU7oQUhn8AbyV", // PerimeterX
	"Gt165vsg5unrihqewVsa", // Placer.ai
	"UYrRuyzxa4lVlQjfGbu6", // PurpleLab
	"OcEEe4IwjgrJ7gwaillS", // Quandoo
	"crfboGxfAjt47x0iEzr1", // Recurly
	"OkPM0a871sHnnkSNYhlP", // SevenRooms
	"gJKYgbam4UplkdHt6DSJ", // SiriusXM
	"dXKtkPktymtunKRbk01y", // Zendesk
	"FWeXAa44MS54AIFT9fmT", // momox
	"OaW6gw5ULi7yaEsP2keM", // tryzapp.com
	"BivdKHPTMnFBypzMev2M", // typea.group
}

const flatRateSKUQueryTpl = `
SELECT
  DISTINCT(sku.description),
FROM
  doitintl-cmp-gcp-data.gcp_billing.gcp_raw_billing billing
RIGHT JOIN (
  SELECT
    DISTINCT billing_account_id,
    customer
  FROM
    doitintl-cmp-gcp-data.gcp_billing.gcp_discounts_v1
  WHERE
    DATE(_PARTITIONTIME) = CURRENT_DATE()) discounts
ON
  billing.billing_account_id = discounts.billing_account_id
WHERE
  DATE(export_time) >= DATE("{start_time}")
  AND DATE(export_time) <= DATE("{end_time}")
  AND service.description LIKE "%BigQuery%"
  AND LOWER(sku.description) LIKE "%flat rate%"
  AND LOWER(sku.description) NOT LIKE "%bi engine%"
  AND customer = "{customer_id}"
`

func IsLegacyFlatRateUser(customerID string) bool {
	return slices.Contains(legacyFlatrateUsers, customerID)
}

// TODO(CMP-21119): Retire this entire file when we no longer have customers with these SKUs.
func GetLegacyFlatRateSKUsQuery(customerID string, startTime string, endTime string) string {
	return strings.NewReplacer(
		"{customer_id}", customerID,
		"{start_time}", startTime,
		"{end_time}", endTime,
	).Replace(flatRateSKUQueryTpl)
}

func costForLegacyFlatRateReservationID(
	args *BQLensQueryArgs,
) (ReservationsCosts, error) {
	reservationCosts := make(ReservationsCosts)
	uniqueReservations := make(map[editionUsage]struct{})

	editionPricebook, ok := args.Pricebooks[pricebookDomain.LegacyFlatRate]
	if !ok {
		return nil, fmt.Errorf("no pricebook found for legacy flat rate")
	}

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

		plan := findPlanForEdition(args.CapacityCommitments, pbEdition, args.StartTime, args.EndTime)

		var usageType pricebookDomain.UsageType

		switch plan {
		case reservationpb.CapacityCommitment_MONTHLY:
			usageType = pricebookDomain.Commit1Mo
		case reservationpb.CapacityCommitment_ANNUAL:
			usageType = pricebookDomain.Commit1Yr
		default:
			// We cannot map an edition capacity commitment to a flat rate SKU usage, so pick the first one.
			usageType = args.FlatRateUsageTypes[0]
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

	usageType := editionsUsage[0].usageType
	pricebook := *editionPricebook
	reservationCosts[defaultPipelineReservationID] = pricebook[string(usageType)]

	return reservationCosts, nil
}
