package assets

import (
	"testing"

	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
)

func TestHasAWSStandaloneFlexsave(t *testing.T) {
	type args struct {
		assets []*pkg.AWSAsset
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "returns true if has standalone",
			args: args{assets: []*pkg.AWSAsset{&pkg.AWSAsset{
				BaseAsset: pkg.BaseAsset{AssetType: "amazon-web-services-standalone"},
			}}},
			want: true,
		},
		{
			name: "returns true if mixed",
			args: args{assets: []*pkg.AWSAsset{
				&pkg.AWSAsset{
					BaseAsset: pkg.BaseAsset{AssetType: "awz"},
				},
				&pkg.AWSAsset{
					BaseAsset: pkg.BaseAsset{AssetType: "amazon-web-services-standalone"},
				}}},
			want: true,
		},
		{
			name: "returns false no assets",
			args: args{assets: []*pkg.AWSAsset{}},
			want: false,
		},
		{
			name: "returns false if no standalone assets",
			args: args{assets: []*pkg.AWSAsset{&pkg.AWSAsset{
				BaseAsset: pkg.BaseAsset{AssetType: "something else"},
			}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasAWSStandaloneFlexsave(tt.args.assets); got != tt.want {
				t.Errorf("HasAWSStandaloneFlexsave() = %v, want %v", got, tt.want)
			}
		})
	}
}
