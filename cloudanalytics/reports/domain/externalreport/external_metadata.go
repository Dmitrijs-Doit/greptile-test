package externalreport

import "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"

func toInternalMetadataType(dimensionType metadata.MetadataFieldType) metadata.MetadataFieldType {
	if dimensionType == metadata.MetadataFieldTypeOrganizationTagExternal {
		dimensionType = metadata.MetadataFieldTypeProjectLabel
	}

	return dimensionType
}
