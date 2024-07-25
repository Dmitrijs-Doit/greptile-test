package digest

import (
	"context"
	"fmt"
	"time"

	domainHighcharts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

var SendReportFontSettings domainHighcharts.HighchartsFontSettings = domainHighcharts.HighchartsFontSettings{
	FontColor:            "rgba(0, 0, 0, .87)",
	FontSize:             "0.6rem",
	LegendFontColor:      "rgba(0, 0, 0, .87)",
	LegendFontSize:       "12px",
	GridLineColor:        "#e6e6e6",
	ChartBackgroundColor: "#fff",
}

func (s *DigestService) getDigestHighchartsImagePath(ctx context.Context, req *GenerateTaskRequest, d *AttributionData, freq Frequency) (*string, error) {
	hcr := s.getDigestHighchartsRequest(d, freq)

	chartImageData, err := s.highchartsService.GetChartImage(ctx, hcr)
	if err != nil {
		return nil, err
	}

	imageIdentifier := req.AttributionID
	if req.OrganizationID != "" {
		imageIdentifier = fmt.Sprintf("%s_%s", imageIdentifier, req.OrganizationID)
	}

	reportID := fmt.Sprintf("%s_%s", imageIdentifier, time.Now().UTC().Format(time.RFC3339))

	path, err := s.highchartsService.SaveImageToGCS(ctx, chartImageData, reportID, req.CustomerID, "digests")
	if err != nil {
		return nil, err
	}

	return common.String(path), nil
}

func (s *DigestService) getDigestHighchartsRequest(d *AttributionData, freq Frequency) *domainHighcharts.HighchartsRequest {
	const (
		colorPreviousMonth  = "#EFEFEF"
		colorMonthToLastDay = "#B5AADA"
		colorLastDayOrWeek  = "#FC3165"
	)

	fontSettings := &domainHighcharts.HighchartsFontSettings{
		FontSize: "0.5rem",
	}

	var (
		lastDayOrWeekTotal float64
		bulletName         string
	)

	switch freq {
	case FrequencyDaily:
		if d.LastDayTotal != nil {
			lastDayOrWeekTotal = *d.LastDayTotal
			bulletName = "Last Day"
		}
	case FrequencyWeekly:
		if d.WeekToLastDay != nil {
			lastDayOrWeekTotal = *d.WeekToLastDay
			bulletName = "Week To Last Day"
		}
	default:
	}

	var currentMonthtotal float64
	if d.CurrentMonthTotal != nil {
		currentMonthtotal = *d.CurrentMonthTotal
	}

	series := []domainHighcharts.HighchartsBulletSeries{
		{
			Type: "bullet",
			Name: "Last Month Total",
			Data: []*domainHighcharts.BulletDataNode{
				{
					Color:  "transparent",
					Target: d.LastMonthTotal,
				},
			},
		},
		{
			Type: "bullet",
			Name: bulletName,
			Data: []*domainHighcharts.BulletDataNode{
				{
					Y: lastDayOrWeekTotal,
				},
			},
		},
		{
			Type: "bullet",
			Name: "Month To Last Day",
			Data: []*domainHighcharts.BulletDataNode{
				{
					Y: currentMonthtotal - lastDayOrWeekTotal,
				},
			},
		},
	}

	var to float64
	if d.LastMonthTotal != nil {
		to = *d.LastMonthTotal
	}

	yAxis := domainHighcharts.HighchartsDataYAxis{
		MaxPadding:    common.Int(0),
		GridLineWidth: common.Int(0),
		Labels: domainHighcharts.HighchartsDataAxisLabels{
			Enabled: common.Bool(true),
			Style:   s.highchartsService.GetStyle(fontSettings),
			X:       common.Int(10),
		},
		PlotBands: []domainHighcharts.BulletPlotBand{
			{
				From:  0,
				To:    to,
				Color: "#efefef",
			},
		},
	}
	chart := domainHighcharts.HighchartsDataChart{
		Inverted:        common.Bool(true),
		MarginLeft:      common.Int(0),
		SpacingLeft:     common.Int(0),
		MarginTop:       40,
		MarginBottom:    common.Int(30),
		BackgroundColor: "#fff",
		Type:            domainHighcharts.HighchartsTypeBullet,
	}
	infile := domainHighcharts.HighchartsData{
		Series:      series,
		Chart:       chart,
		YAxis:       yAxis,
		Colors:      []string{colorPreviousMonth, colorLastDayOrWeek, colorMonthToLastDay},
		PlotOptions: getBulletPlotOptions(),
		Legend: domainHighcharts.HighchartsDataLegend{
			Align:         common.String("left"),
			VerticalAlign: common.String("top"),
			Reversed:      common.Bool(true),
			Enabled:       true,
			ItemStyle: domainHighcharts.HighchartsLegendItemStyle{
				Color:        SendReportFontSettings.LegendFontColor,
				FontSize:     SendReportFontSettings.LegendFontSize,
				FontWeight:   "400",
				TextOverflow: "ellipsis",
			},
		},
	}
	hcr := &domainHighcharts.HighchartsRequest{
		Infile:         &infile,
		Width:          500,
		Height:         130,
		Scale:          2,
		AsyncRendering: false,
	}

	return hcr
}

func getBulletPlotOptions() *domainHighcharts.BulletPlotOptions {
	return &domainHighcharts.BulletPlotOptions{
		Series: &domainHighcharts.BulletPlotOptionsSeries{
			PointPadding: 0,
			PointWidth:   common.Int(45),
			BorderWidth:  1.5,
			BorderColor:  common.String("#efefef"),
			TargetOptions: struct {
				Width string `json:"width"`
			}{
				Width: "100%",
			},
			Animation: false,
			Stacking:  "normal",
		},
	}
}
