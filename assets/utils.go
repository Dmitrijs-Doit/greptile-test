package assets

import "github.com/doitintl/hello/scheduled-tasks/assets/pkg"

func HasAWSStandaloneFlexsave(assets []*pkg.AWSAsset) bool {
	for _, asset := range assets {
		if asset.AssetType == pkg.AssetStandaloneAWS {
			return true
		}
	}

	return false
}
