package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	budgets "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	domainHighCharts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/domain"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type HighchartsDataSeries struct {
	Data         []float64 `json:"data"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Color        string    `json:"color"`
	DashStyle    string    `json:"dashStyle"`
	ShowInLegend bool      `json:"showInLegend"`
	Marker       struct {
		Enabled bool `json:"enabled"`
	} `json:"marker"`
	total float64
}

type ByData []*HighchartsDataSeries

func (a ByData) Len() int           { return len(a) }
func (a ByData) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByData) Less(i, j int) bool { return a[i].Data[0] < a[j].Data[0] }

func getColors() []string {
	return []string{
		"#B9BCE7",
		"#EC6584",
		"#3C40AE",
		"#F6C6C3",
		"#FFF5C2",
		"#707DDC",
		"#D44666",
		"#DCDDE7",
		"#F2A8B2",
		"#4E5ECC",
		"#FAE3D4",
		"#959DE4",
		"#EF899D",
	}
}

func getReportCurrency(r *report.Report) string {
	if r.Config.Metric == report.MetricUsage {
		return ""
	}

	return r.Config.Currency.Symbol()
}

func (s *Highcharts) GetHighchartsRequestBudget(utilization string, budget *budgets.Budget, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) *domainHighCharts.HighchartsRequest {
	highchartsFontSettings.FontSize = "0.5rem"
	amount := budget.Config.Amount
	alerts := budget.Config.Alerts

	var y float64

	title := domainHighCharts.HighchartsDataTitle{
		Text:  "",
		Style: s.GetStyle(highchartsFontSettings),
	}

	if utilization == "current" {
		y = budget.Utilization.Current
		title.Text = "Current Spend"
	} else {
		y = budget.Utilization.Forecasted
		title.Text = "Forecasted Spend"
	}

	xAxis := domainHighCharts.HighchartsDataXAxis{
		Categories: []string{""},
		Labels: domainHighCharts.HighchartsDataAxisLabels{
			Align:        "left",
			ReserveSpace: common.Bool(true),
			UseHTML:      common.Bool(true),
			Style:        s.GetStyle(highchartsFontSettings),
		},
	}
	series := []domainHighCharts.HighchartsBulletSeries{{
		Type: "bullet",
		Data: []*domainHighCharts.BulletDataNode{{
			Color:  "#64B5F6",
			Target: common.Float(amount),
			Y:      y,
		}},
	}}
	yAxis := domainHighCharts.HighchartsDataYAxis{
		EndOnTick:     common.Bool(false),
		MaxPadding:    common.Int(0),
		GridLineWidth: common.Int(0),
		Labels:        s.GetLabels(highchartsFontSettings),
		PlotBands: []domainHighCharts.BulletPlotBand{
			{
				From:  0,
				To:    (alerts[0].Percentage * amount) / 100,
				Color: "#a5d6a7",
			},
			{
				From:  (alerts[0].Percentage * amount) / 100,
				To:    (alerts[1].Percentage * amount) / 100,
				Color: "#fff59d",
			},
			{
				From:  (alerts[1].Percentage * amount) / 100,
				To:    (alerts[2].Percentage * amount) / 100,
				Color: "#ef9a9a",
			},
		},
	}
	chart := domainHighCharts.HighchartsDataChart{
		Inverted:        common.Bool(true),
		MarginLeft:      common.Int(0),
		SpacingLeft:     common.Int(0),
		MarginTop:       20,
		MarginBottom:    common.Int(24),
		BackgroundColor: highchartsFontSettings.ChartBackgroundColor,
		Type:            domainHighCharts.HighchartsTypeBullet,
	}

	infile := domainHighCharts.HighchartsData{
		Title:       title,
		Series:      series,
		XAxis:       xAxis,
		Chart:       chart,
		YAxis:       yAxis,
		PlotOptions: getBulletPlotOptions(),
	}
	hcr := &domainHighCharts.HighchartsRequest{
		Infile:         &infile,
		Scale:          4,
		AsyncRendering: false,
		Width:          400,
		Height:         70,
		Callback:       getBudgetCallback(budget),
	}

	return hcr
}

// shouldReverseYAxis - if all data points in series are negative, yaxis should be reversed
func shouldReverseYAxis(series interface{}) (bool, error) {
	dataSeries, ok := series.([]*HighchartsDataSeries)
	if !ok {
		return false, errors.New("error casting sereis to HighchartsDataSeries")
	}

	if len(dataSeries) == 0 {
		return false, nil
	}

	totals := make([]float64, len(dataSeries[0].Data))

	for _, seriesItem := range dataSeries {
		for i, dataPoint := range seriesItem.Data {
			totals[i] += dataPoint
		}
	}

	for _, total := range totals {
		if total > 0 {
			return false, nil
		}
	}

	return true, nil
}

func (s *Highcharts) GetHighchartsRequestReport(ctx context.Context, reportQueryRequest *cloudanalytics.QueryRequest, reportQueryResult *cloudanalytics.QueryResult, r *report.Report, isTreemapExact bool, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) (*domainHighCharts.HighchartsRequest, error) {
	if reportQueryRequest.Comparative != nil {
		metricNumber := reportQueryRequest.GetMetricIndex() + 1
		if r.Config.CalculatedMetric != nil {
			metricNumber = int(report.MetricCustom)
		}

		rows, err := getComparativeRows(reportQueryResult.Rows, len(reportQueryRequest.Rows), len(reportQueryRequest.Cols), metricNumber, reportQueryRequest.Comparative)
		if err != nil {
			return nil, err
		}

		reportQueryResult.Rows = rows
	}

	xAxis, err := s.getXAxis(ctx, reportQueryRequest, reportQueryResult, r, highchartsFontSettings)
	if err != nil {
		return nil, err
	}

	reversed := false

	var series interface{}

	if r.Config.Renderer == report.RendererTreemapChart {
		var err error

		series, err = getTreemapSeries(reportQueryRequest, reportQueryResult, r, isTreemapExact)
		if err != nil {
			return nil, err
		}
	} else {
		var err error

		series, err = getSeries(reportQueryRequest, reportQueryResult, xAxis.Categories, r)
		if err != nil {
			return nil, err
		}

		reversed, err = shouldReverseYAxis(series)
		if err != nil {
			return nil, err
		}
	}

	yAxis := domainHighCharts.HighchartsDataYAxis{
		GridLineColor: highchartsFontSettings.GridLineColor,
		Reversed:      reversed,
		Labels:        s.GetLabels(highchartsFontSettings),
		Title: domainHighCharts.HighchartsDataYAxisTitle{
			Enabled: false,
		},
	}
	if reportQueryRequest.Comparative != nil {
		yAxis.StackLabels = domainHighCharts.HighchartsDataYAxisStackLabels{
			Enabled:      true,
			AllowOverlap: false,
			Style:        s.GetStyle(highchartsFontSettings),
			Formatter:    getCallback(reportQueryRequest, r, false),
		}
	}

	if reportQueryRequest.LogScale {
		yAxis.Type = common.String("logarithmic")
	}

	chart := domainHighCharts.HighchartsDataChart{
		BackgroundColor: highchartsFontSettings.ChartBackgroundColor,
		Width:           1200,
		MarginTop:       20,
		ResetZoomButton: struct {
			Position string `json:"position"`
		}{
			Position: "left",
		},
	}
	legendReverse := true
	infile := domainHighCharts.HighchartsData{
		Series: series,
		XAxis:  *xAxis,
		Colors: getColors(),
		Chart:  chart,
		Credits: struct {
			Enabled bool `json:"enabled"`
		}{
			Enabled: false,
		},
		Boost: struct {
			Enabled bool `json:"enabled"`
		}{
			Enabled: false,
		},
		Title: domainHighCharts.HighchartsDataTitle{},
		YAxis: yAxis,
		Legend: domainHighCharts.HighchartsDataLegend{
			Enabled: true,
			ItemStyle: domainHighCharts.HighchartsLegendItemStyle{
				Color:        highchartsFontSettings.LegendFontColor,
				FontSize:     highchartsFontSettings.LegendFontSize,
				FontWeight:   "400",
				TextOverflow: "ellipsis",
			},
			Reversed: &legendReverse,
		},
	}
	hcr := &domainHighCharts.HighchartsRequest{
		Infile:         &infile,
		Scale:          2,
		AsyncRendering: false,
	}

	s.setRendererOptions(hcr, reportQueryRequest, r, highchartsFontSettings)

	return hcr, nil
}

func getBulletPlotOptions() domainHighCharts.BulletPlotOptions {
	return domainHighCharts.BulletPlotOptions{
		Series: &domainHighCharts.BulletPlotOptionsSeries{
			PointPadding: 0.25,
			BorderWidth:  0,
			Color:        "#000",
			TargetOptions: struct {
				Width string `json:"width"`
			}{
				Width: "100%",
			},
			Animation: false,
		},
	}
}

func getStackedColumnPlotOptions() domainHighCharts.StackedColumnPlotOptions {
	return domainHighCharts.StackedColumnPlotOptions{
		Column: struct {
			Stacking      string  `json:"stacking"`
			BorderWidth   int     `json:"borderWidth"`
			MaxPointWidth int     `json:"maxPointWidth"`
			GroupPadding  float64 `json:"groupPadding"`
			PointPadding  float64 `json:"pointPadding"`
			DataLabels    struct {
				Enabled bool `json:"enabled"`
			} `json:"dataLabels"`
		}{
			Stacking:      "normal",
			BorderWidth:   0,
			MaxPointWidth: 50,
			GroupPadding:  0.05,
			PointPadding:  0.025,
			DataLabels: struct {
				Enabled bool `json:"enabled"`
			}{Enabled: false},
		},
	}
}

func getColumnPlotOptions() domainHighCharts.ColumnPlotOptions {
	return domainHighCharts.ColumnPlotOptions{
		Column: struct {
			BorderWidth   int     `json:"borderWidth"`
			MaxPointWidth int     `json:"maxPointWidth"`
			GroupPadding  float64 `json:"groupPadding"`
			PointPadding  float64 `json:"pointPadding"`
			DataLabels    struct {
				Enabled bool `json:"enabled"`
			} `json:"dataLabels"`
		}{
			BorderWidth:   0,
			MaxPointWidth: 50,
			GroupPadding:  0.05,
			PointPadding:  0.025,
			DataLabels: struct {
				Enabled bool `json:"enabled"`
			}{Enabled: false},
		},
	}
}

func getBarPlotOptions() domainHighCharts.BarPlotOptions {
	return domainHighCharts.BarPlotOptions{
		Bar: struct {
			BorderWidth   int     `json:"borderWidth"`
			MaxPointWidth int     `json:"maxPointWidth"`
			GroupPadding  float64 `json:"groupPadding"`
			PointPadding  float64 `json:"pointPadding"`
			DataLabels    struct {
				Enabled bool `json:"enabled"`
			} `json:"dataLabels"`
		}{
			BorderWidth:   0,
			MaxPointWidth: 50,
			GroupPadding:  0.05,
			PointPadding:  0.025,
			DataLabels: struct {
				Enabled bool `json:"enabled"`
			}{Enabled: false},
		},
	}
}

func getStackedBarPlotOptions() domainHighCharts.StackedBarPlotOptions {
	return domainHighCharts.StackedBarPlotOptions{
		Bar: struct {
			Stacking      string  `json:"stacking"`
			BorderWidth   int     `json:"borderWidth"`
			MaxPointWidth int     `json:"maxPointWidth"`
			GroupPadding  float64 `json:"groupPadding"`
			PointPadding  float64 `json:"pointPadding"`
			DataLabels    struct {
				Enabled bool `json:"enabled"`
			} `json:"dataLabels"`
		}{
			Stacking:      "normal",
			BorderWidth:   0,
			MaxPointWidth: 50,
			GroupPadding:  0.05,
			PointPadding:  0.025,
			DataLabels: struct {
				Enabled bool `json:"enabled"`
			}{Enabled: false},
		},
	}
}

func isCalculatedMetricUsageBased(qr *cloudanalytics.QueryRequest) bool {
	if qr.CalculatedMetric != nil && qr.CalculatedMetric.Format != metrics.MetricPercentageFormat {
		for _, v := range qr.CalculatedMetric.Variables {
			if v.Metric == report.MetricUsage {
				return true
			}
		}
	}

	return false
}

func getCurrency(qr *cloudanalytics.QueryRequest, r *report.Report) string {
	if r.Config.Aggregator == report.AggregatorCount {
		return ""
	}

	if isCalculatedMetricUsageBased(qr) {
		return ""
	}

	return getReportCurrency(r)
}

func getTreemapCallback(qr *cloudanalytics.QueryRequest, r *report.Report) string {
	if qr.CalculatedMetric != nil && qr.CalculatedMetric.Format == metrics.MetricPercentageFormat {
		return `function(chart){
			chart.series[0].update({
				dataLabels:{
					style: {
						textAlign: "center",
						textDecoration: "none",
						textOutline: "none",
						fontWeight: "400"
					},
					formatter: function(){
						return this.point.name + ": " + Math.round(this.point.value * 100) + "%";
					}
				}
			})
		}`
	}

	currency := getCurrency(qr, r)

	return `function(chart){
		chart.series[0].update({
			dataLabels:{
				style: {
					textAlign: "center",
					textDecoration: "none",
					textOutline: "none",
					fontWeight: "400"
				},
				formatter: function(){
					if (this.point.realValue >= 1e18) {
						const num = Math.round(this.point.realValue/1e18 * 100) / 100;
						return this.point.name + ": ` + currency + `" + num + "E";
					}
					if (this.point.realValue >= 1e15) {
						const num = Math.round(this.point.realValue/1e15 * 100) / 100;
						return this.point.name + ": ` + currency + `" + num + "P";
					}
					if (this.point.realValue >= 1e12) {
						const num = Math.round(this.point.realValue/1e12 * 100) / 100;
						return this.point.name + ": ` + currency + `" + num + "T";
					}
					if (this.point.realValue >= 1e9) {
						const num = Math.round(this.point.realValue/1e9 * 100) / 100;
						return this.point.name + ": ` + currency + `" + num + "G";
					}
					if (this.point.realValue >= 1e6) {
						const num = Math.round(this.point.realValue/1e6 * 100) / 100;
						return this.point.name + ": ` + currency + `" + num + "M";
					}
					if (this.point.realValue >= 1e3) {
						const num = Math.round(this.point.realValue/1e3 * 100) / 100;
						return this.point.name + ": ` + currency + `" + num + "k";
					}
					return this.point.name + ": ` + currency + `" + Math.round(this.point.realValue * 10) / 10;
				}
			}
		})
	}`
}

func getCallback(qr *cloudanalytics.QueryRequest, r *report.Report, vertical bool) string {
	var locateLabel string
	if vertical {
		locateLabel = `rotation: -90,
					align: "center",
					textAlign: "center",
					y: 20,
					x: 5,`
	}

	if r.Config.Comparative != nil && *r.Config.Comparative == "percent" {
		return `function(chart) {
				chart.yAxis[0].update({
					stackLabels: {
						allowOverlap: false,
						formatter: function() {
							if (!this.total) {
								return "";
							}
							return Math.round(this.total * 100)/100 + "%";
						}
					},
					labels: {
						formatter: function() {
							return Math.round(this.value) + "%";
						}
					}
				}, true, true)
			}`
	}

	if shouldFormatToPercent(r.Config, qr) {
		return fmt.Sprintf(`function(chart) {
			chart.yAxis[0].update({
				stackLabels: {
					allowOverlap: false,
					%s
					formatter: function() {
						if (!this.total) {
							return "";
						}
						return Math.round(this.total * 100)/100 + "%%";
					}
				},
				labels: {
					formatter: function() {
						return Math.round(this.value * 100) + "%%";
					}
				}
			}, true, true)
		}`, locateLabel)
	}

	currency := getCurrency(qr, r)

	return `function(chart) {
		chart.yAxis[0].update({
			stackLabels: {
				formatter: function() {
					if (!this.total) {
						return "";
					}
					const absTotal = Math.abs(this.total);
					const sign = this.total < 0 ? "-" : "";

					if (absTotal >= 1e18) {
						const num = Math.round(absTotal/1e18 * 10) / 10;
						return sign + "` + currency + `" + num + "E";
					}
					if (absTotal >= 1e15) {
						const num = Math.round(absTotal/1e15 * 10) / 10;
						return sign + "` + currency + `" + num + "P";
					}
					if (absTotal >= 1e12) {
						const num = Math.round(absTotal/1e12 * 10) / 10;
						return sign + "` + currency + `" + num + "T";
					}
					if (absTotal >= 1e9) {
						const num = Math.round(absTotal/1e9 * 10) / 10;
						return sign + "` + currency + `" + num + "G";
					}
					if (absTotal >= 1e6) {
						const num = Math.round(absTotal/1e6 * 10) / 10;
						return sign + "` + currency + `" + num + "M";
					}
					if (absTotal >= 1e3) {
						const num = Math.round(absTotal/1e3 * 10) / 10;
						return sign + "` + currency + `" + num + "k";

					}
					return sign + "` + currency + `" + Math.round(absTotal * 10) / 10;
				}
			},
			labels: {
				formatter: function() {
					const absVal = Math.abs(this.value);
					const sign = this.value < 0 ? "-" : "";

					if (absVal >= 1e18) {
						const num = Math.round(absVal/1e18 * 10) / 10;
						return sign + "` + currency + `" + num + "E";
					}
					if (absVal >= 1e15) {
						const num = Math.round(absVal/1e15 * 10) / 10;
						return sign + "` + currency + `" + num + "P";
					}
					if (absVal >= 1e12) {
						const num = Math.round(absVal/1e12 * 10) / 10;
						return sign + "` + currency + `" + num + "T";
					}
					if (absVal >= 1e9) {
						const num = Math.round(absVal/1e9 * 10) / 10;
						return sign + "` + currency + `" + num + "G";
					}
					if (absVal >= 1e6) {
						const num = Math.round(this.value/1e6 * 10) / 10;
						return sign + "` + currency + `" + num + "M";
					}
					if (absVal >= 1e3) {
						const num = Math.round(absVal/1e3 * 10) / 10;
						return sign + "` + currency + `" + num + "k";
					}
					return sign + "` + currency + `" + Math.round(absVal * 10) / 10;
				}
			}

		})
		chart.xAxis[0].update({
			stackLabels: {
				formatter: function() {
					if (!this.total) {
						return "";
					}
					if (this.total >= 1e18) {
						const num = Math.round(this.total/1e18 * 10) / 10;
						return "` + currency + `" + num + "E";
					}
					if (this.total >= 1e15) {
						const num = Math.round(this.total/1e15 * 10) / 10;
						return "` + currency + `" + num + "P";
					}
					if (this.total >= 1e12) {
						const num = Math.round(this.total/1e12 * 10) / 10;
						return "` + currency + `" + num + "T";
					}
					if (this.total >= 1e9) {
						const num = Math.round(this.total/1e9 * 10) / 10;
						return "` + currency + `" + num + "G";
					}
					if (this.total >= 1e6) {
						const num = Math.round(this.total/1e6 * 10) / 10;
						return "` + currency + `" + num + "M";
					}
					if (this.total >= 1e3) {
						const num = Math.round(this.total/1e3 * 10) / 10;
						return "` + currency + `" + num + "k";
					}
					return "` + currency + `" + Math.round(this.total * 10) / 10;
				}
			}
		})
	}`
}

func getYAxisCurrencyLabels(currency string) string {
	return `function(chart) {
		chart.yAxis[0].update({
			labels: {
				formatter: function() {
					if (this.value >= 1e18) {
						const num = Math.round(this.value/1e18 * 10) / 10;
						return "` + currency + `" + num + "E";
					}
					if (this.value >= 1e15) {
						const num = Math.round(this.value/1e15 * 10) / 10;
						return "` + currency + `" + num + "P";
					}
					if (this.value >= 1e12) {
						const num = Math.round(this.value/1e12 * 10) / 10;
						return "` + currency + `" + num + "T";
					}
					if (this.value >= 1e9) {
						const num = Math.round(this.value/1e9 * 10) / 10;
						return "` + currency + `" + num + "G";
					}
					if (this.value >= 1e6) {
						const num = Math.round(this.value/1e6 * 10) / 10;
						return "` + currency + `" + num + "M";
					}
					if (this.value >= 1e3) {
						const num = Math.round(this.value/1e3 * 10) / 10;
						return "` + currency + `" + num + "k";
					}
					return "` + currency + `" + Math.round(this.value * 10) / 10;
				}
			}
		})
	}`
}

func getBudgetCallback(budget *budgets.Budget) string {
	return getYAxisCurrencyLabels(budget.Config.Currency.Symbol())
}

func getYAxisLabelsCallback(qr *cloudanalytics.QueryRequest, r *report.Report) string {
	if shouldFormatToPercent(r.Config, qr) {
		return `function(chart) {
			chart.yAxis[0].update({
				labels: {
					formatter: function() {
						return Math.round(this.value * 100) + "%";
					}
				}
			})
		}`
	}

	currency := getCurrency(qr, r)

	return getYAxisCurrencyLabels(currency)
}

func (s *Highcharts) setRendererOptions(hcr *domainHighCharts.HighchartsRequest, qr *cloudanalytics.QueryRequest, r *report.Report, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) {
	switch r.Config.Renderer {
	case report.RendererColumnChart:
		plotOptions := getColumnPlotOptions()
		hcr.Infile.Chart.Type = domainHighCharts.HighchartsTypeColumn
		hcr.Infile.PlotOptions = plotOptions
		hcr.Callback = getYAxisLabelsCallback(qr, r)
	case report.RendererStackedColumnChart:
		hcr.Infile.Chart.Type = domainHighCharts.HighchartsTypeColumn
		hcr.Infile.YAxis.StackLabels = domainHighCharts.HighchartsDataYAxisStackLabels{
			Enabled:      true,
			AllowOverlap: false,
			Style:        s.GetStyle(highchartsFontSettings),
		}
		plotOptions := getStackedColumnPlotOptions()
		hcr.Infile.PlotOptions = plotOptions
		hcr.Callback = getCallback(qr, r, true)
	case report.RendererBarChart:
		plotOptions := getBarPlotOptions()
		hcr.Infile.PlotOptions = plotOptions
		hcr.Infile.Chart.Type = domainHighCharts.HighchartsTypeBar
		hcr.Callback = getYAxisLabelsCallback(qr, r)
	case report.RendererStackedBaChart:
		plotOptions := getStackedBarPlotOptions()
		hcr.Infile.PlotOptions = plotOptions
		hcr.Infile.Chart.Type = domainHighCharts.HighchartsTypeBar
		hcr.Infile.YAxis.StackLabels = domainHighCharts.HighchartsDataYAxisStackLabels{
			Enabled:      true,
			AllowOverlap: false,
			Style:        s.GetStyle(highchartsFontSettings),
		}
		hcr.Callback = getCallback(qr, r, false)
	case report.RendererLineChart:
		hcr.Infile.Chart.Type = domainHighCharts.HighchartsTypeLine
		hcr.Infile.PlotOptions = struct{}{}
		hcr.Callback = getYAxisLabelsCallback(qr, r)
	case report.RendererSplineChart:
		hcr.Infile.Chart.Type = domainHighCharts.HighchartsTypeSpline
		hcr.Infile.PlotOptions = struct{}{}
		hcr.Callback = getYAxisLabelsCallback(qr, r)
	case report.RendererStackedAreaChart:
		hcr.Infile.Chart.Type = domainHighCharts.HighchartsTypeArea
		hcr.Infile.PlotOptions = struct {
			Area struct {
				Stacking string `json:"stacking"`
			} `json:"area"`
		}{Area: struct {
			Stacking string `json:"stacking"`
		}{Stacking: "normal"}}
		hcr.Infile.YAxis.StackLabels = domainHighCharts.HighchartsDataYAxisStackLabels{
			Enabled:      true,
			AllowOverlap: false,
			Style:        s.GetStyle(highchartsFontSettings),
		}
		hcr.Callback = getCallback(qr, r, true)
	case report.RendererAreaChart:
		hcr.Infile.Chart.Type = domainHighCharts.HighchartsTypeArea
		hcr.Infile.PlotOptions = struct{}{}
		hcr.Callback = getYAxisLabelsCallback(qr, r)
	case report.RendererAreaSplineChart:
		hcr.Infile.Chart.Type = domainHighCharts.HighchartsTypeAreaSpline
		hcr.Infile.PlotOptions = struct{}{}
		hcr.Callback = getYAxisLabelsCallback(qr, r)
	case report.RendererTreemapChart:
		hcr.Infile.Chart.Type = domainHighCharts.HighchartsTypeTreemap
		hcr.Infile.PlotOptions = struct{}{}
		hcr.Callback = getTreemapCallback(qr, r)

	default:
		// All unsupported renderers default to stacked column chart
		hcr.Infile.Chart.Type = domainHighCharts.HighchartsTypeColumn
		hcr.Infile.YAxis.StackLabels = domainHighCharts.HighchartsDataYAxisStackLabels{
			Enabled:      true,
			AllowOverlap: false,
			Style:        s.GetStyle(highchartsFontSettings),
		}
		plotOptions := getStackedColumnPlotOptions()
		hcr.Infile.PlotOptions = plotOptions
		hcr.Callback = getCallback(qr, r, true)
	}
}

func getComparativeHeader(comparative *string, row []bigquery.Value, diffHeaderIndex int) string {
	comparativeSign := "∆"
	label := row[diffHeaderIndex]
	kvs := map[string]string{"percent": `%s %s%%`, "values": "%s %s", "both": `%s %s%%`}

	if template, ok := kvs[*comparative]; ok {
		return fmt.Sprintf(template, label, comparativeSign)
	}

	header, ok := label.(string)
	if !ok {
		return ""
	}

	return header
}

func getComparativeRecord(row []bigquery.Value, diffHeaderIndex int, diffValueIndex int, replaceIndex int, comparative *string) []bigquery.Value {
	newRecord := row
	comparativeHeader := getComparativeHeader(comparative, row, diffHeaderIndex)
	newRecord[diffHeaderIndex] = comparativeHeader

	var v float64

	diffArr, ok := row[diffValueIndex].(cloudanalytics.ComparativeColumnValue)
	if !ok {
		diffArr = cloudanalytics.ComparativeColumnValue{Pct: 0, Val: 0}
	}

	switch *comparative {
	case "percent":
		v, ok = diffArr.Pct.(float64)
		if !ok {
			v = 0
		}
	case "values":
		v, ok = diffArr.Val.(float64)
		if !ok {
			v = 0
		}
	default:
		v = 0
	}

	newRecord[replaceIndex] = v

	return newRecord
}

func getComparativeRows(rows [][]bigquery.Value, rowsLen int, colsLen int, metricNumber int, comparative *string) ([][]bigquery.Value, error) {
	var newRecords = make([][]bigquery.Value, 0)

	diffHeaderIndex := rowsLen + colsLen - 1
	replaceValueIndex := diffHeaderIndex + metricNumber

	numMetrics := int(report.MetricEnumLength)
	if metricNumber == 4 {
		numMetrics = metricNumber
	}

	diffValueIndex := replaceValueIndex + numMetrics

	for i, row := range rows {
		recordRowsKey, err := query.GetRowKey(row, rowsLen)
		if err != nil {
			recordRowsKey = "<nil>"
		}

		if i > 0 {
			prevRow := rows[i-1]

			prevRecordRowKey, err := query.GetRowKey(prevRow, rowsLen)
			if err != nil {
				prevRecordRowKey = "<nil>"
			}

			if prevRecordRowKey == recordRowsKey {
				newRecord := getComparativeRecord(row, diffHeaderIndex, diffValueIndex, replaceValueIndex, comparative)
				newRecords = append(newRecords, newRecord)
			}
		}
	}

	return newRecords, nil
}

func (s *Highcharts) getXAxis(ctx context.Context, reportQueryRequest *cloudanalytics.QueryRequest, reportQueryResult *cloudanalytics.QueryResult, r *report.Report, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) (*domainHighCharts.HighchartsDataXAxis, error) {
	rows := reportQueryResult.Rows

	categories, err := getSortedCategories(len(reportQueryRequest.Rows), reportQueryRequest.Cols, reportQueryRequest.Metric, reportQueryResult.Rows, r.Config.ColOrder)
	if err != nil {
		s.loggerProvider(ctx).Errorf("Error getting sorted categories for report id %s: %v", r.ID, err)
		return nil, err
	}

	labels := s.GetLabels(highchartsFontSettings)
	x := &domainHighCharts.HighchartsDataXAxis{
		Crosshair: true,
		Labels:    labels,
	}

	if reportQueryResult.ForecastRows != nil {
		rows = reportQueryResult.ForecastRows
		for i := len(categories); i < len(rows); i++ {
			category, err := getCategory(1, reportQueryResult.ForecastRows[i], reportQueryRequest.Cols)
			if err != nil {
				return nil, err
			}

			categories = append(categories, category)
		}
	}

	x.Categories = categories

	return x, nil
}

func getSortedCategories(requestRows int, requestCols []*domainQuery.QueryRequestX, requestMetric report.Metric, resultRows [][]bigquery.Value, reportColOrder string) ([]string, error) {
	if reportColOrder != "a_to_z" {
		return getSortedCategoriesByTotal(requestRows, requestCols, requestMetric, resultRows, reportColOrder)
	}

	categoryKeys, err := getCategoryKeys(requestRows, resultRows, requestCols)
	if err != nil {
		return nil, err
	}

	sortCategoryKeysAtoZ(categoryKeys, requestCols)

	categories := make([]string, len(categoryKeys))
	for i, cs := range categoryKeys {
		categories[i] = strings.Join(cs, "-")
	}

	return categories, nil
}

func sortCategoryKeysAtoZ(categoryKeys [][]string, requestCols []*domainQuery.QueryRequestX) [][]string {
	colIndexToBaseValueMappingFuncMap := make(map[int]func(string) string)

	for i, col := range requestCols {
		mapperFunc := domainQuery.KeyMap[col.Key].BaseValueMappingFunc
		if mapperFunc != nil {
			colIndexToBaseValueMappingFuncMap[i] = mapperFunc
		}
	}

	sort.Slice(categoryKeys, func(categoryI, categoryJ int) bool {
		for keyIndex := range categoryKeys[categoryI] {
			firstKey := categoryKeys[categoryI][keyIndex]
			secondKey := categoryKeys[categoryJ][keyIndex]

			if mapperFunc, ok := colIndexToBaseValueMappingFuncMap[keyIndex]; ok {
				firstKey = mapperFunc(firstKey)
				secondKey = mapperFunc(secondKey)
			}

			if firstKey == secondKey {
				continue
			}

			return firstKey < secondKey
		}

		return false
	})

	return categoryKeys
}

func getCategoryKeys(colsFirstIndex int, resultRows [][]bigquery.Value, requestCols []*domainQuery.QueryRequestX) ([][]string, error) {
	uniqueCategoryKeys := make(map[string]bool)
	categoryKeys := make([][]string, 0)

	for _, row := range resultRows {
		categorySlice, err := getCategoryAsSlice(colsFirstIndex, row, requestCols)
		if err != nil {
			return nil, err
		}

		key := strings.Join(categorySlice, "-")
		if _, ok := uniqueCategoryKeys[key]; !ok {
			uniqueCategoryKeys[key] = true

			categoryKeys = append(categoryKeys, categorySlice)
		}
	}

	return categoryKeys, nil
}

func getCategory(colsFirstIndex int, row []bigquery.Value, queryRequestCols []*domainQuery.QueryRequestX) (string, error) {
	slice, err := getCategoryAsSlice(colsFirstIndex, row, queryRequestCols)
	if err != nil {
		return "", err
	}

	return strings.Join(slice, "-"), nil
}

func getCategoryAsSlice(colsFirstIndex int, row []bigquery.Value, queryRequestCols []*domainQuery.QueryRequestX) ([]string, error) {
	colLen := len(queryRequestCols)
	categoryArr := make([]string, 0)

	for colIndex := 0; colIndex < colLen; colIndex++ {
		i := colsFirstIndex + colIndex
		if row[i] != nil {
			if keyStr, err := query.BigqueryValueToString(row[i]); err != nil {
				return nil, err
			} else {
				categoryArr = append(categoryArr, keyStr)
			}
		} else {
			nullFallback := getNullFallback(queryRequestCols[colIndex].ID)
			categoryArr = append(categoryArr, *nullFallback)
		}
	}

	return categoryArr, nil
}

func getCategoryIndex(category string, categories []string) int {
	for i, c := range categories {
		if c == category {
			return i
		}
	}

	return -1
}

func getColTotal(i int, series []*HighchartsDataSeries) float64 {
	total := 0.0
	for _, hcds := range series {
		total += hcds.Data[i]
	}

	return total
}

func calculatePercentOfColumn(i int, series []*HighchartsDataSeries) {
	colTotal := getColTotal(i, series)
	for _, hcds := range series {
		hcds.Data[i] = hcds.Data[i] / colTotal
	}
}

func calculatePercentOfTotal(categories []string, series []*HighchartsDataSeries) {
	generalTotal := 0.0

	for i := range categories {
		colTotal := getColTotal(i, series)
		generalTotal += colTotal
	}

	for i := range categories {
		for _, hcds := range series {
			hcds.Data[i] = hcds.Data[i] / generalTotal
		}
	}
}

func calculatePercentOfRow(series []*HighchartsDataSeries) {
	for _, hcds := range series {
		rowTotal := 0.0
		for _, d := range hcds.Data {
			rowTotal += d
		}

		for i, d := range hcds.Data {
			hcds.Data[i] = d / rowTotal
		}
	}
}

func getTreeMapDataNodeID(toIndex, resultRowIndex int, reportQueryResult *cloudanalytics.QueryResult, reportQueryRequest *cloudanalytics.QueryRequest) (string, error) {
	idArray := make([]string, 0)

	for i := 0; i <= toIndex; i++ {
		row := reportQueryResult.Rows[resultRowIndex]
		if row[i] != nil {
			if keyStr, err := query.BigqueryValueToString(row[i]); err != nil {
				return "", err
			} else {
				idArray = append(idArray, keyStr)
			}
		} else {
			nullFallback := getNullFallback(reportQueryRequest.Rows[i].ID)
			idArray = append(idArray, *nullFallback)
		}
	}

	santizedIDArray := make([]string, 0)

	for _, el := range idArray {
		if el == "" {
			santizedIDArray = append(santizedIDArray, "[N/A]")
		} else {
			santizedIDArray = append(santizedIDArray, el)
		}
	}

	id := strings.Join(santizedIDArray, "_")

	return id, nil
}

func isIDExistInTreemapDataSeries(data []*domainHighCharts.TreemapDataNode, newID string) bool {
	for _, node := range data {
		if node.ID == newID {
			return true
		}
	}

	return false
}

func getParentID(ID string) string {
	idArray := strings.Split(ID, "_")
	idArray = idArray[:len(idArray)-1]

	if len(idArray) == 0 {
		return ""
	}

	return strings.Join(idArray, "_")
}

func getTreemapNodeName(ID string) string {
	idArray := strings.Split(ID, "_")
	return idArray[len(idArray)-1]
}

func addValueToDataNode(data []*domainHighCharts.TreemapDataNode, id string, value float64, isTreemapExact bool) {
	for _, node := range data {
		if node.ID == id {
			node.RealValue += value
			getTreemapValue(value, isTreemapExact)
		}
	}
}

func getTreemapSeries(reportQueryRequest *cloudanalytics.QueryRequest, reportQueryResult *cloudanalytics.QueryResult, r *report.Report, isTreemapExact bool) ([]*domainHighCharts.HighchartsTreemapSeries, error) {
	series := make([]*domainHighCharts.HighchartsTreemapSeries, 0)
	levels := make([]domainHighCharts.TreemapSeriesLevel, 1)
	levels[0] = domainHighCharts.TreemapSeriesLevel{
		Level:        1,
		ColorByPoint: true,
		BorderWidth:  2,
		DataLabels: struct {
			Enabled bool `json:"enabled"`
		}{Enabled: true},
	}
	s := domainHighCharts.HighchartsTreemapSeries{
		LayoutAlgorithm: "squarified",
		Type:            "treemap",
		Opacity:         0.8,
		Levels:          levels,
		DataLabels: domainHighCharts.TreemapSeriesDataLabels{
			Enabled: false,
		},
	}
	metricNumber := reportQueryRequest.GetMetricIndex()
	metricIndex := len(r.Config.Rows) + metricNumber

	for toIndex := range r.Config.Rows {
		for resultRowIndex := range reportQueryResult.Rows {
			id, err := getTreeMapDataNodeID(toIndex, resultRowIndex, reportQueryResult, reportQueryRequest)
			if err != nil {
				return nil, err
			}

			value, ok := reportQueryResult.Rows[resultRowIndex][metricIndex].(float64)
			if !ok {
				value = 0.0
			}

			if !isIDExistInTreemapDataSeries(s.Data, id) {
				dataNode := domainHighCharts.TreemapDataNode{
					ID:        id,
					Parent:    getParentID(id),
					RealValue: value,
					Value:     getTreemapValue(value, isTreemapExact),
					Name:      getTreemapNodeName(id),
				}
				s.Data = append(s.Data, &dataNode)
			} else {
				addValueToDataNode(s.Data, id, value, isTreemapExact)
			}
		}
	}

	filteredS := make([]*domainHighCharts.TreemapDataNode, 0)

	for _, node := range s.Data {
		if node.Parent == "" {
			filteredS = append(filteredS, node)
		}
	}

	s.Data = filteredS
	series = append(series, &s)

	return series, nil
}

func getSeries(reportQueryRequest *cloudanalytics.QueryRequest, reportQueryResult *cloudanalytics.QueryResult, categories []string, r *report.Report) ([]*HighchartsDataSeries, error) {
	series := make([]*HighchartsDataSeries, 0)
	metricFirstIndex := len(reportQueryRequest.Rows) + len(reportQueryRequest.Cols)
	metricNumber := reportQueryRequest.GetMetricIndex()
	selectedMetric := metricFirstIndex + metricNumber
	// The index for the trend feature that matches the selected metric
	trendIndex := selectedMetric + int(report.MetricEnumLength)
	usingTrendFeatures := len(reportQueryRequest.Trends) > 0 &&
		len(reportQueryResult.Rows) > 0 &&
		trendIndex < len(reportQueryResult.Rows[0])

	for _, row := range reportQueryResult.Rows {
		name, err := getSeriesName(row, reportQueryRequest.Rows)
		if err != nil {
			return nil, err
		}

		hcds := HighchartsDataSeries{
			Data: make([]float64, len(categories)),
		}

		category, err := getCategory(len(reportQueryRequest.Rows), row, reportQueryRequest.Cols)
		if err != nil {
			return nil, err
		}

		metricIndex := getCategoryIndex(category, categories)

		var metric float64
		switch v := row[selectedMetric].(type) {
		case int64:
			metric = float64(v)
		case float64:
			metric = v
		default:
			metric = 0
		}

		// Skip row if the report includes a trend filter and the row trend does not match
		if usingTrendFeatures {
			if rowTrend, ok := row[trendIndex].(string); ok && len(rowTrend) > 0 {
				hasTrend := false

				for _, trendFeature := range reportQueryRequest.Trends {
					if trendFeature == report.Feature(rowTrend) {
						hasTrend = true
						break
					}
				}

				if !hasTrend {
					continue
				}
			}
		}

		if r.Config.Aggregator == report.AggregatorTotalOverTotal && reportQueryRequest.Metric != report.MetricUsage {
			switch reportQueryRequest.GetMetricIndex() {
			case int(report.MetricCost):
				if denominator, ok := row[selectedMetric+1].(float64); !ok || denominator == 0 {
					metric = 0
				} else {
					metric = metric / denominator
				}
			case int(report.MetricSavings):
				if denominator, ok := row[selectedMetric-2].(float64); !ok || denominator == 0 {
					metric = 0
				} else {
					v := metric / denominator
					metric = (1 - 1/(v+1))
				}
			// relevant only for CSP report with margin metric
			case int(report.MetricMargin):
				if denominator, ok := row[selectedMetric-3].(float64); !ok || denominator == 0 {
					metric = 0
				} else {
					metric = metric / denominator
				}
			}
		}

		if len(series) == 0 {
			hcds.Name = name
			hcds.Data[metricIndex] = metric
			series = append(series, &hcds)
			hcds.total += metric
		} else if series[len(series)-1].Name != name {
			hcds.Name = name
			series = append(series, &hcds)
			hcds.Data[metricIndex] = metric
			hcds.total += metric
		} else {
			series[len(series)-1].Data[metricIndex] = metric
			series[len(series)-1].total += metric
		}
	}

	sortedSeries := sortDataSeries(series, r.Config.RowOrder == "asc")

	if reportQueryResult.ForecastRows != nil {
		hcds := HighchartsDataSeries{
			Name:      "ML Forecast",
			Type:      "line",
			Color:     "#B6B6B6",
			DashStyle: "ShortDot",
			Marker: struct {
				Enabled bool `json:"enabled"`
			}{
				Enabled: false,
			},
		}

		for _, row := range reportQueryResult.ForecastRows {
			data, ok := row[len(row)-1].(float64)
			if !ok {
				data = 0
			}

			hcds.Data = append(hcds.Data, data)
		}

		sortedSeries = append(sortedSeries, &hcds)
	}

	switch r.Config.Aggregator {
	case report.AggregatorPercentCol:
		for i := range categories {
			calculatePercentOfColumn(i, sortedSeries)
		}
	case report.AggregatorPercentTotal:
		calculatePercentOfTotal(categories, sortedSeries)
	case report.AggregatorPercentRow:
		calculatePercentOfRow(sortedSeries)
	}

	if len(sortedSeries) <= 8 {
		for _, s := range sortedSeries {
			s.ShowInLegend = true
		}
	} else {
		setShowInLegend(sortedSeries, reportQueryResult.ForecastRows != nil)
	}

	return sortedSeries, nil
}

func getSeriesName(row []bigquery.Value, queryRequestRows []*domainQuery.QueryRequestX) (string, error) {
	rowLen := len(queryRequestRows)
	nameArr := make([]string, 0)

	for i := 0; i < rowLen; i++ {
		if row[i] != nil {
			if name, err := query.BigqueryValueToString(row[i]); err != nil {
				return "", err
			} else {
				nameArr = append(nameArr, name)
			}
		} else {
			nullFallback := getNullFallback(queryRequestRows[i].ID)
			nameArr = append(nameArr, *nullFallback)
		}
	}

	return strings.Join(nameArr, "-"), nil
}

func getNullFallback(mdID string) *string {
	md, _, err := cloudanalytics.ParseID(mdID)
	if err != nil || md == nil || md.NullFallback == nil {
		nullFallback := "[N/A]"
		return &nullFallback
	}

	return md.NullFallback
}

func (s *Highcharts) GetLabels(highchartsFontSettings *domainHighCharts.HighchartsFontSettings) domainHighCharts.HighchartsDataAxisLabels {
	style := s.GetStyle(highchartsFontSettings)
	labels := domainHighCharts.HighchartsDataAxisLabels{
		Style: style,
	}

	return labels
}

func setShowInLegend(series []*HighchartsDataSeries, isForecast bool) {
	cpy := make([]*HighchartsDataSeries, len(series))
	copy(cpy, series)
	sort.Slice(cpy, func(i, j int) bool { return cpy[i].total > cpy[j].total })

	mlForecastIndex := -1
	lowestSeriesName := ""

	for i := 0; i < 8; i++ {
		name := cpy[i].Name
		if name == "ML Forecast" {
			mlForecastIndex = i
		}

		if i == 7 {
			lowestSeriesName = name
		}

		setSeriesShowInLegend(series, name, true)
	}

	if mlForecastIndex == -1 && isForecast {
		setSeriesShowInLegend(series, "ML Forecast", true)
		setSeriesShowInLegend(series, lowestSeriesName, false)
	}
}

func setSeriesShowInLegend(series []*HighchartsDataSeries, name string, value bool) {
	for _, s := range series {
		if s.Name == name {
			s.ShowInLegend = value
		}
	}
}

func (s *Highcharts) GetStyle(highchartsFontSettings *domainHighCharts.HighchartsFontSettings) domainHighCharts.HighchartsDataStyle {
	return domainHighCharts.HighchartsDataStyle{
		Color:         highchartsFontSettings.FontColor,
		FontFamily:    `"Lucida Grande", "Lucida Sans Unicode", Verdana, Arial, Helvetica, sans-serif`,
		FontSize:      highchartsFontSettings.FontSize,
		FontWeight:    400,
		LetterSpacing: "0.03333em",
		LineHeight:    1.66,
		TextOutline:   "none",
	}
}

func getSortedCategoriesByTotal(colsFirstIndex int, requestCols []*domainQuery.QueryRequestX, requestMetric report.Metric, resultRows [][]bigquery.Value, colOrder string) ([]string, error) {
	type categoryWithTotal struct {
		Category string
		Total    float64
	}

	categoriesWithTotal := make([]categoryWithTotal, 0)

	rows := resultRows
	for _, row := range rows {
		metricFirstIndex := colsFirstIndex + len(requestCols)
		metricValue := int(requestMetric)
		selectedMetric := metricFirstIndex + metricValue

		category, err := getCategory(colsFirstIndex, row, requestCols)
		if err != nil {
			return nil, err
		}

		cwt := categoryWithTotal{Category: category}
		isNewCategory := true

		for i, cat := range categoriesWithTotal {
			if cat.Category == category {
				isNewCategory = false
				categoriesWithTotal[i].Total += row[selectedMetric].(float64)
			}
		}

		if isNewCategory {
			cwt.Total = row[selectedMetric].(float64)
			categoriesWithTotal = append(categoriesWithTotal, cwt)
		}
	}

	if colOrder == "asc" {
		sort.Slice(categoriesWithTotal, func(i, j int) bool { return categoriesWithTotal[i].Total < categoriesWithTotal[j].Total })
	}

	if colOrder == "desc" {
		sort.Slice(categoriesWithTotal, func(i, j int) bool { return categoriesWithTotal[i].Total > categoriesWithTotal[j].Total })
	}

	newCategories := make([]string, 0)
	for _, cwt := range categoriesWithTotal {
		newCategories = append(newCategories, cwt.Category)
	}

	return newCategories, nil
}

func getTreemapValue(value float64, isTreemapExact bool) float64 {
	if isTreemapExact {
		return math.Abs(value)
	}

	return math.Sqrt(math.Abs(value))
}

func shouldFormatToPercent(cfg *report.Config, qr *cloudanalytics.QueryRequest) bool {
	if cfg.Aggregator == report.AggregatorCount {
		return false
	}

	if qr.CalculatedMetric != nil {
		return qr.CalculatedMetric.Format == metrics.MetricPercentageFormat
	}

	if cfg.Aggregator == report.AggregatorTotalOverTotal {
		return cfg.Metric != report.MetricCost
	}

	return cfg.Aggregator != report.AggregatorTotal
}

func sortDataSeries(s []*HighchartsDataSeries, ascending bool) []*HighchartsDataSeries {
	if ascending {
		sort.Sort(ByData(s))
	} else {
		sort.Sort(sort.Reverse(ByData(s)))
	}

	return s
}
