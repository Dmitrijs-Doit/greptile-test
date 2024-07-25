package externalreport

const (
	ErrMsgFormat = "%s: %s"

	ErrConfigFilterRequiresValuesOrRegexp  = "internal config filter requires 'values' or 'regexp'"
	ErrConfigFilterIsFilterForLimits       = "internal config filter is used for limits"
	ErrInvalidConfigFilterType             = "invalid config filter type"
	ErrMetricTypeNotSupported              = "metric type not supported"
	ErrBasicMetricValue                    = "invalid basic metric value"
	ErrInvalidMetricType                   = "invalid metric type"
	ErrInvalidMetric                       = "invalid metric"
	ErrInvalidNumberOfValues               = "invalid number of values"
	ErrInvalidReport                       = "invalid report"
	ErrInvalidAdvancedAnalysis             = "invalid advanced analysis"
	ErrInvalidSplitType                    = "invalid split type"
	ErrInvalidTargetTotalType              = "invalid target total for custom mode"
	ErrInvalidCustomTimeRangeZero          = "invalid custom time range. Values cannot be zero"
	ErrInvalidCustomTimeRangeNegativeRange = "invalid custom time range. from cannot start after to"
	ErrInternal                            = "internal error"
	ErrInvalidSortGroupsValue              = "invalid sortGroups value"
	ErrInvalidSortDimensionsValue          = "invalid sortDimensions value"
	ErrInvalidDatasourceValue              = "invalid datasource value"
)
