package googlecloud

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	cloudbilling "google.golang.org/api/cloudbilling/v1"
	cloudresourcemanagerv1 "google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
	"google.golang.org/api/option"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

var runeSet = []rune("0123456789abcdefghijklmnopqrstuvwxyz")

func randomSequenceN(n int) string {
	rand.Seed(time.Now().UnixNano())

	suffix := make([]rune, n)
	for i := range suffix {
		suffix[i] = runeSet[rand.Intn(len(runeSet))]
	}

	return string(suffix)
}

var appEngineCredsJSON []byte

func init() {
	ctx := context.Background()

	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretAppEngine)
	if err != nil {
		log.Fatalln(err)
	}

	appEngineCredsJSON = data
}

func CreateProject(ctx context.Context, l logger.ILogger, clientOptions option.ClientOption, policy *SandboxPolicy) (*cloudresourcemanagerv1.Project, error) {
	crmv1, err := cloudresourcemanagerv1.NewService(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	crmv2, err := cloudresourcemanagerv2.NewService(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	projectID := fmt.Sprintf("%s-%s-%s", policy.NamePrefix, now.Format("20060102"), randomSequenceN(6))
	l.Info(projectID)

	var parent *cloudresourcemanagerv1.ResourceId

	if policy.Folder != nil {
		folder, err := crmv2.Folders.Get(policy.Folder.Name).Do()
		if err != nil {
			return nil, err
		}

		l.Info(marshalJSON(folder))

		// if folder.Parent != policy.Organization.Name {
		// 	err := errors.New("could not find folder in organization")
		// 	return nil, err
		// }
		parent = &cloudresourcemanagerv1.ResourceId{
			Id:   folder.Name[8:],
			Type: "folder",
		}
	} else {
		parent = &cloudresourcemanagerv1.ResourceId{
			Id:   policy.Organization.Name[14:],
			Type: "organization",
		}
	}

	projectCreateReq := &cloudresourcemanagerv1.Project{
		ProjectId: projectID,
		Name:      projectID,
		Parent:    parent,
	}

	operation, err := crmv1.Projects.Create(projectCreateReq).Do()
	if err != nil {
		return nil, err
	}

	l.Info(marshalJSON(operation))

	for !operation.Done {
		time.Sleep(2 * time.Second)

		op, err := crmv1.Operations.Get(operation.Name).Do()
		if err != nil {
			return nil, err
		}

		l.Info(marshalJSON(op))

		operation = op
	}

	if operation.Error != nil {
		return nil, fmt.Errorf("%d: %s", operation.Error.Code, operation.Error.Message)
	}

	project, err := crmv1.Projects.Get(projectID).Do()
	if err != nil {
		return nil, err
	}

	return project, nil
}

func AddProjectOwner(ctx context.Context, clientOptions option.ClientOption, projectID string, member string) error {
	l := logger.FromContext(ctx)

	options := clientOptions
	if clientOptions == nil {
		options = option.WithCredentialsJSON(appEngineCredsJSON)
	}

	crmv1, err := cloudresourcemanagerv1.NewService(ctx, options)
	if err != nil {
		return err
	}

	policy, err := crmv1.Projects.GetIamPolicy(projectID, &cloudresourcemanagerv1.GetIamPolicyRequest{}).Do()
	if err != nil {
		return err
	}

	l.Infof("original policy: %s", marshalJSON(policy))

	if strings.HasPrefix(member, "user:") {
		// role = "roles/resourcemanager.projectOwnerInvitee"
		policy.Bindings = append(policy.Bindings, &cloudresourcemanagerv1.Binding{
			Members: []string{member},
			Role:    "roles/editor",
		})
		policy.Bindings = append(policy.Bindings, &cloudresourcemanagerv1.Binding{
			Members: []string{member},
			Role:    "roles/resourcemanager.projectIamAdmin",
		})
	} else {
		policy.Bindings = append(policy.Bindings, &cloudresourcemanagerv1.Binding{
			Members: []string{member},
			Role:    "roles/owner",
		})
	}

	setPolicyReq := &cloudresourcemanagerv1.SetIamPolicyRequest{
		Policy: policy,
	}

	l.Infof("set policy request: %s", marshalJSON(setPolicyReq))

	updatedPolicy, err := crmv1.Projects.SetIamPolicy(projectID, setPolicyReq).Do()
	if err != nil {
		return err
	}

	l.Infof("updated policy: %s", marshalJSON(updatedPolicy))

	return nil
}

func UpdateProjectBillingInfo(ctx context.Context, billingAccount *cloudbilling.BillingAccount, project *cloudresourcemanagerv1.Project) (*cloudbilling.ProjectBillingInfo, error) {
	if billingAccount == nil {
		err := errors.New("billingAccount is nil")
		return nil, err
	}

	creds := option.WithCredentialsJSON(appEngineCredsJSON)

	cb, err := cloudbilling.NewService(ctx, creds)
	if err != nil {
		return nil, err
	}

	projectResourceName := "projects/" + project.ProjectId
	projectBillingInfo := &cloudbilling.ProjectBillingInfo{
		BillingAccountName: billingAccount.Name,
	}

	updatedProjectBillingInfo, err := cb.Projects.UpdateBillingInfo(projectResourceName, projectBillingInfo).Do()
	if err != nil {
		return nil, err
	}

	return updatedProjectBillingInfo, nil
}

func DisableProjectBilling(ctx context.Context, project *cloudresourcemanagerv1.Project) (*cloudbilling.ProjectBillingInfo, error) {
	creds := option.WithCredentialsJSON(appEngineCredsJSON)

	cb, err := cloudbilling.NewService(ctx, creds)
	if err != nil {
		return nil, err
	}

	projectResourceName := "projects/" + project.ProjectId
	projectBillingInfo := &cloudbilling.ProjectBillingInfo{
		BillingAccountName: "",
	}

	updatedProjectBillingInfo, err := cb.Projects.UpdateBillingInfo(projectResourceName, projectBillingInfo).Do()
	if err != nil {
		return nil, err
	}

	return updatedProjectBillingInfo, nil
}
