package domain

const (
	StageAppendToTempCSPBillingAccountTableSetTaskState                       = "AppendToTempCSPBillingAccountTable:setTaskState"
	StageAppendToTempCSPBillingAccountTableDeleteCSPBillingAccount            = "AppendToTempCSPBillingAccountTable:deleteCSPBillingAccount"
	StageAppendToTempCSPBillingAccountTableAppendToTempCSPBillingAccountTable = "AppendToTempCSPBillingAccountTable:appendToTempCSPBillingAccountTable"

	StageUpdateCSPTableAndDeleteTempBigQueryTableExists              = "UpdateCSPTableAndDeleteTemp:BigQueryTableExists"
	StageUpdateCSPTableAndDeleteTempDeleteCSPBillingAccountFromTable = "UpdateCSPTableAndDeleteTemp:deleteCSPBillingAccountFromTable"
	StageUpdateCSPTableAndDeleteTempDstTableDelete                   = "UpdateCSPTableAndDeleteTemp:dstTable.Delete"
	StageUpdateCSPTableAndDeleteTempJoinAllCSPTempTables             = "UpdateCSPTableAndDeleteTemp:joinAllCSPTempTables"

	StageJoinCSPTempTableAddRemoveToCopiedTables      = "JoinCSPTempTable:AddRemoveToCopiedTables"
	StageJoinCSPTempTableJoinCSPTempTable             = "JoinCSPTempTable:joinCSPTempTable"
	StageJoinCSPTempTableCreateCSPAggregatedTableTask = "JoinCSPTempTable:createCSPAggregatedTableTask"

	StageUpdateCSPAggregatedTable = "UpdateCSPAggregatedTable"
)
