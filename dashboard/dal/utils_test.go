package dal

import (
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

func Test_getCleanDashboardDocumentPathFromRef(t *testing.T) {
	type args struct {
		ref *firestore.DocumentRef
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test public dashboard path",
			args: args{
				ref: &firestore.DocumentRef{
					Path: fmt.Sprintf("projects/%s/databases/(default)/documents/customers/CUSTOMER_ID/publicDashboards/DASHBOARD_ID", common.ProjectID),
				},
			},
			want: "customers/CUSTOMER_ID/publicDashboards/DASHBOARD_ID",
		},
		{
			name: "test private dashboard path",
			args: args{
				ref: &firestore.DocumentRef{
					Path: fmt.Sprintf("projects/%s/databases/(default)/documents/dashboards/customization/users/USER_ID/duc/CUSTOMER_ID/dashboards/DASHBOARD_ID", common.ProjectID),
				},
			},
			want: "dashboards/customization/users/USER_ID/duc/CUSTOMER_ID/dashboards/DASHBOARD_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getCleanDashboardDocumentPathFromRef(tt.args.ref); got != tt.want {
				t.Errorf("getCleanDashboardDocumentPathFromRef() = %v, want %v", got, tt.want)
			}
		})
	}
}
