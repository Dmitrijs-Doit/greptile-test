package invoicing

import (
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

func TestGoogleCloud_calculateValuesPerBucket(t *testing.T) {
	defaultEntityID := "iSE11561QSTjkuJ3C7xT"
	defaultEntityRef := &firestore.DocumentRef{
		ID: defaultEntityID,
	}

	defaultBucketID := "RGbMYUZhcXKWilgTMumX"
	defaultBucketRef := &firestore.DocumentRef{
		ID: defaultBucketID,
	}

	customerID := "someCustomerID"

	entityID := "someEntityID"
	entityRef := &firestore.DocumentRef{
		ID: entityID,
	}

	bucketID := "someBucketID"
	bucketRef := &firestore.DocumentRef{
		ID: bucketID,
	}

	projectsValuePerBucketProjects := ProjectsValuePerBucket{
		fmt.Sprintf("%s%s", entityID, bucketID): &ProjectsValue{
			Bucket: bucketRef,
			Entity: entityRef,
			Value:  -100.5,
		},
		fmt.Sprintf("%s%s", defaultEntityID, defaultBucketID): &ProjectsValue{
			Bucket: defaultBucketRef,
			Entity: defaultEntityRef,
			Value:  -500.7,
		},
	}

	type args struct {
		projectsValues              map[string]float64
		projectsSettingsMap         map[string]*common.AssetSettings
		billingAccountAssetSettings common.AssetSettings
		useBillingAccountSettings   bool
		defaultEntity               *firestore.DocumentRef
		defaultBucket               *firestore.DocumentRef
	}

	tests := []struct {
		name    string
		args    args
		want    ProjectsValuePerBucket
		wantErr error
	}{
		{
			name: "happy path",
			args: args{
				projectsValues: map[string]float64{
					"project1": -100.5,
					"project2": -200.3,
					"project3": -300.4,
				},
				projectsSettingsMap: map[string]*common.AssetSettings{
					"google-cloud-project-project1": {
						BaseAsset: common.BaseAsset{
							Customer: &firestore.DocumentRef{
								ID: customerID,
							},
							Entity: entityRef,
							Bucket: bucketRef,
						},
					},
				},
				billingAccountAssetSettings: common.AssetSettings{
					BaseAsset: common.BaseAsset{
						Customer: &firestore.DocumentRef{
							ID: customerID,
						},
					},
				},
				defaultEntity: defaultEntityRef,
				defaultBucket: defaultBucketRef,
			},
			want:    projectsValuePerBucketProjects,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateValuesPerBucket(
				tt.args.projectsValues,
				tt.args.projectsSettingsMap,
				tt.args.billingAccountAssetSettings,
				tt.args.useBillingAccountSettings,
				tt.args.defaultEntity,
				tt.args.defaultBucket,
			)

			assert.Equal(t, tt.want, got)
		})
	}
}
