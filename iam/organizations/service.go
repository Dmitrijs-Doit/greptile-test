package organizations

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

type OrgsIAMService struct {
	*logger.Logging
	*connection.Connection
}

var (
	ErrForbidden           = errors.New("forbidden")
	ErrInternalServerError = errors.New("internal server error")
	ErrNotFound            = errors.New("organizations not found")
	ErrCustomerMissing     = errors.New("customer id not found")
	ErrUserNotFound        = errors.New("user not found")
)

const (
	RootOrgID       = "root"
	PresetGCPOrgID  = "GCPOrg"
	PresetAWSOrgID  = "AWSOrg"
	PresetDoitOrgID = "doit-international"
)

var (
	PresetOrgAssetTypes = map[string][]string{
		PresetGCPOrgID: {common.Assets.GoogleCloud, common.Assets.GoogleCloudStandalone},
		PresetAWSOrgID: {common.Assets.AmazonWebServices, common.Assets.AmazonWebServicesStandalone},
	}
)

func IsRootOrg(orgID string) bool {
	return orgID == RootOrgID
}

func IsPresetOrg(orgID string) bool {
	return orgID == PresetGCPOrgID || orgID == PresetAWSOrgID || orgID == PresetDoitOrgID
}

func IsPartnerOrg(orgID string) bool {
	return orgID == PresetGCPOrgID || orgID == PresetAWSOrgID
}

func GetDoitOrgRef(fs *firestore.Client) *firestore.DocumentRef {
	return fs.Collection("organizations").Doc(PresetDoitOrgID)
}

func GetPresetOrgCloudProviderRestriction(orgID string) []string {
	switch orgID {
	case PresetGCPOrgID, PresetAWSOrgID:
		return PresetOrgAssetTypes[orgID]
	default:
		return nil
	}
}

func IsOrgAllowedAssetType(orgID, assetType string) bool {
	if IsPresetOrg(orgID) && assetType != "" {
		return slice.Contains(GetPresetOrgCloudProviderRestriction(orgID), assetType)
	}

	return true
}

func NewIAMOrganizationService(log *logger.Logging, conn *connection.Connection) *OrgsIAMService {
	return &OrgsIAMService{
		log,
		conn,
	}
}

func (s *OrgsIAMService) DeleteIAMOrgs(ctx context.Context, req *RemoveIAMOrgsRequest) error {
	fs := s.Firestore(ctx)
	// validate user if not doitEmployee
	if v, ok := ctx.Value(common.DoitEmployee).(bool); ok && !v {
		user, err := s.GetCurrentUser(ctx, fs, req.UserID)
		if err != nil {
			return err
		}

		if !user.HasUsersPermission(ctx) {
			return ErrForbidden
		}
	}

	// Delete orgs from firestore and from users
	if err := s.handleDeleteOrgs(ctx, fs, req); err != nil {
		return s.HandleErrors(err)
	}

	return nil
}

func (s *OrgsIAMService) HandleErrors(err error) error {
	switch status.Code(err) {
	case codes.NotFound:
		return ErrNotFound
	case codes.PermissionDenied:
		return ErrForbidden
	default:
		return ErrInternalServerError
	}
}
