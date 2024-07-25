package mpa

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/api/googleapi"

	"github.com/doitintl/googleadmin"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	admin "google.golang.org/api/admin/directory/v1"
	groupssettings "google.golang.org/api/groupssettings/v1"
)

const (
	awsOpsGroup               = "awsops@doit-intl.com"
	createGoogleGroupRoute    = "/tasks/amazon-web-services/master-payer-accounts/google-group/create"
	errorWrongParamString     = "google group was created with wrong %s: %s"
	errorRequestMissingString = "request is missing parameter: %s"
	errorAlreadyExist         = "Entity already exists."

	groupssettingsAllMembers    = "ALL_MEMBERS"
	groupssettingsAnyoneCanPost = "ANYONE_CAN_POST"
	groupssettingsTrue          = "true"
	groupssettingsAllow         = "ALLOW"
	groupssettingsModerateNone  = "MODERATE_NONE"
)

var (
	errorRequestFailed = fmt.Errorf("google group creation failed")
	errorMembers       = fmt.Errorf("failed to add member [%s] to google group", awsOpsGroup)
	errorMissingDomain = fmt.Errorf(errorRequestMissingString, "domain")
	errorMissingEmail  = fmt.Errorf(errorRequestMissingString, "root email")
)

// CreateGoogleGroup creates an MPA related google group which includes awsops@doit-intl.com
func (s *MasterPayerAccountService) CreateGoogleGroup(ctx context.Context, req *MPAGoogleGroup) error {
	logger := s.getLogger(ctx, req.Domain)
	logger.Printf("creating MPA google group for domain: [%s], email: [%s]\n", req.Domain, req.RootEmail)

	req.RootEmail = clearAliases(req.RootEmail)
	if err := validateRequest(req); err != nil {
		return err
	}

	description := getGroupDescription(req.Domain)
	name := getGroupName(req.RootEmail)
	group := googleadmin.CreateGroupObject(name, req.RootEmail, description)
	member := googleadmin.CreateGroupMemberObject(awsOpsGroup)

	groupRes, err := s.googleAdmin.CreateGroupWithMembers(group, []*admin.Member{member})
	if err != nil {
		return handleUpdateError(err, req.RootEmail, "create")
	}

	if err := validateGroupFields(groupRes, req.RootEmail, name); err != nil {
		return err
	}

	groupSettings, err := s.AdjustGoogleGroupSettings(groupRes.Email)
	if err != nil {
		return err
	}

	return validateGroupSettings(groupSettings)
}

// AdjustGoogleGroupSettings - update settings as for MPA groups requirements
func (s *MasterPayerAccountService) AdjustGoogleGroupSettings(email string) (*groupssettings.Groups, error) {
	email = clearAliases(email)

	groupSettings, err := s.googleAdmin.GetGroupSettings(email)
	if err != nil {
		return nil, err
	}

	groupSettings.AllowExternalMembers = groupssettingsTrue
	groupSettings.WhoCanModerateMembers = groupssettingsAllMembers
	groupSettings.WhoCanPostMessage = groupssettingsAnyoneCanPost
	groupSettings.SpamModerationLevel = groupssettingsAllow
	groupSettings.MessageModerationLevel = groupssettingsModerateNone

	groupSettingsRes, err := s.googleAdmin.UpdateGroupSettings(email, groupSettings)
	if err != nil {
		return nil, err
	}

	return groupSettingsRes, nil
}

// UpdateGoogleGroup  updates existing google group with new email & domain
func (s *MasterPayerAccountService) UpdateGoogleGroup(ctx context.Context, req *MPAGoogleGroupUpdate) error {
	logger := s.getLogger(ctx, req.Domain)
	logger.Printf("updating MPA google group for domain: [%s], email: [%s], current email: [%s]\n", req.Domain, req.RootEmail, req.CurrentRootEmail)

	req.CurrentRootEmail = clearAliases(req.CurrentRootEmail)
	req.RootEmail = clearAliases(req.RootEmail)

	if len(req.CurrentRootEmail) == 0 {
		return errorMissingEmail
	}

	if err := validateRequest(&req.MPAGoogleGroup); err != nil {
		return err
	}

	group, err := s.googleAdmin.GetGroup(req.CurrentRootEmail)
	if err != nil {
		return err
	}

	group.Email = req.RootEmail
	group.Name = getGroupName(req.RootEmail)
	group.Description = getGroupDescription(req.Domain)

	_, err = s.googleAdmin.UpdateGroup(req.CurrentRootEmail, group)
	if err != nil {
		return handleUpdateError(err, req.RootEmail, "update")
	}

	updatedGroup, err := s.googleAdmin.GetGroup(req.RootEmail) //	call get group to verify changes did apply
	if err != nil {
		return err
	}

	return validateGroupFields(updatedGroup, req.RootEmail, group.Name)
}

// DeleteGoogleGroup deletes google group in case no MPA is bind to it
func (s *MasterPayerAccountService) DeleteGoogleGroup(ctx context.Context, req *MPAGoogleGroup) error {
	logger := s.getLogger(ctx, req.Domain)
	logger.Printf("deleting MPA google group for domain: [%s], email: [%s]\n", req.Domain, req.RootEmail)

	req.RootEmail = clearAliases(req.RootEmail)
	if err := validateRequest(req); err != nil {
		return err
	}

	masterPayerAccountsForDomain, err := s.mpaDAL.GetMasterPayerAccountsForDomain(ctx, req.Domain)
	if err != nil {
		return err
	}

	if len(masterPayerAccountsForDomain) != 0 {
		logger.Printf("domain [%s] has %d more MPAs which are in use. aborting deletion of google group for email: [%s]\n", req.Domain, len(masterPayerAccountsForDomain), req.RootEmail)
		return nil
	}

	return s.googleAdmin.DeleteGroup(req.RootEmail)
}

// CreateGoogleGroupCloudTask create cloud task for google group creation (required due to 1 minute invocation duration)
func (s *MasterPayerAccountService) CreateGoogleGroupCloudTask(ctx context.Context, req *MPAGoogleGroup) error {
	logger := s.getLogger(ctx, req.Domain)
	logger.Printf("creating cloud task for MPA google group creation. domain: [%s], email: [%s]\n", req.Domain, req.RootEmail)

	req.RootEmail = clearAliases(req.RootEmail)
	if err := validateRequest(req); err != nil {
		return err
	}

	if err := s.checkGoogleGroupExistence(req.RootEmail); err != nil { //	check existence before initiating a cloud-task for creation
		if err.Error() == errorAlreadyExist {
			logger.Printf("MPA google group already exist for domain: [%s], email: [%s]. aborting request\n", req.Domain, req.RootEmail)
			return nil
		}

		return err
	}

	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   createGoogleGroupRoute,
		Queue:  common.TaskQueueMPAGoogleGroup,
	}
	conf := config.Config(req)

	if _, err := s.cloudTaskClient.CreateTask(ctx, conf); err != nil {
		return err
	}

	return nil
}

// checkGoogleGroupExistence check existence of a group for a given rootEmail
func (s *MasterPayerAccountService) checkGoogleGroupExistence(rootEmail string) error {
	group, err := s.googleAdmin.GetGroup(rootEmail)
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok { //	group does not exist
			if gapiErr.Code == http.StatusNotFound && strings.Contains(gapiErr.Message, "Resource Not Found") {
				return nil
			}
		}

		return err //	error interacting with google api
	}

	if group != nil { //	group does exist
		return fmt.Errorf(errorAlreadyExist)
	}

	return nil
}

func (s *MasterPayerAccountService) getLogger(ctx context.Context, domain string) logger.ILogger {
	l := s.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		"domain":  domain,
		"service": "mpa",
		"flow":    "google-group",
	})

	return l
}

func (s *MasterPayerAccountService) GetMasterPayerAccountByAccountNumber(ctx context.Context, accountNumber string) (*domain.MasterPayerAccount, error) {
	return s.mpaDAL.GetMasterPayerAccountByAccountNumber(ctx, accountNumber)
}

func validateRequest(req *MPAGoogleGroup) error {
	if len(req.Domain) == 0 {
		return errorMissingDomain
	}

	if len(req.RootEmail) == 0 {
		return errorMissingEmail
	}

	return nil
}

func validateGroupFields(group *admin.Group, email, name string) error {
	if group.ServerResponse.HTTPStatusCode != 200 {
		return errorRequestFailed
	}

	if group.Email != email {
		return getWrongParamError("email", group.Email)
	}

	if group.Name != name {
		return getWrongParamError("name", group.Name)
	}

	if group.DirectMembersCount < 1 {
		return errorMembers
	}

	return nil
}

func validateGroupSettings(groupSettings *groupssettings.Groups) error {
	if groupSettings.AllowExternalMembers != groupssettingsTrue {
		return getWrongParamError("AllowExternalMembers", groupSettings.AllowExternalMembers)
	}

	if groupSettings.WhoCanModerateMembers != groupssettingsAllMembers {
		return getWrongParamError("WhoCanModerateMembers", groupSettings.WhoCanModerateMembers)
	}

	if groupSettings.WhoCanPostMessage != groupssettingsAnyoneCanPost {
		return getWrongParamError("WhoCanPostMessage", groupSettings.WhoCanPostMessage)
	}

	if groupSettings.SpamModerationLevel != groupssettingsAllow {
		return getWrongParamError("SpamModerationLevel", groupSettings.SpamModerationLevel)
	}

	if groupSettings.MessageModerationLevel != groupssettingsModerateNone {
		return getWrongParamError("MessageModerationLevel", groupSettings.MessageModerationLevel)
	}

	return nil
}

func getGroupName(email string) string {
	replacer := strings.NewReplacer("@doit-intl.com", "", "@doit.com", "")
	return replacer.Replace(email)
}

func getGroupDescription(domain string) string {
	return fmt.Sprintf("This group is used for root email credentials for %s", domain)
}

func getWrongParamError(param, value string) error {
	return fmt.Errorf(errorWrongParamString, param, value)
}

func handleUpdateError(err error, email, action string) error {
	if gapiErr, ok := err.(*googleapi.Error); ok { //	 root email is taken
		if gapiErr.Code == http.StatusConflict && gapiErr.Message == errorAlreadyExist {
			return fmt.Errorf("email [%s] is already used for a different google group. %s cannot be done. error: [%w]", email, action, err)
		}
	}

	return err
}

func clearAliases(email string) string {
	aliasRegex := regexp.MustCompile("\\+([1-9]+)")
	return aliasRegex.ReplaceAllString(email, "")
}
