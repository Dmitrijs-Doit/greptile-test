package mpa

import (
	"cmp"
	"context"
	"io"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/goccy/go-json"
)

func (s *MasterPayerAccountService) validateCURFileExistence(accountID, curBucket, curPath string) error {
	objects, err := s.awsClient.ListObjectsV2(accountID, curBucket)
	if err != nil {
		return err
	}

	if objects == nil || objects.Contents == nil {
		return errorBucketEmpty
	}

	for _, obj := range objects.Contents {
		if strings.Contains(*obj.Key, curPath) && strings.Contains(*obj.Key, "Manifest.json") {
			return nil
		}
	}

	return errorCURNotFound
}

func (s *MasterPayerAccountService) validateCUR(accountID, curBucket, curPath string) error {
	describeReportDefinitions, err := s.awsClient.DescribeReportDefinitions(accountID)
	if err != nil {
		return err
	}

	var mostRelevantError error

	for _, report := range describeReportDefinitions.ReportDefinitions {
		reportName := extractCURName(curPath)
		validBucket := *report.S3Bucket == curBucket
		validPath := curPath == *report.S3Prefix || reportName == *report.ReportName

		if !validBucket || !validPath {
			mostRelevantError = errorCURPath
			continue
		}

		validFormat := false
		if compression, ok := ReportFormats[*report.Format]; ok {
			validFormat = compression == *report.Compression
		}

		validTimeUnit := *report.TimeUnit == TimeGranularityHourly

		validProperties := validTimeUnit && validFormat

		validSchema := false

		for _, element := range report.AdditionalSchemaElements {
			if *element == SchemaElementsResources {
				validSchema = true
			}
		}

		if !validProperties || !validSchema {
			mostRelevantError = errorCURProperties
		}

		if validBucket && validPath && validProperties && validSchema {
			return nil
		}
	}

	return mostRelevantError
}

func (s *MasterPayerAccountService) validateRole(accountID, roleArn string, isSaaS bool) ([]string, error) {
	role, err := s.awsClient.GetRole(accountID, roleArn)
	if err != nil {
		return []string{"iam:GetRole"}, err
	}

	if role == nil || role.Role == nil {
		return []string{}, errorRole
	}

	validArn := *role.Role.Arn == roleArn
	validName := *role.Role.RoleName == doitRole

	if !validName || !validArn {
		return []string{}, errorRole
	}

	if isSaaS {
		roleDocument, err := url.QueryUnescape(*role.Role.AssumeRolePolicyDocument)
		if err != nil {
			return []string{}, err
		}

		return s.validateSaaSRoleHasRequiredPermissions(roleDocument)
	}

	return []string{}, nil
}

func (s *MasterPayerAccountService) validatePolicy(ctx context.Context, accountID, roleArn, curBucket string, isSaaS bool, timeCreated *time.Time) ([]string, error) {
	policyArn := getPolicyArn(roleArn)

	policy, err := s.awsClient.GetPolicy(accountID, policyArn)
	if err != nil {
		return []string{"iam:GetPolicy"}, err
	}

	if policy == nil || policy.Policy == nil {
		return []string{}, errorPolicy
	}

	validArn := *policy.Policy.Arn == policyArn
	validName := *policy.Policy.PolicyName == doitPolicy

	if !validName || !validArn {
		return []string{}, errorPolicy
	}

	policyVersion, err := s.awsClient.GetPolicyVersion(accountID, *policy.Policy.Arn, *policy.Policy.DefaultVersionId)
	if err != nil {
		return []string{"iam:GetPolicyVersion"}, err
	}

	if policyVersion == nil || policyVersion.PolicyVersion == nil {
		return []string{}, errorPolicy
	}

	policyDocument, err := url.QueryUnescape(*policyVersion.PolicyVersion.Document)
	if err != nil {
		return []string{}, err
	}

	if isSaaS {
		return s.validateSaaSPolicyHasRequiredPermissions(ctx, accountID, curBucket, policyDocument, timeCreated)
	}

	return []string{}, s.validatePolicyHasRequiredPermissions(accountID, curBucket, policyDocument)
}

func getDoitPolicyFromTemplate(templatePath string) ([]byte, error) {
	doitPolicyFile, err := os.Open(templatePath)
	if err != nil {
		return nil, err
	}
	defer doitPolicyFile.Close()

	return io.ReadAll(doitPolicyFile)
}

func getPolicyArn(roleArn string) string {
	return strings.Replace(roleArn, "role", "policy", -1)
}

func extractCURName(curPath string) string {
	pathParts := strings.Split(curPath, "/")
	length := len(pathParts)

	return pathParts[length-1]
}

func getPolicyPermissionsFromTemplate(templateFilePath, accountID, curBucket string) (*PolicyPermissions, error) {
	// Read the expected permissions template from the filesystem
	policyFileReader, err := os.Open(templateFilePath)
	if err != nil {
		return nil, err
	}
	defer policyFileReader.Close()

	// initialize a new json decoder that produces idempotent results, whatever the IAM policy statement/actions order
	// and replaces the template accountID placeholders with the actual compared accountID
	decoder := json.NewDecoder(policyFileReader)
	policyPermissions := &PolicyPermissions{
		accountID: accountID,
		curBucket: curBucket,
	}

	return policyPermissions, decoder.Decode(policyPermissions)
}

// getConditionalOptionalPermissionsFromTemplate returns a PolicyPermissions pointer
// containing statements that are conditionally included if the created time of the
// actual policy is after the policy date in the template.
func getConditionalOptionalPermissionsFromTemplate(l logger.ILogger, templatePath string, policyTimeCreated time.Time) *PolicyPermissions {
	jsonPayload, err := os.ReadFile(templatePath)
	if err != nil {
		l.Errorf("failed to read template file with error: %s", err)
		return nil
	}

	cp := DatedPolicyPermissions{}

	err = json.Unmarshal(jsonPayload, &cp)
	if err != nil {
		l.Errorf("failed to parse json with error: %s", err)
		return nil
	}

	optionalPermissions := &PolicyPermissions{}

	for dateStr, statements := range cp.DatedStatement {
		startTime, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			l.Errorf("failed to parse start date with error: %s", err)
			continue
		}

		if policyTimeCreated.Sub(startTime) < 0 {
			optionalPermissions.Statement = append(optionalPermissions.Statement, statements...)
		}
	}

	if len(optionalPermissions.Statement) == 0 {
		return nil
	}

	return optionalPermissions
}

// applyPermissionMask returns a PolicyPermissions pointer which values are the symmetric difference
// of 'permissions' and 'mask' actions and resources.
//
// e.g. removing all actions and resources elements present in 'mask' from 'permissions'
// removing the entire statement if the resulting actions or resources are nil.
//
// applyPermissionMask is idempotent, the returned values and nested values are all lexically ordered.
// if mask is nil, applyPermissionMask returns the permissions input pointer.
func applyPermissionMask(permissions, mask *PolicyPermissions) *PolicyPermissions {
	if mask == nil {
		return permissions
	}

	res := permissions

	maskMap := mask.ToProcessedStatementsMap()
	permissionsMap := permissions.ToProcessedStatementsMap()

	for sid, maskStatement := range maskMap {
		p, ok := permissionsMap[sid]
		if !ok {
			continue
		}

		for _, pr := range p.Resource {
			// if the resource do not intersect, then the action does not need to be masked
			if slices.Contains(maskStatement.Resource, "*") || pr == "*" || slices.Contains(maskStatement.Resource, pr) {
				// delete the items that are present in the mask actions
				p.Action = sliceMask(p.Action, maskStatement.Action)
				slices.Sort(p.Action)

				for i, s := range res.Statement {
					if s.Sid == sid {
						res.Statement[i].Action = p.Action

						// remove the whole statement if the resulting actions are empty
						if len(res.Statement[i].Action.([]string)) == 0 {
							res.Statement = slices.Delete(res.Statement, i, i+1)
						}
					}
				}
				// do not continue iterating on resource as a match has already been found
				break
			}
		}
	}

	// ensure the statements are lexically ordered, for a deterministic result
	slices.SortStableFunc(res.Statement, func(a, b Statement) int {
		return cmp.Compare(a.Sid, b.Sid)
	})

	if len(res.Statement) == 0 {
		res.Statement = nil
	}

	return res
}
