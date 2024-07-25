package aws

/*
import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

const (
	projectID       = "doitintl-cmp-spot0-dev"
	withSpotScaling = true
)

func PermissionScript() error {
	ctx := context.Background()

	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("could not initialize firestore client. error %s", err)
	}

	awsCorePolicies := []string{
		"arn:aws:iam::aws:policy/SecurityAudit",
		"arn:aws:iam::aws:policy/AWSSavingsPlansReadOnlyAccess",
		"arn:aws:iam::aws:policy/job-function/Billing",
	}
	awsCoreFeaturePermissions := []string{
		"ec2:Describe*",
		"iam:List*",
		"iam:Get*",
		"autoscaling:Describe*",
		"aws-portal:ViewBilling",
		"aws-portal:ViewUsage",
		"cloudformation:ListStacks",
		"cloudformation:ListStackResources",
		"cloudformation:DescribeStacks",
		"cloudformation:DescribeStackEvents",
		"cloudformation:DescribeStackResources",
		"cloudformation:GetTemplate",
		"cloudfront:Get*",
		"cloudfront:List*",
		"cloudtrail:DescribeTrails",
		"cloudtrail:GetEventSelectors",
		"cloudtrail:ListTags",
		"cloudwatch:Describe*",
		"cloudwatch:Get*",
		"cloudwatch:List*",
		"config:Get*",
		"config:Describe*",
		"config:Deliver*",
		"config:List*",
		"cur:Describe*",
		"dms:Describe*",
		"dms:List*",
		"dynamodb:DescribeTable",
		"dynamodb:List*",
		"ec2:Describe*",
		"ec2:GetReservedInstancesExchangeQuote",
		"ecs:List*",
		"ecs:Describe*",
		"elasticache:Describe*",
		"elasticache:ListTagsForResource",
		"elasticbeanstalk:Check*",
		"elasticbeanstalk:Describe*",
		"elasticbeanstalk:List*",
		"elasticbeanstalk:RequestEnvironmentInfo",
		"elasticbeanstalk:RetrieveEnvironmentInfo",
		"elasticfilesystem:Describe*",
		"elasticloadbalancing:Describe*",
		"elasticmapreduce:Describe*",
		"elasticmapreduce:List*",
		"es:List*",
		"es:Describe*",
		"firehose:ListDeliveryStreams",
		"firehose:DescribeDeliveryStream",
		"iam:List*",
		"iam:Get*",
		"iam:GenerateCredentialReport",
		"kinesis:Describe*",
		"kinesis:List*",
		"kms:DescribeKey",
		"kms:GetKeyRotationStatus",
		"kms:ListKeys",
		"lambda:List*",
		"logs:Describe*",
		"redshift:Describe*",
		"route53:Get*",
		"route53:List*",
		"rds:Describe*",
		"rds:ListTagsForResource",
		"s3:GetBucketAcl",
		"s3:GetBucketLocation",
		"s3:GetBucketLogging",
		"s3:GetBucketPolicy",
		"s3:GetBucketTagging",
		"s3:GetBucketVersioning",
		"s3:GetBucketWebsite",
		"s3:List*",
		"sagemaker:Describe*",
		"sagemaker:List*",
		"savingsplans:DescribeSavingsPlans",
		"sdb:GetAttributes",
		"sdb:List*",
		"ses:Get*",
		"ses:List*",
		"sns:Get*",
		"sns:List*",
		"sqs:GetQueueAttributes",
		"sqs:ListQueues",
		"storagegateway:List*",
		"storagegateway:Describe*",
		"workspaces:Describe*",
	}
	awsSpotScalingFeaturePermissions := []string{
		"ec2:Describe*",
		"ec2:CreateLaunchTemplate",
		"ec2:CreateLaunchTemplateVersion",
		"ec2:ModifyLaunchTemplate",
		"ec2:RunInstances",
		"ec2:TerminateInstances",
		"ec2:CreateTags",
		"ec2:DeleteTags",
		"ec2:CreateLaunchTemplateVersion",
		"ec2:CancelSpotInstanceRequests",
		"autoscaling:CreateOrUpdateTags",
		"autoscaling:UpdateAutoScalingGroup",
		"autoscaling:Describe*",
		"autoscaling:AttachInstances",
		"autoscaling:BatchDeleteScheduledAction",
		"autoscaling:BatchPutScheduledUpdateGroupAction",
		"cloudformation:ListStacks",
		"cloudformation:Describe*",
		"iam:PassRole",
		"events:PutRule",
		"events:PutTargets",
		"events:PutEvents",
	}
	awsQuotasFeaturePermissions := []string{
		"support:DescribeTrustedAdvisorCheckSummaries",
		"support:DescribeTrustedAdvisorCheckRefreshStatuses",
		"support:DescribeTrustedAdvisorChecks",
		"support:DescribeSeverityLevels",
		"support:RefreshTrustedAdvisorCheck",
		"support:DescribeSupportLevel",
		"support:DescribeCommunications",
		"support:DescribeServices",
		"support:DescribeIssueTypes",
		"support:DescribeTrustedAdvisorCheckResult",
		"trustedadvisor:DescribeNotificationPreferences",
		"trustedadvisor:DescribeCheckRefreshStatuses",
		"trustedadvisor:DescribeCheckItems",
		"trustedadvisor:DescribeAccount",
		"trustedadvisor:DescribeAccountAccess",
		"trustedadvisor:DescribeChecks",
		"trustedadvisor:DescribeCheckSummaries",
	}

	awsFeaturePermissions := []FeaturePermissions{
		{
			FeatureName: "core",
			Permissions: awsCoreFeaturePermissions,
			Policies:    awsCorePolicies,
		}, {
			FeatureName: "quotas",
			Permissions: awsQuotasFeaturePermissions,
		},
	}

	if withSpotScaling {
		awsFeaturePermissions = append(awsFeaturePermissions, FeaturePermissions{
			FeatureName: "spot-scaling",
			Permissions: awsSpotScalingFeaturePermissions,
		})
	}

	// updates cloud-connect with new awsFeaturePermissions document
	_,
		err = fs.Collection("app").Doc("cloud-connect").Set(ctx, map[string]interface{}{
		"awsFeaturePermissions": awsFeaturePermissions,
	}, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("could not update aws feature permissions. error %s", err)
	}

	if withSpotScaling {
		return addSpot0PermissionAndUpdateRoles(ctx, fs)
	}

	return nil
}

func addSpot0PermissionAndUpdateRoles(ctx context.Context, fs *firestore.Client) error {

	// adds spot0 permission
	spot0PermissionRef := fs.Collection("permissions").Doc("akRhXeDLdP3mh87dXKSt")
	_, err := spot0PermissionRef.Set(ctx, struct {
		Desc  string `firestore:"desc"`
		Order int    `firestore:"order"`
		Title string `firestore:"title"`
	}{
		Desc:  "Manage AWS auto-scaling groups",
		Order: 101,
		Title: "Spot Scaling Manager",
	})
	if err != nil {
		return fmt.Errorf("could not update permissions collection with spot0 permission. error %s", err)
	}

	var adminRolePermissions struct {
		Permissions []*firestore.DocumentRef
	}
	oldAdminRole,
	err := fs.Collection("roles").Doc("59w2TJPTCa3XPsJ3KITY").Get(ctx)
	if err != nil {
		return fmt.Errorf("could not get admin role. error %s", err)
	}

	err = oldAdminRole.DataTo(&adminRolePermissions)
	if err != nil {
		return fmt.Errorf("could not unmarshal admin role. error %s", err)
	}

	adminRolePermissions.Permissions = append(adminRolePermissions.Permissions, spot0PermissionRef)

	// update admin role
	_, err = fs.Collection("roles").Doc("59w2TJPTCa3XPsJ3KITY").Set(ctx, map[string]interface{}{
		"permissions": adminRolePermissions.Permissions,
	}, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("could not insert spot0 permissions into admin role. error %s", err)
	}

	var powerUserRolePermissions struct {
		Permissions []*firestore.DocumentRef
	}
	oldPowerUserRole, err := fs.Collection("roles").Doc("Y2IN1X2rmWoZhTJDsYAN").Get(ctx)
	if err != nil {
		return fmt.Errorf("could not get power user role. error %s", err)
	}

	err = oldPowerUserRole.DataTo(&powerUserRolePermissions)
	if err != nil {
		return fmt.Errorf("could not unmarshal admin role. error %s", err)
	}

	powerUserRolePermissions.Permissions = append(powerUserRolePermissions.Permissions, spot0PermissionRef)

	// update power user role
	_, err = fs.Collection("roles").Doc("Y2IN1X2rmWoZhTJDsYAN").Set(ctx, map[string]interface{}{
		"permissions": powerUserRolePermissions.Permissions,
	}, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("could not insert spot0 permissions into power user role. error %s", err)
	}

	return nil
}
*/
