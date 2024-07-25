package domain

var (
	ColumnTitles = map[string]string{
		"partitionsAvailable":        "Partition(s) to Remove",
		"userId":                     "User",
		"lastAccess":                 "Last Queried",
		"cost":                       "Storage (US$)",
		"storageSizeTB":              "Storage (TB)",
		"jobRef":                     "jobRef",
		"tableCreateDate":            "Table Create Date",
		"tableIdBaseName":            "Base Table ID",
		"projectId":                  "Project",
		"datasetId":                  "Dataset",
		"potentialClusteringFields":  "Cluster By",
		"potentialPartitionFields":   "Partition Fields",
		"tableId":                    "Table",
		"tableFullId":                "Table",
		"jobId":                      "Query ID",
		"firstExecution":             "First Execution Time",
		"lastExecution":              "Last Execution Time",
		"allJobs":                    "Jobs Executions",
		"scanTBperQuery":             "Scan TB Per Query",
		"reducingBy50":               "Savings By Reducing Jobs",
		"scanPricePerQuery":          "Scan Price Per Query",
		"totalScanPrice":             "Total Scan Price",
		"totalScanTB":                "Total Scan TB",
		"reducingBy40":               "Savings By Reducing Jobs",
		"reducingBy30":               "Savings By Reducing Jobs",
		"reducingBy20":               "Savings By Reducing Jobs",
		"reducingBy10":               "Savings By Reducing Jobs",
		"scanTB":                     "Scan (TB)",
		"scanPrice":                  "Scan (US$)",
		"partitionType":              "Partition Type",
		"partitionField":             "Partition Field",
		"potentialSavings":           "Savings Potential",
		"avgSlotMs":                  "Average Slots Ms",
		"scheduledQueryStart":        "Scheduled Query Start",
		"recommendedQueryStart":      "Recommended Query Start",
		"maxAvgDailySlots":           "Maximum average daily slots usage",
		"totalSpend":                 "Total spend on analysis",
		"observationStart":           "Observation start",
		"observationEnd":             "Observation end",
		"maxSlotsUsed":               "Maximum slots usage",
		"optimalSlotsMonthlyCost":    "Optimal slots monthly charge",
		"optimalSlotsRecommendation": "Optimal slots amount",
		"scheduledTime":              "Scheduled time",
		"slots":                      "Average Slots Used",
		"totalLogicalGB":             "Logical storage",
		"totalPhysicalGB":            "Physical storage",
		"totalLogicalCost":           "Logical cost",
		"totalPhysicalCost":          "Physical cost",
		"compressionRatio":           "Compression ratio",
		"savings":                    "Potential Savings",
	}

	ColumnsSigns = map[string]string{
		"cost":                    "$",
		"storageSizeTB":           "TB",
		"scanTBperQuery":          "TB",
		"reducingBy50":            "$",
		"scanPricePerQuery":       "$",
		"totalScanPrice":          "$",
		"totalScanTB":             "TB",
		"reducingBy40":            "$",
		"reducingBy30":            "$",
		"reducingBy20":            "$",
		"reducingBy10":            "$",
		"scanTB":                  "TB",
		"scanPrice":               "$",
		"potentialSavings":        "$",
		"optimalSlotsMonthlyCost": "$",
		"totalSpend":              "$",
		"totalLogicalGB":          "GB",
		"totalPhysicalGB":         "GB",
		"totalLogicalCost":        "$",
		"totalPhysicalCost":       "$",
		"savings":                 "$",
	}

	ColumnVisibility = map[string]bool{
		"userId":           true,
		"tableCreateDate":  true,
		"billingProjectId": true,
		"location":         true,
		"compressionRatio": true,
	}
)