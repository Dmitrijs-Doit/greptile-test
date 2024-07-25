package utils

import (
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func GetCloudConnectAssetType(platform pkg.StandalonePlatform) string {
	switch platform {
	case pkg.GCP:
		return common.Assets.GoogleCloud
	case pkg.AWS:
		return common.Assets.AmazonWebServices
	default:
	}

	return ""
}
