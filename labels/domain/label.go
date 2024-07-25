package labels

import (
	"time"

	"cloud.google.com/go/firestore"
)

type Label struct {
	Name         string                   `firestore:"name"`
	Color        LabelColor               `firestore:"color"`
	CreatedBy    string                   `firestore:"createdBy"`
	Customer     *firestore.DocumentRef   `firestore:"customer"`
	TimeModified time.Time                `firestore:"timeModified"`
	TimeCreated  time.Time                `firestore:"timeCreated"`
	Objects      []*firestore.DocumentRef `firestore:"objects"`

	Ref *firestore.DocumentRef `firestore:"-"`
}

type LabelColor string

const (
	LightBlue    LabelColor = "#BEE1F5"
	Salmon       LabelColor = "#F2C7CF"
	LightGrey    LabelColor = "#CECEDB"
	Teal         LabelColor = "#9DD9D9"
	Green        LabelColor = "#A2B3AC"
	Grey         LabelColor = "#A3A7C1"
	GreyBlue     LabelColor = "#A9C8DF"
	SilverPink   LabelColor = "#CCA8B2"
	ColumbiaBlue LabelColor = "#C5DEE3"
	Pink         LabelColor = "#E6ABBB"
	SlateGrey    LabelColor = "#6A7F9F"
)

func (c LabelColor) IsValid() bool {
	switch c {
	case LightBlue, Salmon, LightGrey, Teal, Green, Grey, GreyBlue, SilverPink, ColumbiaBlue, Pink, SlateGrey:
		return true
	default:
		return false
	}
}

type ObjectType string

const (
	AlertType             ObjectType = "alert"
	AttributionsGroupType ObjectType = "attribution_group"
	AttributionType       ObjectType = "attribution"
	BudgetType            ObjectType = "budget"
	MetricType            ObjectType = "metric"
	ReportType            ObjectType = "report"
)

func (o ObjectType) IsValid() bool {
	switch o {
	case AlertType, AttributionsGroupType, AttributionType, BudgetType, MetricType, ReportType:
		return true
	default:
		return false
	}
}
