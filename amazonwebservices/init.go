package amazonwebservices

import (
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

const (
	slackChannel = "#sales-ops"

	organizationAccountAccessRoleArnFormat = "arn:aws:iam::%s:role/OrganizationAccountAccessRole"

	// DefaultPayerAccount is MPA 1
	DefaultPayerAccount = "561602220360"
)

var Regions map[string]string

// AccountIDRegexp is regular exp. to for AWS account ids
var AccountIDRegexp = regexp.MustCompile("^\\d{12}$")

func init() {
	resolver := endpoints.DefaultResolver()
	partitions := resolver.(endpoints.EnumPartitions).Partitions()
	Regions = make(map[string]string)

	regionNameExceptions := []string{"Europe (Spain)", "Europe (Zurich)"}

	for _, p := range partitions {
		for id, v := range p.Regions() {
			if !slice.Contains(regionNameExceptions, v.Description()) {
				// hotfix for EU location names - does not apply to Europe (Spain) and Europe (Zurich)
				Regions[id] = strings.Replace(v.Description(), "Europe", "EU", 1)
			} else {
				Regions[id] = v.Description()
			}
		}
	}
}
