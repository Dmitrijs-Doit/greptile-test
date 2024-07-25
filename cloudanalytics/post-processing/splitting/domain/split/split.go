package split

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
)

// swagger:enum Mode
// example: "proportional"
type Mode string

const (
	ModeEven         Mode = "even"
	ModeCustom       Mode = "custom"
	ModeProportional Mode = "proportional"
)

type Split struct {
	ID            string                     `json:"id"            firestore:"id"`
	Key           string                     `json:"key"           firestore:"key"`
	Type          metadata.MetadataFieldType `json:"type"          firestore:"type"`
	Origin        string                     `json:"origin"        firestore:"origin"`
	Mode          Mode                       `json:"mode"          firestore:"mode"`
	Targets       []SplitTarget              `json:"targets"       firestore:"targets"`
	IncludeOrigin bool                       `json:"includeOrigin" firestore:"includeOrigin"`
}

type SplitTarget struct {
	ID    string  `json:"id"    firestore:"id"`
	Value float64 `json:"value" firestore:"value"`
}

type SplitTargetPerMetric map[string][]float64

func (mode Mode) Validate() error {
	switch mode {
	case
		ModeEven,
		ModeCustom,
		ModeProportional:
		return nil
	default:
		return fmt.Errorf("%s: %s", ErrInvalidSplitMode, mode)
	}
}
