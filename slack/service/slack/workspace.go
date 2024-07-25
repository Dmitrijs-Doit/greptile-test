package slack

import (
	"context"

	firestorePkg "github.com/doitintl/firestore/pkg"
)

// GetCustomerWorkspaceDecrypted - given customerID returns parsed workspace, ID, decrypted bot token, decrypted user token
func (s *SlackService) GetWorkspaceDecrypted(ctx context.Context, customerID string) (*firestorePkg.SlackWorkspace, string, string, string, error) {
	return s.firestoreDAL.GetCustomerWorkspaceDecrypted(ctx, customerID)
}

func (s *SlackService) isCustomer(ctx context.Context, teamID string) (bool, string, error) {
	workspace, _, _, _, err := s.firestoreDAL.GetWorkspaceDecrypted(ctx, teamID)
	if err != nil {
		return false, "", err
	}

	if !workspace.Authenticated {
		return false, "", nil
	}

	return workspace.Authenticated, workspace.Customer.ID, nil
}
