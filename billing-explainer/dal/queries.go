package dal

const tempFunction = `CREATE TEMP FUNCTION
getKeyFromSystemLabels(labels ARRAY<STRUCT<KEY STRING,
    value STRING>>,
    key_lookup STRING)
RETURNS STRING AS ( (
    SELECT
    l.value
    FROM
    UNNEST(labels) l
    WHERE
    l.key = key_lookup
    LIMIT
    1) );`

func GetSummaryPageQuery(customerSnapshotTable, payerTable, accountIDString, PayerID, flexsaveCondition string) string {
	const flexsaveAccountFilter = "SELECT DISTINCT(aws_account_id) FROM me-doit-intl-com.measurement.flexsave_accounts"

	var baseQuery = tempFunction + `

	WITH
		-- DoiT data
		src AS (
        SELECT
            cost_type,
            service_id,
            report,
            cost,
            description,
			project_id,
            project.name AS project_name,
			service_description,
			system_labels
        FROM
            ` + customerSnapshotTable + ` WHERE (project_id IN (` + accountIDString + `)` + flexsaveCondition + ` ) AND getKeyFromSystemLabels(system_labels, "aws/payer_account_id") = '` + PayerID + `'),
        -- Get DoiT support breakdown
        doit_support AS (
        SELECT
            project_id, service_description, description, getKeyFromSystemLabels(system_labels, 'doit/aws_support_group_id') AS support_group, sum(cost) as cost
        FROM
            src
        WHERE
            LOWER(service_id) LIKE '%support%'
        GROUP BY 1,2,3,4
        ),
        -- Get the RDS Savings from FlexsaveManagementFee lineItems
        flexsave_rds_savings AS (
               SELECT
                     IFNULL(SUM(r.savings), 0.0) AS cost
                FROM
                     src, UNNEST(report) r
                WHERE cost_type = "FlexsaveManagementFee" ),

        support_base_cost AS (
            SELECT
                getKeyFromSystemLabels(system_labels, 'doit/aws_support_group_id') support_group, CAST(SUM(r.ext_metric.value) AS FLOAT64) as base_cost
            FROM
               src, UNNEST(report) r
            WHERE
                r.ext_metric.key = 'support_base_cost'
            GROUP BY 1
        ), doit_support_details AS (
            SELECT
                ARRAY_AGG(STRUCT(project_id, service_description, description, cost, IFNULL(base_cost, 0.0) AS base_cost)) AS details
            FROM
                doit_support LEFT JOIN support_base_cost USING (support_group)
            WHERE cost > 0
        ),
        doit_invoice AS (
        SELECT
            SUM(cost) AS cost,
            CASE
				WHEN cost_type IN ("Fee", "RIFee", "SavingsPlanRecurringFee", "FlexsaveRIFee") OR (cost_type = "FlexsaveRecurringFee" OR project_name = "Flexsave"
                	OR (cost_type != "FlexsaveRIFee" AND project_name IN (` + flexsaveAccountFilter + `)) ) -- Apply filter on project_name for FSAccounts to extract FSCharges
				THEN "OtherCharges"
            	WHEN cost_type IN ("DiscountedUsage", "FlexsaveUsage") THEN "Savings"
				ELSE 'Service' END AS type,
            CASE
                WHEN cost_type = "Fee" THEN "ocbCharges"
                WHEN (cost_type = "FlexsaveRecurringFee"
                OR project_name = "Flexsave") THEN "flexsaveCharges"
                WHEN cost_type != "FlexsaveRIFee" AND project_name IN (` + flexsaveAccountFilter + `)  THEN "flexsaveCharges"
                WHEN cost_type  IN ("FlexsaveRIFee", "RIFee") THEN "reservationRecurringFee"
                WHEN cost_type IN ("DiscountedUsage", "FlexsaveUsage") THEN "reservationAppliedUsage"
                WHEN cost_type IN ("FlexsaveManagementFee") THEN "Usage"
                ELSE
                cost_type
            END
                AS cost_type,
            'DoiT' AS source,
            [] AS details,
        FROM
            src
        WHERE
            (cost_type like '%Usage%' or cost_type like '%RecurringFee%' OR cost_type LIKE '%Fee%')
        	AND NOT (service_id LIKE '%AWS%Support%' OR service_id = 'OCBPremiumSupport')
			AND cost_type != 'Tax'
		GROUP BY
			cost_type , type, source

		UNION ALL

        SELECT
            SUM(cost),
            'Savings' AS type,
            CASE WHEN cost_type = "FlexsaveNegation" THEN "FlexsaveSavings" ELSE
            cost_type END AS cost_type,
            'DoiT' AS source,
            [] AS details,
        FROM
            src
        WHERE
            cost_type IN ('SavingsPlanNegation',
            'FlexsaveNegation')
        GROUP BY
            cost_type,
            type,
            source

        UNION ALL

        SELECT
            SUM(cost),
            'Support' AS type,
            cost_type,
            'DoiT' AS source,
            (SELECT details FROM doit_support_details) AS details,
        FROM
            src
        WHERE
			(service_id LIKE '%AWS%Support%' OR service_id = 'OCBPremiumSupport')
			AND cost_type NOT IN ('Tax', 'Credit')
			GROUP BY cost_type, type, source

		UNION ALL

        SELECT
			SUM(cost),
        	CASE WHEN cost_type = 'Credit' THEN 'Credit' ELSE 'Discount' END as type,
            cost_type,
            'DoiT' AS source,
            [] AS details,
        FROM
            src
        WHERE
            cost < 0
            AND cost_type NOT IN ('SavingsPlanNegation', 'FlexsaveNegation', 'Refund', 'Tax')
        GROUP BY
            cost_type, type, source

        UNION ALL

		SELECT
			-1 * SUM(r.savings) AS cost,
            'Discount' as type,
        	CASE WHEN r.savings_description = 'EDP' THEN 'EdpDiscount'
            	WHEN r.savings_description = 'Private Pricing' THEN 'PrivateRateDiscount'
            	END AS cost_type,
        	'DoiT' AS source,
        	[] AS details,
        FROM
        	src, UNNEST(report) r
        WHERE
        	r.savings_description IS NOT NULL
        	AND r.savings_description != ''
        	AND r.savings_description != 'Flexsave'
			AND NOT (service_id LIKE '%AWS%Support%' OR service_id = 'OCBPremiumSupport')
        GROUP BY
        	cost_type, type, source

        UNION ALL

        SELECT
            cost * -1 AS cost,
            "Savings" AS type,
			"FlexsaveSavings" AS cost_type,
			'DoiT' AS source,
        	[] AS details
		FROM
			flexsave_rds_savings

        UNION ALL
        SELECT
            cost,
            "Service" AS type,
			"Usage" AS cost_type,
			'DoiT' AS source,
        	[] AS details
		FROM
        flexsave_rds_savings

        UNION ALL

		SELECT
            SUM(cost),
            CASE WHEN cost_type = "Tax" THEN "Tax" ELSE "Refund" END AS type,
            cost_type,
            'DoiT' AS source,
            [] AS details,
        FROM
            src
        WHERE
            cost_type IN ("Tax", "Refund")
        GROUP BY
            cost_type,
            type,
            source

        ),
		-- Payer data
        payer_src AS (
            SELECT
            cost_type,
            service_id,
            service_description,
            report,
            cost,
            description,
			project_id
        FROM
       ` + payerTable + `
        WHERE DATE(export_time) BETWEEN @startDateTime AND @endDateTime
			AND (project_id IN (` + accountIDString + `)` + flexsaveCondition + ` )
			AND getKeyFromSystemLabels(system_labels, "aws/payer_account_id") = '` + PayerID + `'
		),
        -- Get Payer support breakdown
        payer_support AS (
            SELECT
                project_id,
                service_description,
                cost_type,
                description,
                SUM(cost) AS cost,
                0.0 AS base_cost,
            FROM
                payer_src
            WHERE
                (service_id LIKE '%AWS%Support%' OR service_id = 'OCBPremiumSupport')
				AND cost_type NOT IN ('Tax', 'Credit', 'Refund')
				AND cost > 0
            GROUP BY 1,2,3,4
        ),
        payer_support_details AS (
            SELECT
                ARRAY_AGG(STRUCT(project_id, service_description, description, cost, base_cost)) AS details
            FROM
                payer_support
        ),
        aws_invoice AS (
        SELECT
            SUM(cost) AS cost,
            CASE WHEN cost_type IN ("Fee", "RIFee", "SavingsPlanRecurringFee")
            THEN "OtherCharges"
            WHEN cost_type IN ("DiscountedUsage") THEN "Savings"
            ELSE
            'Service' END AS  type,
            CASE WHEN cost_type = "Fee" THEN "ocbCharges"
            WHEN cost_type  in ("RIFee") THEN "reservationRecurringFee"
            WHEN cost_type = "DiscountedUsage" THEN "reservationAppliedUsage"
            ELSE cost_type END AS cost_type,
            'AWS' AS source,
            [] AS details,
        FROM
            payer_src
        WHERE
            (cost_type like '%Usage%' or cost_type like '%RecurringFee%' OR cost_type LIKE '%Fee%') AND
			NOT (service_id LIKE '%AWS%Support%' OR service_id = 'OCBPremiumSupport')
			AND cost_type != 'Tax'
		GROUP BY cost_type , type, source

        UNION ALL

        SELECT
            SUM(cost) AS cost,
            'Savings'
            AS type,
            cost_type,
            'AWS' AS source,
            [] AS details,
        FROM
            payer_src
        WHERE
            cost_type IN ('SavingsPlanNegation')
        GROUP BY
            cost_type,
            type,
            source

        UNION ALL

        SELECT
            SUM(cost),
            'Support' AS type,
            'Usage' AS cost_type,
            'AWS' AS source,
            (SELECT details FROM payer_support_details) AS details,
        FROM
            payer_support
        GROUP BY
			cost_type, type, source

        UNION ALL

        SELECT
			SUM(cost),
        	CASE WHEN cost_type = 'Credit' THEN 'Credit' ELSE 'Discount' END AS type,
            cost_type,
            'AWS' AS source,
            [] AS details,
        FROM
            payer_src
        WHERE
            cost < 0
            AND cost_type NOT IN ('SavingsPlanNegation', 'Refund', 'Tax')
        GROUP BY
            cost_type, type, source

        UNION ALL

        SELECT
            SUM(cost),
            CASE WHEN cost_type = "Tax" THEN "Tax" ELSE "Refund" END AS type,
            cost_type,
            'AWS' AS source,
            [] AS details,
        FROM
            payer_src
        WHERE
            cost_type IN ("Tax", "Refund")
        GROUP BY
            cost_type,
            type,
            source
    )

	-- Union between both sources
    SELECT * FROM aws_invoice
	UNION ALL
    SELECT * FROM doit_invoice
    `

	return baseQuery
}

func GetServiceBreakdownQuery(customerSnapshotTable, payerTable, accountIDString, PayerID, flexsaveCondition string) string {
	var baseQuery = tempFunction + `
            
        WITH
        -- DoiT data
        src AS (
        SELECT
            cost_type,
            service_id,
            report,
            cost,
            description,
            project_id,
            project.name AS project_name,
            service_description,
            system_labels
        FROM
            ` + customerSnapshotTable + `
        WHERE
            (project_id IN (` + accountIDString + `)` + flexsaveCondition + ` )
            AND getKeyFromSystemLabels(system_labels,
            "aws/payer_account_id") = '` + PayerID + `'),
        -- Get the RDS Savings from FlexsaveManagementFee lineItems
        flexsave_rds_savings AS (
        SELECT
            service_id,
            IFNULL(SUM(r.savings), 0.0) AS cost
        FROM
            src,
            UNNEST(report) r
        WHERE
            cost_type = "FlexsaveManagementFee"
        GROUP BY
            1),
        doit_invoice AS (
        SELECT
            service_id,
            SUM(cost) AS cost,
            CASE
            WHEN cost_type = "Fee" THEN "ocbCharges"
            WHEN (cost_type = "FlexsaveRecurringFee" OR project_name = "Flexsave") THEN "flexsaveCharges"
            WHEN cost_type != "FlexsaveRIFee" AND project_name IN ( SELECT DISTINCT(aws_account_id) FROM me-doit-intl-com.measurement.flexsave_accounts) THEN "flexsaveCharges"
            WHEN cost_type IN ("FlexsaveRIFee",
            "RIFee") THEN "reservationRecurringFee"
            WHEN cost_type IN ("DiscountedUsage", "FlexsaveUsage") THEN "reservationAppliedUsage"
            WHEN cost_type IN ("FlexsaveManagementFee") THEN "Usage"
            ELSE
            cost_type
        END
            AS cost_type,
            'DoiT' AS source
        FROM
            src
        WHERE
            (cost_type LIKE '%Usage%'
            OR cost_type LIKE '%RecurringFee%'
            OR cost_type LIKE '%Fee%')
            AND NOT (service_id LIKE '%AWS%Support%'
            OR service_id = 'OCBPremiumSupport')
            AND cost_type != 'Tax'
        GROUP BY
            cost_type,
            source,
            service_id
        UNION ALL
        SELECT
            service_id,
            SUM(cost),
            CASE
            WHEN cost_type = "FlexsaveNegation" THEN "FlexsaveSavings"
            ELSE
            cost_type
        END
            AS cost_type,
            'DoiT' AS source
        FROM
            src
        WHERE
            cost_type IN ('SavingsPlanNegation',
            'FlexsaveNegation')
        GROUP BY
            cost_type,
            source,
            service_id
        UNION ALL
        SELECT
            service_id,
            SUM(cost),
            cost_type,
            'DoiT' AS source
        FROM
            src
        WHERE
            ((service_id LIKE '%AWS%Support%'
            OR service_id = 'OCBPremiumSupport')
            AND cost_type != 'Tax')
			OR (cost < 0
            AND cost_type NOT IN ('SavingsPlanNegation',
            'FlexsaveNegation',
            'Refund',
            'Tax'))
        GROUP BY
            cost_type,
            source,
            service_id
        UNION ALL
        SELECT
            service_id,
            -1 * SUM(r.savings) AS cost,
            CASE
            WHEN r.savings_description = 'EDP' THEN 'EdpDiscount'
            WHEN r.savings_description = 'Private Pricing' THEN 'PrivateRateDiscount'
        END
            AS cost_type,
            'DoiT' AS source
        FROM
            src,
            UNNEST(report) r
        WHERE
            r.savings_description IS NOT NULL
            AND r.savings_description != ''
            AND r.savings_description != 'Flexsave'
			AND NOT (service_id LIKE '%AWS%Support%' OR service_id = 'OCBPremiumSupport')
        GROUP BY
            cost_type,
            source,
            service_id
        UNION ALL
        SELECT
            service_id,
            cost * -1 AS cost,
            "FlexsaveSavings" AS cost_type,
            'DoiT' AS source,
        FROM
            flexsave_rds_savings
        UNION ALL
        SELECT
            service_id,
            cost,
            "FlexsaveRDSManagementFee" AS cost_type,
            'DoiT' AS source
        FROM
            flexsave_rds_savings
        UNION ALL
        SELECT
            service_id,
            SUM(cost),
            cost_type,
            'DoiT' AS source
        FROM
            src
        WHERE
            cost_type IN ( "Refund", "Tax")
        GROUP BY
            cost_type,
            source,
            service_id),
        -- Payer data
        payer_src AS (
        SELECT
            cost_type,
            service_id,
            service_description,
            report,
            cost,
            description,
            project_id
        FROM
            ` + payerTable + `
        WHERE
            DATE(export_time) BETWEEN @startDateTime AND @endDateTime
			AND (project_id IN (` + accountIDString + `)` + flexsaveCondition + ` )
			AND getKeyFromSystemLabels(system_labels, "aws/payer_account_id") = '` + PayerID + `'
		),
        -- Get Payer support breakdown
        payer_support AS (
        SELECT
            project_id,
            service_description,
            service_id,
            cost_type,
            description,
            SUM(cost) AS cost,
            0.0 AS base_cost,
        FROM
            payer_src
        WHERE
            (service_id LIKE '%AWS%Support%'
            OR service_id = 'OCBPremiumSupport')
            AND cost_type NOT IN ('Tax',
            'Credit',
            'Refund')
            AND cost > 0
        GROUP BY
            1,
            2,
            3,
            4,
            5 ),
        aws_invoice AS (
        SELECT
            service_id,
            SUM(cost) AS cost,
            CASE
            WHEN cost_type = "Fee" THEN "ocbCharges"
            WHEN cost_type IN ("RIFee") THEN "reservationRecurringFee"
            WHEN cost_type = "DiscountedUsage" THEN "reservationAppliedUsage"
            ELSE
            cost_type
        END
            AS cost_type,
            'AWS' AS source
        FROM
            payer_src
        WHERE
            (cost_type LIKE '%Usage%'
            OR cost_type LIKE '%RecurringFee%'
            OR cost_type LIKE '%Fee%')
            AND NOT (service_id LIKE '%AWS%Support%'
            OR service_id = 'OCBPremiumSupport')
            AND cost_type != 'Tax'
        GROUP BY
            cost_type,
            source,
            service_id
        UNION ALL
        SELECT
            service_id,
            SUM(cost) AS cost,
            cost_type,
            'AWS' AS source,
        FROM
            payer_src
        WHERE
            cost_type IN ('SavingsPlanNegation')
        GROUP BY
            cost_type,
            source,
            service_id
        UNION ALL
        SELECT
            service_id,
            SUM(cost),
            'Usage' AS cost_type,
            'AWS' AS source
        FROM
            payer_support
        GROUP BY
            cost_type,
            source,
            service_id
        UNION ALL
        SELECT
            service_id,
            SUM(cost),
            cost_type,
            'AWS' AS source,
        FROM
            payer_src
        WHERE
            cost < 0
            AND cost_type NOT IN ('SavingsPlanNegation',
            'Refund',
            'Tax')
        GROUP BY
            cost_type,
            source,
            service_id
        UNION ALL
        SELECT
            service_id,
            SUM(cost),
            cost_type,
            'AWS' AS source
        FROM
            payer_src
        WHERE
            cost_type IN ("Tax", "Refund")
        GROUP BY
            cost_type,
            source,
            service_id ),
        -- Union between both sources
        res AS (
        SELECT
            *
        FROM
            aws_invoice
        UNION ALL
        SELECT
            *
        FROM
            doit_invoice ),
        service_id_description_mapping_doit AS (
        SELECT
            service_id,
            MAX(service_description) AS service_description
        FROM
            src
        GROUP BY
            service_id ),
        service_id_description_mapping AS (
        SELECT
            service_id,
            MAX(service_description) AS service_description
        FROM
            payer_src
        GROUP BY
            service_id ),
		
        final_service_id_description_mapping AS (
            SELECT
                CASE
                WHEN doit.service_id IS NULL THEN payer.service_id
                WHEN payer.service_id IS NULL THEN doit.service_id
                ELSE
                payer.service_id
            END
                AS service_id,
                CASE
                WHEN doit.service_description IS NULL THEN payer.service_description
                WHEN payer.service_description IS NULL THEN doit.service_description
                ELSE
                payer.service_description
            END
                AS service_description,
            FROM
                service_id_description_mapping payer
            FULL JOIN
                service_id_description_mapping_doit doit
            USING
                (service_id) )
        SELECT
			CASE
				WHEN service_id IN ("AWS Premium Support", "AWS Support Costs", "OCBPremiumSupport", "AWS Support (Enterprise)") THEN "AWS Support"
				WHEN service_id IN ("Discount") THEN "Discount"
                WHEN cost_type = "FlexsaveAdjustment" THEN "Service/NA"
			ELSE
			s.service_description
			END
			AS service_description,
			source,
			ARRAY_AGG(STRUCT(cost_type,
				cost)) AS cost_breakdown
        FROM
        	res
        LEFT JOIN
            final_service_id_description_mapping s
        USING
        	(service_id)
        WHERE
        	cost_type != "Tax"
        GROUP BY
			1,
			2
		ORDER BY
			1,
			2`

	return baseQuery
}

func GetAccountBreakdownQuery(customerSnapshotTable, payerTable, accountIDString, PayerID, flexsaveCondition string) string {
	var baseQuery = tempFunction + `
        WITH
        -- DoiT data
        src AS (
        SELECT
            cost_type,
            service_id,
            report,
            cost,
            description,
            project_id,
            project.name AS project_name,
            service_description,
            system_labels
        FROM
            ` + customerSnapshotTable + `
        WHERE
            (project_id IN (` + accountIDString + `)` + flexsaveCondition + ` )
            AND getKeyFromSystemLabels(system_labels,
            "aws/payer_account_id") = '` + PayerID + `'),
        -- Get the RDS Savings from FlexsaveManagementFee lineItems
        flexsave_rds_savings AS (
        SELECT
            project_id,
            IFNULL(SUM(r.savings), 0.0) AS cost
        FROM
            src,
            UNNEST(report) r
        WHERE
            cost_type = "FlexsaveManagementFee"
        GROUP BY
            1),
        doit_invoice AS (
        SELECT
            project_id,
            SUM(cost) AS cost,
            CASE
            WHEN cost_type = "Fee" THEN "ocbCharges"
            WHEN (cost_type = "FlexsaveRecurringFee" OR project_name = "Flexsave") THEN "flexsaveCharges"
            WHEN cost_type != "FlexsaveRIFee" AND project_name IN ( SELECT DISTINCT(aws_account_id) FROM me-doit-intl-com.measurement.flexsave_accounts) THEN "flexsaveCharges"
            WHEN cost_type IN ("FlexsaveRIFee",
            "RIFee") THEN "reservationRecurringFee"
            WHEN cost_type IN ("DiscountedUsage", "FlexsaveUsage") THEN "reservationAppliedUsage"
            WHEN cost_type IN ("FlexsaveManagementFee") THEN "Usage"
            ELSE
            cost_type
        END
            AS cost_type,
            'DoiT' AS source
        FROM
            src
        WHERE
            (cost_type LIKE '%Usage%'
            OR cost_type LIKE '%RecurringFee%'
            OR cost_type LIKE '%Fee%')
            AND NOT (service_id LIKE '%AWS%Support%'
            OR service_id = 'OCBPremiumSupport')
            AND cost_type != 'Tax'
        GROUP BY
            cost_type,
            source,
            project_id
        UNION ALL
        SELECT
            project_id,
            SUM(cost),
            CASE
            WHEN cost_type = "FlexsaveNegation" THEN "FlexsaveSavings"
            ELSE
            cost_type
        END
            AS cost_type,
            'DoiT' AS source
        FROM
            src
        WHERE
            cost_type IN ('SavingsPlanNegation',
            'FlexsaveNegation')
        GROUP BY
            cost_type,
            source,
            project_id
        UNION ALL
        SELECT
            project_id,
            SUM(cost),
            cost_type,
            'DoiT' AS source
        FROM
            src
        WHERE
            ((service_id LIKE '%AWS%Support%'
            OR service_id = 'OCBPremiumSupport')
            AND cost_type != 'Tax')
			OR (cost < 0
            AND cost_type NOT IN ('SavingsPlanNegation',
            'FlexsaveNegation',
            'Refund',
            'Tax'))
        GROUP BY
            cost_type,
            source,
            project_id
        UNION ALL
        SELECT
            project_id,
            -1 * SUM(r.savings) AS cost,
            CASE
            WHEN r.savings_description = 'EDP' THEN 'EdpDiscount'
            WHEN r.savings_description = 'Private Pricing' THEN 'PrivateRateDiscount'
        END
            AS cost_type,
            'DoiT' AS source
        FROM
            src,
            UNNEST(report) r
        WHERE
            r.savings_description IS NOT NULL
            AND r.savings_description != ''
            AND r.savings_description != 'Flexsave'
			AND NOT (service_id LIKE '%AWS%Support%' OR service_id = 'OCBPremiumSupport')
        GROUP BY
            cost_type,
            source,
            project_id
        UNION ALL
        SELECT
            project_id,
            cost * -1 AS cost,
            "FlexsaveSavings" AS cost_type,
            'DoiT' AS source,
        FROM
            flexsave_rds_savings
        UNION ALL
        SELECT
            project_id,
            cost,
            "FlexsaveRDSManagementFee" AS cost_type,
            'DoiT' AS source
        FROM
            flexsave_rds_savings
        UNION ALL
        SELECT
            project_id,
            SUM(cost),
            cost_type,
            'DoiT' AS source
        FROM
            src
        WHERE
            cost_type IN ( "Refund", "Tax")
        GROUP BY
            cost_type,
            source,
            project_id),
        -- Payer data
        payer_src AS (
        SELECT
            cost_type,
            service_id,
            service_description,
            report,
            cost,
            description,
            project_id
        FROM
            ` + payerTable + `
        WHERE
            DATE(export_time) BETWEEN @startDateTime AND @endDateTime
			AND (project_id IN (` + accountIDString + `)` + flexsaveCondition + ` )
			AND getKeyFromSystemLabels(system_labels, "aws/payer_account_id") = '` + PayerID + `'
		),
        -- Get Payer support breakdown
        payer_support AS (
        SELECT
            project_id,
            service_description,
            service_id,
            cost_type,
            description,
            SUM(cost) AS cost,
            0.0 AS base_cost,
        FROM
            payer_src
        WHERE
            (service_id LIKE '%AWS%Support%'
            OR service_id = 'OCBPremiumSupport')
            AND cost_type NOT IN ('Tax',
            'Credit',
            'Refund')
            AND cost > 0
        GROUP BY
            1,
            2,
            3,
            4,
            5 ),
        aws_invoice AS (
        SELECT
            project_id,
            SUM(cost) AS cost,
            CASE
            WHEN cost_type = "Fee" THEN "ocbCharges"
            WHEN cost_type IN ("RIFee") THEN "reservationRecurringFee"
            WHEN cost_type = "DiscountedUsage" THEN "reservationAppliedUsage"
            ELSE
            cost_type
        END
            AS cost_type,
            'AWS' AS source
        FROM
            payer_src
        WHERE
            (cost_type LIKE '%Usage%'
            OR cost_type LIKE '%RecurringFee%'
            OR cost_type LIKE '%Fee%')
            AND NOT (service_id LIKE '%AWS%Support%'
            OR service_id = 'OCBPremiumSupport')
            AND cost_type != 'Tax'
        GROUP BY
            cost_type,
            source,
            project_id
        UNION ALL
        SELECT
            project_id,
            SUM(cost) AS cost,
            cost_type,
            'AWS' AS source,
        FROM
            payer_src
        WHERE
            cost_type IN ('SavingsPlanNegation')
        GROUP BY
            cost_type,
            source,
            project_id
        UNION ALL
        SELECT
            project_id,
            SUM(cost),
            'Usage' AS cost_type,
            'AWS' AS source
        FROM
            payer_support
        GROUP BY
            cost_type,
            source,
            project_id
        UNION ALL
        SELECT
            project_id,
            SUM(cost),
            cost_type,
            'AWS' AS source,
        FROM
            payer_src
        WHERE
            cost < 0
            AND cost_type NOT IN ('SavingsPlanNegation',
            'Refund',
            'Tax')
        GROUP BY
            cost_type,
            source,
            project_id
        UNION ALL
        SELECT
            project_id,
            SUM(cost),
            cost_type,
            'AWS' AS source
        FROM
            payer_src
        WHERE
            cost_type IN ("Tax", "Refund")
        GROUP BY
            cost_type,
            source,
            project_id ),
        -- Union between both sources
        res AS (
        SELECT
            *
        FROM
            aws_invoice
        UNION ALL
        SELECT
            *
        FROM
            doit_invoice )
        SELECT
            project_id AS account_id,
			source,
			ARRAY_AGG(STRUCT(cost_type,
				cost)) AS cost_breakdown
        FROM
        	res
        WHERE
        	cost_type != "Tax"
        GROUP BY
			1,
			2
		ORDER BY
			1,
			2`

	return baseQuery
}
