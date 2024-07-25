package reportvalidator

const (
	ErrMsgFormat = "%s: %s"

	ErrInvalidLimitTopBottom                = "filter id is not listed in the rows field"
	ErrInvalidPromotionalCreditTimeInterval = "includePromotionalCredits requires monthly or greater time interval resolution"
	ErrInvalidTreemapsAggregator            = "treemaps renderer can only be used with aggregator set to total"
	ErrInvalidTreemapsFeatures              = "treemaps renderer cannot be used with trends or forecast features"
	ErrInvalidTreemapsDimension             = "treemaps renderer cannot be used with dimension"
	ErrInvalidTreemapsDisplayValues         = "treemaps renderer cannot be used with comparative mode"
	ErrInvalidCustomMetricAttribution       = "custom metric must filter attribution for"
	ErrInvalidLimitByCustomMetric           = "can only limit by a custom metric if the metric itself is selected as the report metric"
	ErrInvalidComparativeAggregation        = "comparative mode must use aggregation 'total'"
	ErrInvalidComparativeForecast           = "comparative mode must not use 'forecast'"
	ErrInvalidComparativeRenderer           = "displayValues: 'absolute_and_percentage' is only compatible with table layouts [table, table_heatmap]"
	ErrInvalidComparativeTimeSeries         = "comparative mode requires a valid time series dimension configuration"
	ErrInvalidComparativeSort               = "comparative mode requires alphabetical sorting"
	ErrInvalidSplit                         = "split element must be present in group"
	ErrInvalidCustomTimeRangeModeNotSet     = "when using custom time range, timeSettings mode must be set to 'custom'"
)
