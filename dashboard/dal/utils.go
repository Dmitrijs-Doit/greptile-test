package dal

import (
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

// Firestore document ref path looks like this: projects/<project>/databases/(default)/documents/<collection>/<document>
var dashboardFirestorePathPrefix = fmt.Sprintf("projects/%s/databases/(default)/documents/", common.ProjectID)

func getCleanDashboardDocumentPathFromRef(ref *firestore.DocumentRef) string {
	after, ok := strings.CutPrefix(ref.Path, dashboardFirestorePathPrefix)
	if !ok {
		return ""
	}

	return after
}
