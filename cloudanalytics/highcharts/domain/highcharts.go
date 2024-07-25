package domain

type HighchartsRequest struct {
	Infile         *HighchartsData `json:"infile"`
	Type           string          `json:"type"`
	B64            bool            `json:"b64"`
	Callback       string          `json:"callback"`
	Scale          int             `json:"scale"`
	Width          int             `json:"width"`
	Height         int             `json:"height"`
	AsyncRendering bool            `json:"asyncRendering"`
}

type HighchartsData struct {
	Chart   HighchartsDataChart `json:"chart"`
	Credits struct {
		Enabled bool `json:"enabled"`
	} `json:"credits"`
	Boost struct {
		Enabled bool `json:"enabled"`
	} `json:"boost"`
	Title       HighchartsDataTitle  `json:"title"`
	Tooltip     struct{}             `json:"tooltip"`
	Colors      []string             `json:"colors"`
	Series      interface{}          `json:"series"`
	XAxis       HighchartsDataXAxis  `json:"xAxis"`
	YAxis       HighchartsDataYAxis  `json:"yAxis"`
	Legend      HighchartsDataLegend `json:"legend"`
	PlotOptions interface{}          `json:"plotOptions"`
}

type HighchartsDataTitle struct {
	Text  string              `json:"text"`
	Style HighchartsDataStyle `json:"style"`
}

type HighchartsDataLegend struct {
	Enabled       bool                      `json:"enabled"`
	ItemStyle     HighchartsLegendItemStyle `json:"itemStyle"`
	Y             *int                      `json:"y,omitempty"`
	Align         *string                   `json:"align,omitempty"`
	VerticalAlign *string                   `json:"verticalAlign,omitempty"`
	Reversed      *bool                     `json:"reversed,omitempty"`
}

type HighchartsLegendItemStyle struct {
	Color        string `json:"color"`
	FontSize     string `json:"fontSize"`
	FontWeight   string `json:"fontWeight"`
	TextOverflow string `json:"textOverflow"`
}

type HighchartsDataChart struct {
	BackgroundColor string         `json:"backgroundColor"`
	Type            HighchartsType `json:"type"`
	ZoomType        string         `json:"zoomType"`
	Width           int            `json:"width,omitempty"`
	Height          int            `json:"height,omitempty"`
	MarginTop       int            `json:"marginTop"`
	ResetZoomButton struct {
		Position string `json:"position"`
	} `json:"resetZoomButton"`
	Inverted     *bool `json:"inverted,omitempty"`
	MarginLeft   *int  `json:"marginLeft,omitempty"`
	SpacingLeft  *int  `json:"spacingLeft,omitempty"`
	MarginBottom *int  `json:"marginBottom,omitempty"`
}

type HighchartsDataXAxis struct {
	Categories []string                 `json:"categories"`
	Crosshair  bool                     `json:"crosshair"`
	Labels     HighchartsDataAxisLabels `json:"labels"`
}

type HighchartsDataAxisLabels struct {
	Style        HighchartsDataStyle `json:"style"`
	Align        string              `json:"align,omitempty"`
	ReserveSpace *bool               `json:"reserveSpace,omitempty"`
	UseHTML      *bool               `json:"useHTML,omitempty"`
	Enabled      *bool               `json:"enabled,omitempty"`
	X            *int                `json:"x,omitempty"`
}

type HighchartsDataStyle struct {
	Color         string  `json:"color"`
	FontFamily    string  `json:"fontFamily"`
	FontSize      string  `json:"fontSize"`
	FontWeight    int     `json:"fontWeight"`
	LetterSpacing string  `json:"letterSpacing"`
	LineHeight    float32 `json:"lineHeight"`
	TextOutline   string  `json:"textOutline"`
}

type HighchartsDataYAxis struct {
	Reversed      bool                           `json:"reversed"`
	Title         HighchartsDataYAxisTitle       `json:"title"`
	Labels        HighchartsDataAxisLabels       `json:"labels"`
	StackLabels   HighchartsDataYAxisStackLabels `json:"stackLabels"`
	GridLineColor string                         `json:"gridLineColor"`
	PlotBands     []BulletPlotBand               `json:"plotBands,omitempty"`
	EndOnTick     *bool                          `json:"endOnTick,omitempty"`
	MaxPadding    *int                           `json:"maxPadding,omitempty"`
	GridLineWidth *int                           `json:"gridLineWidth,omitempty"`
	Type          *string                        `json:"type,omitempty"`
}

type HighchartsDataYAxisStackLabels struct {
	Enabled      bool                `json:"enabled"`
	Style        HighchartsDataStyle `json:"style"`
	AllowOverlap bool                `json:"allowOverlap"`
	Rotation     int                 `json:"rotation"`
	Formatter    string              `json:"formatter,omitempty"`
}

type HighchartsDataYAxisTitle struct {
	Enabled bool `json:"enabled"`
}

type StackedColumnPlotOptions struct {
	Column struct {
		Stacking      string  `json:"stacking"`
		BorderWidth   int     `json:"borderWidth"`
		MaxPointWidth int     `json:"maxPointWidth"`
		GroupPadding  float64 `json:"groupPadding"`
		PointPadding  float64 `json:"pointPadding"`
		DataLabels    struct {
			Enabled bool `json:"enabled"`
		} `json:"dataLabels"`
	} `json:"column"`
}

type ColumnPlotOptions struct {
	Column struct {
		BorderWidth   int     `json:"borderWidth"`
		MaxPointWidth int     `json:"maxPointWidth"`
		GroupPadding  float64 `json:"groupPadding"`
		PointPadding  float64 `json:"pointPadding"`
		DataLabels    struct {
			Enabled bool `json:"enabled"`
		} `json:"dataLabels"`
	} `json:"column"`
}

type BarPlotOptions struct {
	Bar struct {
		BorderWidth   int     `json:"borderWidth"`
		MaxPointWidth int     `json:"maxPointWidth"`
		GroupPadding  float64 `json:"groupPadding"`
		PointPadding  float64 `json:"pointPadding"`
		DataLabels    struct {
			Enabled bool `json:"enabled"`
		} `json:"dataLabels"`
	} `json:"bar"`
}

type StackedBarPlotOptions struct {
	Bar struct {
		Stacking      string  `json:"stacking"`
		BorderWidth   int     `json:"borderWidth"`
		MaxPointWidth int     `json:"maxPointWidth"`
		GroupPadding  float64 `json:"groupPadding"`
		PointPadding  float64 `json:"pointPadding"`
		DataLabels    struct {
			Enabled bool `json:"enabled"`
		} `json:"dataLabels"`
	} `json:"bar"`
}

type TreemapDataNode struct {
	ID               string  `json:"id"`
	Value            float64 `json:"value"`
	RealValue        float64 `json:"realValue"`
	Parent           string  `json:"parent"`
	Name             string  `json:"name"`
	TotalSeriesValue float64 `json:"totalSeriesValue"`
}

type HighchartsTreemapSeries struct {
	Type            string                  `json:"type"`
	LayoutAlgorithm string                  `json:"layoutAlgorithm"`
	Data            []*TreemapDataNode      `json:"data"`
	Opacity         float64                 `json:"opacity"`
	Levels          []TreemapSeriesLevel    `json:"levels"`
	DataLabels      TreemapSeriesDataLabels `json:"dataLabels"`
}

type TreemapSeriesLevel struct {
	Level        int  `json:"level"`
	ColorByPoint bool `json:"colorByPoint"`
	BorderWidth  int  `json:"borderWidth"`
	DataLabels   struct {
		Enabled bool `json:"enabled"`
	} `json:"dataLabels"`
}

type TreemapSeriesDataLabels struct {
	Enabled bool `json:"enabled"`
}

type HighchartsBulletSeries struct {
	Type         string            `json:"type"`
	Data         []*BulletDataNode `json:"data"`
	Name         string            `json:"name"`
	ShowInLegend *bool             `json:"showInLegend,omitempty"`
}

type BulletDataNode struct {
	Color      string                    `json:"color"`
	Target     *float64                  `json:"target,omitempty"`
	Y          float64                   `json:"y"`
	DataLabels *HighchartsDataAxisLabels `json:"dataLabels,omitempty"`
}

type BulletPlotBand struct {
	From  float64 `json:"from"`
	To    float64 `json:"to"`
	Color string  `json:"color"`
}

type BulletPlotOptions struct {
	Series *BulletPlotOptionsSeries `json:"series"`
}

type BulletPlotOptionsSeries struct {
	PointPadding  float64 `json:"pointPadding"`
	PointWidth    *int    `json:"pointWidth,omitempty"`
	BorderWidth   float64 `json:"borderWidth"`
	BorderColor   *string `json:"borderColor,omitempty"`
	Color         string  `json:"color"`
	TargetOptions struct {
		Width string `json:"width"`
	} `json:"targetOptions"`
	Animation bool   `json:"animation"`
	Stacking  string `json:"stacking,omitempty"`
}

type HighchartsExporting struct {
	Enabled      bool `json:"enabled"`
	SourceHeight int  `json:"sourceHeight"`
	SourceWidth  int  `json:"sourceWidth"`
}

type HighchartsType string

// Highchart chart types
const (
	HighchartsTypeArea       HighchartsType = "area"
	HighchartsTypeColumn     HighchartsType = "column"
	HighchartsTypeBar        HighchartsType = "bar"
	HighchartsTypeLine       HighchartsType = "line"
	HighchartsTypeSpline     HighchartsType = "spline"
	HighchartsTypeAreaSpline HighchartsType = "areaspline"
	HighchartsTypeTreemap    HighchartsType = "treemap"
	HighchartsTypeBullet     HighchartsType = "bullet"
)

type HighchartsFontSettings struct {
	FontColor            string
	FontSize             string
	LegendFontColor      string
	LegendFontSize       string
	GridLineColor        string
	ChartBackgroundColor string
}

// HighchartsFontSettings instanciation for different highchart server requests
var (
	SlackUnfurlFontSettings = HighchartsFontSettings{
		FontColor:            "rgba(150, 157, 168, 1)",
		FontSize:             "0.8rem",
		LegendFontColor:      "rgba(150, 157, 168, 1)",
		LegendFontSize:       "14px",
		GridLineColor:        "rgba(150, 157, 168, .3)",
		ChartBackgroundColor: "transparent",
	}
	SendReportFontSettings = HighchartsFontSettings{
		FontColor:            "rgba(0, 0, 0, .87)",
		FontSize:             "0.6rem",
		LegendFontColor:      "rgba(0, 0, 0, .87)",
		LegendFontSize:       "12px",
		GridLineColor:        "#e6e6e6",
		ChartBackgroundColor: "#fff",
	}
	DashboardSubscriptionSettings = HighchartsFontSettings{
		FontColor:            "rgba(0, 0, 0, .87)",
		FontSize:             "1.6rem",
		LegendFontColor:      "rgba(0, 0, 0, .87)",
		LegendFontSize:       "20px",
		GridLineColor:        "#e6e6e6",
		ChartBackgroundColor: "#fff",
	}
)
