package mpa

import (
	"context"
	"fmt"
	defaultHttp "net/http"
	"testing"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/costandusagereportservice"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	cloudtasksMock "github.com/doitintl/cloudtasks/mocks"
	googleAdminMock "github.com/doitintl/googleadmin/mocks"
	mpaDalMock "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	mpaDal "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	awsMock "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/http"
	httpMock "github.com/doitintl/http/mocks"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/mock"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/groupssettings/v1"
)

func TestMPAService_ValidateMPA(t *testing.T) {
	type fields struct {
		Logger *loggerMocks.ILogger
		AwsDAL *awsMock.IAWSDal
	}

	type args struct {
		ctx context.Context
		req *ValidateMPARequest
	}

	reportOrCsv := costandusagereportservice.ReportFormatTextOrcsv

	var (
		accountID                  = "111111111111"
		roleArn                    = "arn:aws:iam::111111111111:role/doitintl_cmp"
		policyArn                  = "arn:aws:iam::111111111111:policy/doitintl_cmp"
		policyVersion              = "1"
		curBucket                  = "doitintl-awsops-111"
		curPath                    = "CUR/doitintl-awsops-111"
		doitPolicyTemplateFileTest = "../../scripts/data/doitintl_cmp.json"

		hourly    = costandusagereportservice.TimeUnitHourly
		gzip      = costandusagereportservice.CompressionFormatGzip
		resources = costandusagereportservice.SchemaElementResources
	)

	req := &ValidateMPARequest{
		AccountID: accountID,
		RoleArn:   roleArn,
		CURBucket: curBucket,
		CURPath:   curPath,
	}
	validDescribeReportDefinitions := costandusagereportservice.DescribeReportDefinitionsOutput{
		ReportDefinitions: []*costandusagereportservice.ReportDefinition{
			{
				S3Bucket:                 &curBucket,
				S3Prefix:                 &curBucket,
				ReportName:               &curBucket,
				Compression:              &gzip,
				TimeUnit:                 &hourly,
				Format:                   &reportOrCsv,
				AdditionalSchemaElements: []*string{&resources},
			},
		},
	}
	badPath := "badPath"
	invalidDescribeReportDefinitions := costandusagereportservice.DescribeReportDefinitionsOutput{
		ReportDefinitions: []*costandusagereportservice.ReportDefinition{
			{
				S3Bucket:                 &badPath,
				S3Prefix:                 &badPath,
				ReportName:               &curBucket,
				Compression:              &gzip,
				TimeUnit:                 &hourly,
				Format:                   &reportOrCsv,
				AdditionalSchemaElements: []*string{&resources},
			},
		},
	}
	validRole := iam.Role{
		Arn:      &roleArn,
		RoleName: &doitRole,
	}
	validGetRoleOutput := iam.GetRoleOutput{
		Role: &validRole,
	}
	validGetPolicyOutput := iam.GetPolicyOutput{
		Policy: &iam.Policy{
			Arn:              &policyArn,
			DefaultVersionId: &policyVersion,
			PolicyName:       &doitPolicy,
		},
	}

	policyPermissions, err := getPolicyPermissionsFromTemplate(doitPolicyTemplateFileTest, accountID, curBucket)
	if err != nil {
		t.Error(err)
	}

	policyPermissionsBytes, err := json.Marshal(policyPermissions)
	if err != nil {
		t.Error(err)
	}

	policyPermissionsString := string(policyPermissionsBytes)
	validGetPolicyVersionOutput := iam.GetPolicyVersionOutput{
		PolicyVersion: &iam.PolicyVersion{
			Document: &policyPermissionsString,
		},
	}

	awsErrAccessDenied := awserr.New(accessDeniedError, "", nil)

	validObjects := s3.ListObjectsV2Output{
		Contents: []*s3.Object{
			{
				Key: aws.String(fmt.Sprintf("%s-Manifest.json", curPath)),
			},
		},
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		args    args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		wantErr bool
	}{
		{
			name: "valid MPA",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.AwsDAL.
					On("DescribeReportDefinitions", accountID).
					Return(&validDescribeReportDefinitions, nil).
					Once().
					On("GetRole", accountID, roleArn).
					Return(&validGetRoleOutput, nil).
					Once().
					On("GetPolicy", accountID, policyArn).
					Return(&validGetPolicyOutput, nil).
					Once().
					On("GetPolicyVersion", accountID, policyArn, policyVersion).
					Return(&validGetPolicyVersionOutput, nil).
					Once()
				f.Logger.
					On("SetLabels", mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.AwsDAL.AssertNumberOfCalls(t, "DescribeReportDefinitions", 1)
				f.AwsDAL.AssertNotCalled(t, "ListObjectsV2")
				f.AwsDAL.AssertNumberOfCalls(t, "GetRole", 1)
				f.AwsDAL.AssertNumberOfCalls(t, "GetPolicy", 1)
				f.AwsDAL.AssertNumberOfCalls(t, "GetPolicyVersion", 1)
			},
			wantErr: false,
		},
		{
			name: "valid MPA fallback (no cur:DescribeReportDefinitions permission)",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.AwsDAL.
					On("DescribeReportDefinitions", accountID).
					Return(nil, awsErrAccessDenied).
					Once().
					On("ListObjectsV2", accountID, curBucket).
					Return(&validObjects, nil).
					Once().
					On("GetRole", accountID, roleArn).
					Return(&validGetRoleOutput, nil).
					Once().
					On("GetPolicy", accountID, policyArn).
					Return(nil, awsErrAccessDenied).
					Once().
					On("GetPolicyVersion", accountID, policyArn, policyVersion).
					Return(&validGetPolicyVersionOutput, nil).
					Once()
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything).
					Twice()
			},
			assert: func(t *testing.T, f *fields) {
				f.AwsDAL.AssertNumberOfCalls(t, "DescribeReportDefinitions", 1)
				f.AwsDAL.AssertNumberOfCalls(t, "ListObjectsV2", 1)
				f.AwsDAL.AssertNumberOfCalls(t, "GetRole", 1)
				f.AwsDAL.AssertNumberOfCalls(t, "GetPolicy", 1)
				f.AwsDAL.AssertNotCalled(t, "GetPolicyVersion")
			},
			wantErr: false,
		},
		{
			name: "invalid MPA - bad cur",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.AwsDAL.
					On("DescribeReportDefinitions", accountID).
					Return(&invalidDescribeReportDefinitions, nil).
					Once()
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Error", mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.AwsDAL.AssertNumberOfCalls(t, "DescribeReportDefinitions", 1)
				f.AwsDAL.AssertNotCalled(t, "ListObjectsV2")
				f.AwsDAL.AssertNotCalled(t, "GetRole")
				f.AwsDAL.AssertNotCalled(t, "GetPolicy")
				f.AwsDAL.AssertNotCalled(t, "GetPolicyVersion")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger: &loggerMocks.ILogger{},
				AwsDAL: &awsMock.IAWSDal{},
			}

			s := &MasterPayerAccountService{
				loggerProvider: func(_ context.Context) logger.ILogger {
					return f.Logger
				},
				awsClient:                  f.AwsDAL,
				doitPolicyTemplateFile:     &doitPolicyTemplateFileTest,
				saasDoitRoleTemplateFile:   &saasDoitRoleTemplateFile,
				saasDoitPolicyTemplateFile: &saasDoitPolicyTemplateFile,
			}

			if tt.on != nil {
				tt.on(f)
			}

			if err := s.ValidateMPA(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("MPAService.ValidateMPA() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func TestMPAService_CreateGoogleGroupCloudTask(t *testing.T) {
	type fields struct {
		Logger          *loggerMocks.ILogger
		GoogleAdmin     *googleAdminMock.GoogleAdmin
		CloudTaskClient *cloudtasksMock.CloudTaskClient
	}

	type args struct {
		ctx context.Context
		req *MPAGoogleGroup
	}

	var (
		domain    = "test-com"
		rootEmail = "mpa.test-com@doit-intl.com"
		name      = "mpa.test-com"
	)

	req := &MPAGoogleGroup{
		Domain:    domain,
		RootEmail: rootEmail,
	}
	validGroupRes := admin.Group{
		Name:               name,
		Email:              rootEmail,
		Aliases:            []string{rootEmail},
		Description:        fmt.Sprintf("This group is used for root email credentials for %s", req.Domain),
		DirectMembersCount: 1,
		ServerResponse: googleapi.ServerResponse{
			HTTPStatusCode: 200,
		},
	}
	gapiErrorGroupNotFound := googleapi.Error{
		Code:    defaultHttp.StatusNotFound,
		Message: "Resource Not Found",
	}
	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   createGoogleGroupRoute,
		Queue:  common.TaskQueueMPAGoogleGroup,
	}
	conf := config.Config(req)

	ctx := context.Background()

	tests := []struct {
		name    string
		args    args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.GoogleAdmin.
					On("GetGroup", rootEmail).
					Return(nil, &gapiErrorGroupNotFound)
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything)
				f.CloudTaskClient.
					On("CreateTask", ctx, conf).
					Return(nil, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNumberOfCalls(t, "GetGroup", 1)
				f.CloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 1)
			},
			wantErr: false,
		},
		{
			name: "success - group already exist",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.GoogleAdmin.
					On("GetGroup", rootEmail).
					Return(&validGroupRes, nil)
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything).
					Twice()
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNumberOfCalls(t, "GetGroup", 1)
				f.CloudTaskClient.AssertNotCalled(t, "CreateTask")
				f.Logger.AssertNumberOfCalls(t, "Printf", 2)
			},
			wantErr: false,
		},
		{
			name: "failure - invalid request",
			args: args{
				ctx,
				&MPAGoogleGroup{},
			},
			on: func(f *fields) {
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNotCalled(t, "GetGroup")
				f.CloudTaskClient.AssertNotCalled(t, "CreateTask")
			},
			wantErr: true,
		},
		{
			name: "failure - get group failed",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.GoogleAdmin.
					On("GetGroup", rootEmail).
					Return(nil, fmt.Errorf("failure in google api"))
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNumberOfCalls(t, "GetGroup", 1)
				f.CloudTaskClient.AssertNotCalled(t, "CreateTask")
			},
			wantErr: true,
		},
		{
			name: "failure - cloud task creation failed",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.GoogleAdmin.
					On("GetGroup", rootEmail).
					Return(nil, &gapiErrorGroupNotFound)
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything)
				f.CloudTaskClient.
					On("CreateTask", ctx, conf).
					Return(nil, fmt.Errorf("failed to create cloud task"))
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNumberOfCalls(t, "GetGroup", 1)
				f.CloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 1)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:          &loggerMocks.ILogger{},
				GoogleAdmin:     &googleAdminMock.GoogleAdmin{},
				CloudTaskClient: &cloudtasksMock.CloudTaskClient{},
			}

			s := &MasterPayerAccountService{
				loggerProvider: func(_ context.Context) logger.ILogger {
					return f.Logger
				},
				googleAdmin:     f.GoogleAdmin,
				cloudTaskClient: f.CloudTaskClient,
			}

			if tt.on != nil {
				tt.on(f)
			}

			if err := s.CreateGoogleGroupCloudTask(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("MPAService.CreateGoogleGroupCloudTask() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func TestMPAService_CreateGoogleGroup(t *testing.T) {
	type fields struct {
		Logger          *loggerMocks.ILogger
		GoogleAdmin     *googleAdminMock.GoogleAdmin
		CloudTaskClient *cloudtasksMock.CloudTaskClient
	}

	type args struct {
		ctx context.Context
		req *MPAGoogleGroup
	}

	var (
		domain    = "test-com"
		rootEmail = "mpa.test-com@doit-intl.com"
		name      = "mpa.test-com"
	)

	req := &MPAGoogleGroup{
		Domain:    domain,
		RootEmail: rootEmail,
	}

	group := admin.Group{
		Name:        name,
		Email:       rootEmail,
		Aliases:     []string{rootEmail},
		Description: fmt.Sprintf("This group is used for root email credentials for %s", req.Domain),
	}
	member := admin.Member{
		Email: awsOpsGroup,
	}
	validGroupRes := admin.Group{
		Name:               name,
		Email:              rootEmail,
		Aliases:            []string{rootEmail},
		Description:        fmt.Sprintf("This group is used for root email credentials for %s", req.Domain),
		DirectMembersCount: 1,
		ServerResponse: googleapi.ServerResponse{
			HTTPStatusCode: 200,
		},
	}
	invalidGroupRes := admin.Group{
		Name:               name,
		Email:              rootEmail,
		Aliases:            []string{rootEmail},
		Description:        fmt.Sprintf("This group is used for root email credentials for %s", req.Domain),
		DirectMembersCount: 0,
		ServerResponse: googleapi.ServerResponse{
			HTTPStatusCode: 200,
		},
	}
	groupsSettings := groupssettings.Groups{}
	groupsSettingsRes := groupssettings.Groups{
		AllowExternalMembers:   groupssettingsTrue,
		WhoCanModerateMembers:  groupssettingsAllMembers,
		WhoCanPostMessage:      groupssettingsAnyoneCanPost,
		SpamModerationLevel:    groupssettingsAllow,
		MessageModerationLevel: groupssettingsModerateNone,
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		args    args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.GoogleAdmin.
					On("CreateGroupWithMembers", &group, []*admin.Member{&member}).
					Return(&validGroupRes, nil).
					Once().
					On("GetGroupSettings", rootEmail).
					Return(&groupsSettings, nil).
					Once().
					On("UpdateGroupSettings", rootEmail, mock.AnythingOfType("*groupssettings.Groups")).
					Return(&groupsSettingsRes, nil).
					Once()
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNumberOfCalls(t, "CreateGroupWithMembers", 1)
			},
			wantErr: false,
		},
		{
			name: "failure - bad request",
			args: args{
				ctx,
				&MPAGoogleGroup{},
			},
			on: func(f *fields) {
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNotCalled(t, "CreateGroupWithMembers")
			},
			wantErr: true,
		},
		{
			name: "failure - missing member",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.GoogleAdmin.
					On("CreateGroupWithMembers", &group, []*admin.Member{&member}).
					Return(&invalidGroupRes, nil)
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNumberOfCalls(t, "CreateGroupWithMembers", 1)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:          &loggerMocks.ILogger{},
				GoogleAdmin:     &googleAdminMock.GoogleAdmin{},
				CloudTaskClient: &cloudtasksMock.CloudTaskClient{},
			}

			s := &MasterPayerAccountService{
				loggerProvider: func(_ context.Context) logger.ILogger {
					return f.Logger
				},
				googleAdmin:     f.GoogleAdmin,
				cloudTaskClient: f.CloudTaskClient,
			}

			if tt.on != nil {
				tt.on(f)
			}

			if err := s.CreateGoogleGroup(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("MPAService.CreateGoogleGroup() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func TestMPAService_UpdateGoogleGroup(t *testing.T) {
	type fields struct {
		Logger          *loggerMocks.ILogger
		GoogleAdmin     *googleAdminMock.GoogleAdmin
		CloudTaskClient *cloudtasksMock.CloudTaskClient
	}

	type args struct {
		ctx context.Context
		req *MPAGoogleGroupUpdate
	}

	var (
		domain           = "test-com"
		rootEmail        = "mpa.test-com2@doit-intl.com"
		currentRootEmail = "mpa.test-com@doit-intl.com"
		name             = "mpa.test-com"
		nameAfterUpdate  = "mpa.test-com2"
	)

	req := &MPAGoogleGroupUpdate{
		MPAGoogleGroup: MPAGoogleGroup{
			Domain:    domain,
			RootEmail: rootEmail,
		},
		CurrentRootEmail: currentRootEmail,
	}

	group := admin.Group{
		Name:               name,
		Email:              currentRootEmail,
		Aliases:            []string{currentRootEmail},
		Description:        fmt.Sprintf("This group is used for root email credentials for %s", req.Domain),
		DirectMembersCount: 1,
	}
	validGroupAfterUpdate := admin.Group{
		Name:               nameAfterUpdate,
		Email:              rootEmail,
		Aliases:            []string{rootEmail, currentRootEmail},
		Description:        fmt.Sprintf("This group is used for root email credentials for %s", req.Domain),
		DirectMembersCount: 1,
		ServerResponse: googleapi.ServerResponse{
			HTTPStatusCode: 200,
		},
	}
	invalidGroupAfterUpdate := admin.Group{
		Name:               name,
		Email:              currentRootEmail,
		Aliases:            []string{currentRootEmail},
		Description:        fmt.Sprintf("This group is used for root email credentials for %s", req.Domain),
		DirectMembersCount: 1,
		ServerResponse: googleapi.ServerResponse{
			HTTPStatusCode: 200,
		},
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		args    args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.GoogleAdmin.
					On("GetGroup", currentRootEmail).
					Return(&group, nil).
					Once().
					On("UpdateGroup", currentRootEmail, mock.AnythingOfType("*admin.Group")).
					Return(nil, nil).
					On("GetGroup", rootEmail).
					Return(&validGroupAfterUpdate, nil)
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNumberOfCalls(t, "GetGroup", 2)
				f.GoogleAdmin.AssertNumberOfCalls(t, "UpdateGroup", 1)
			},
			wantErr: false,
		},
		{
			name: "failure - bad request",
			args: args{
				ctx,
				&MPAGoogleGroupUpdate{},
			},
			on: func(f *fields) {
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNotCalled(t, "GetGroup")
				f.GoogleAdmin.AssertNotCalled(t, "UpdateGroup")
			},
			wantErr: true,
		},
		{
			name: "failure - group was not updated",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.GoogleAdmin.
					On("GetGroup", currentRootEmail).
					Return(&group, nil).
					Once().
					On("UpdateGroup", currentRootEmail, mock.AnythingOfType("*admin.Group")).
					Return(nil, nil).
					On("GetGroup", rootEmail).
					Return(&invalidGroupAfterUpdate, nil)
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNumberOfCalls(t, "GetGroup", 2)
				f.GoogleAdmin.AssertNumberOfCalls(t, "UpdateGroup", 1)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:          &loggerMocks.ILogger{},
				GoogleAdmin:     &googleAdminMock.GoogleAdmin{},
				CloudTaskClient: &cloudtasksMock.CloudTaskClient{},
			}

			s := &MasterPayerAccountService{
				loggerProvider: func(_ context.Context) logger.ILogger {
					return f.Logger
				},
				googleAdmin:     f.GoogleAdmin,
				cloudTaskClient: f.CloudTaskClient,
			}

			if tt.on != nil {
				tt.on(f)
			}

			if err := s.UpdateGoogleGroup(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("MPAService.UpdateGoogleGroup() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func TestMPAService_DeleteGoogleGroup(t *testing.T) {
	type fields struct {
		Logger          *loggerMocks.ILogger
		GoogleAdmin     *googleAdminMock.GoogleAdmin
		CloudTaskClient *cloudtasksMock.CloudTaskClient
		MpaDal          *mpaDalMock.MasterPayerAccounts
	}

	type args struct {
		ctx context.Context
		req *MPAGoogleGroup
	}

	var (
		rootEmail = "mpa.test-com@doit-intl.com"
		domain    = "test.com"
	)

	req := &MPAGoogleGroup{
		RootEmail: rootEmail,
		Domain:    domain,
	}

	mpas := []*mpaDal.MasterPayerAccount{
		{
			RootEmail: &rootEmail,
		},
	}

	var mpasEmpty []*mpaDal.MasterPayerAccount

	ctx := context.Background()

	tests := []struct {
		name    string
		args    args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.MpaDal.
					On("GetMasterPayerAccountsForDomain", ctx, domain).
					Return(mpasEmpty, nil)
				f.GoogleAdmin.
					On("DeleteGroup", rootEmail).
					Return(nil).
					Once()
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNumberOfCalls(t, "DeleteGroup", 1)
				f.MpaDal.AssertNumberOfCalls(t, "GetMasterPayerAccountsForDomain", 1)
			},
			wantErr: false,
		},
		{
			name: "success - not deleted due to other MPAs for the given domain",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.MpaDal.
					On("GetMasterPayerAccountsForDomain", ctx, domain).
					Return(mpas, nil)
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Twice()
			},
			assert: func(t *testing.T, f *fields) {
				f.MpaDal.AssertNumberOfCalls(t, "GetMasterPayerAccountsForDomain", 1)
				f.Logger.AssertNumberOfCalls(t, "Printf", 2)
				f.GoogleAdmin.AssertNotCalled(t, "DeleteGroup")
			},
			wantErr: false,
		},
		{
			name: "failure - bad request",
			args: args{
				ctx,
				&MPAGoogleGroup{},
			},
			on: func(f *fields) {
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNotCalled(t, "DeleteGroup")
				f.MpaDal.AssertNotCalled(t, "GetMasterPayerAccountsForDomain")
			},
			wantErr: true,
		},
		{
			name: "failure - group was not deleted",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.MpaDal.
					On("GetMasterPayerAccountsForDomain", ctx, domain).
					Return(mpasEmpty, nil)
				f.GoogleAdmin.
					On("DeleteGroup", rootEmail).
					Return(fmt.Errorf("group was not deleted")).
					Once()
				f.Logger.
					On("SetLabels", mock.Anything).
					On("Printf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.GoogleAdmin.AssertNumberOfCalls(t, "DeleteGroup", 1)
				f.MpaDal.AssertNumberOfCalls(t, "GetMasterPayerAccountsForDomain", 1)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:          &loggerMocks.ILogger{},
				GoogleAdmin:     &googleAdminMock.GoogleAdmin{},
				CloudTaskClient: &cloudtasksMock.CloudTaskClient{},
				MpaDal:          &mpaDalMock.MasterPayerAccounts{},
			}

			s := &MasterPayerAccountService{
				loggerProvider: func(_ context.Context) logger.ILogger {
					return f.Logger
				},
				mpaDAL:          f.MpaDal,
				googleAdmin:     f.GoogleAdmin,
				cloudTaskClient: f.CloudTaskClient,
			}

			if tt.on != nil {
				tt.on(f)
			}

			if err := s.DeleteGoogleGroup(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("MPAService.DeleteGoogleGroup() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func TestMasterPayerAccountService_LinkMpaToSauron(t *testing.T) {
	type fields struct {
		sauronClient *httpMock.IClient
		clientKeys   *clientKeys
	}

	type args struct {
		ctx  context.Context
		data *LinkMpaToSauronData
	}

	ctx := context.Background()

	data := &LinkMpaToSauronData{
		AccountNumber: "12345",
		Email:         "person@example.com",
		Name:          "mpaName",
	}

	APIKey := "test-api-key-for-sauron"
	headers := map[string]string{
		http.ContentType:         http.ApplicationJSON,
		http.AuthorizationHeader: APIKey,
	}
	httpRequest := &http.Request{URL: "cmp_payer/", CustomHeaders: headers, Payload: data}

	tests := []struct {
		name    string
		fields  fields
		args    args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		wantErr bool
	}{
		{
			name: "Calls sauron with info",
			args: args{ctx, data},
			on: func(f *fields) {
				f.sauronClient.
					On("Post", ctx, httpRequest).
					Return(&http.Response{StatusCode: 201}, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.sauronClient.AssertNumberOfCalls(t, "Post", 1)
			},
			wantErr: false,
		},
		{
			name: "sauron client call fails with error",
			args: args{ctx, data},
			on: func(f *fields) {
				f.sauronClient.
					On("Post", ctx, httpRequest).
					Return(nil, http.ErrInternalServerError).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.sauronClient.AssertNumberOfCalls(t, "Post", 1)
			},
			wantErr: true,
		},
		{
			name: "sauron client call succeeds but with bad status",
			args: args{ctx, data},
			on: func(f *fields) {
				f.sauronClient.
					On("Post", ctx, httpRequest).
					Return(&http.Response{StatusCode: 403}, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.sauronClient.AssertNumberOfCalls(t, "Post", 1)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{sauronClient: &httpMock.IClient{}, clientKeys: &clientKeys{SauronApiKey: APIKey}}
			s := &MasterPayerAccountService{
				sauronClient: f.sauronClient,
				clientKeys:   f.clientKeys,
			}

			if tt.on != nil {
				tt.on(f)
			}

			if err := s.LinkMpaToSauron(tt.args.ctx, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("LinkMpaToSauron() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
