package cspreport

import "github.com/doitintl/hello/scheduled-tasks/common"

func GetCSPMetadataTable() string {
	if common.Production {
		return "doitintl_csp_metadata_v1"
	}

	return "doitintl_csp_metadata_v1beta"
}
