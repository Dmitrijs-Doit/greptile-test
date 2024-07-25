package scripts

import (
	"fmt"
	"regexp"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var sevLevelRegexp = regexp.MustCompile(`Current severity level: (.*?)\n`)

func AddLevelKnownIssuesAWS(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)
	dryRun := ctx.Query("dryRun") == "true"

	var errors []error

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		errors = append(errors, fmt.Errorf("error creating firestore client"))
		return errors
	}
	defer fs.Close()

	AWSKnonwIssues := fs.Collection("knownIssues").Where("platform", "==", "amazon-web-services").Documents(ctx)

	for {
		doc, err := AWSKnonwIssues.Next()

		if err == iterator.Done {
			break
		} else if err != nil {
			errors = append(errors, fmt.Errorf("error iterating ramp plans: %v", err))
		}

		outageDescription := doc.Data()["outageDescription"]
		outageDescriptionString, ok := outageDescription.(string)

		if !ok {
			errors = append(errors, fmt.Errorf("error converting outageDescription to string"))
			continue
		}

		level := getAwsKnownIssueLevel(outageDescriptionString)
		if level != "" {
			l.Infof("dry run: %v: update known issue: %s level: %s", dryRun, doc.Ref.ID, level)

			if dryRun {
				continue
			}

			_, err = doc.Ref.Set(ctx, map[string]interface{}{
				"exposureLevel": level,
			}, firestore.MergeAll)
			if err != nil {
				errors = append(errors, fmt.Errorf("error updating known issue: %v", err))
			}
		}
	}

	return errors
}

func getAwsKnownIssueLevel(knownIssueDescription string) string {
	match := sevLevelRegexp.FindStringSubmatch(knownIssueDescription)

	if len(match) > 0 {
		return match[1]
	}

	return ""
}
