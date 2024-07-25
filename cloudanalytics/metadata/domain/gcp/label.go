package domain

import (
	"strings"

	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
)

const GkeCostAllocationLabelPrefix = "k8s-label/"

var SystemLabelsMap = map[string]string{
	"compute.googleapis.com/cores":                 "Cores",
	"compute.googleapis.com/memory":                "Memory",
	"compute.googleapis.com/machine_spec":          "Machine Spec",
	"compute.googleapis.com/is_unused_reservation": "Unused Reservation",
	"cmp/eligible_for_commitment":                  "CUD Eligible",
	"cmp/commitment_type":                          "CUD Type",
	"cmp/machine_type":                             "Machine Type",
	"cmp/memory_to_core_ratio":                     "GB/CPU",
	"cmp/compute_resource_name":                    "GCE Resource",
	"cmp/flexsave_eligibility":                     "Flexsave Eligibility",
}

func FormatLabel(key string, mdType metadataDomain.MetadataFieldType) string {
	if mdType == metadataDomain.MetadataFieldTypeLabel {
		return strings.TrimPrefix(key, GkeCostAllocationLabelPrefix)
	}

	if mdType == metadataDomain.MetadataFieldTypeSystemLabel {
		if prettyLabel, prs := SystemLabelsMap[key]; prs {
			return prettyLabel
		}
	}

	return key
}
