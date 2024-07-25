package doitemployees

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

type doitRole struct {
	RoleName    string   `firestore:"roleName"`
	Description string   `firestore:"description"`
	Users       []string `firestore:"users"`
}

type Service struct {
	*connection.Connection
}

type UserDetails struct {
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	CustomerID  string `json:"customerId,omitempty"`
}

//go:generate mockery --name ServiceInterface --output ./mocks
type ServiceInterface interface {
	GetByID(ctx context.Context, userID string) (*UserDetails, error)
	CheckDoiTEmployeeRole(ctx context.Context, roleName string, email string) (bool, error)
	SyncRole(ctx context.Context, roleName string, users []string) error
	IsDoitEmployee(ctx context.Context) bool
}

func NewService(conn *connection.Connection) ServiceInterface {
	return &Service{conn}
}

func (s *Service) GetByID(ctx context.Context, ID string) (*UserDetails, error) {
	fs := s.Firestore(ctx)

	snap, err := fs.Collection("app").Doc("doit-employees").Collection("doitEmployees").Doc(ID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var userDetails UserDetails
	if err := snap.DataTo(&userDetails); err != nil {
		return nil, err
	}

	return &userDetails, nil
}

func (s *Service) CheckDoiTEmployeeRole(ctx context.Context, roleName string, email string) (bool, error) {
	fs := s.Firestore(ctx)

	doitRolesDocs, err := fs.Collection("app/doit-employees/doitRoles").Where("roleName", "==", roleName).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return false, err
	}

	if len(doitRolesDocs) == 0 {
		return false, nil
	}

	var role doitRole

	if err := doitRolesDocs[0].DataTo(&role); err != nil {
		return false, err
	}

	if len(role.Users) == 0 {
		return false, nil
	}

	for _, user := range role.Users {
		if user == email {
			return true, nil
		}
	}

	return false, nil
}

// SyncRole - appends list of users to a doit role if they do not exist, also keep existing users
func (s *Service) SyncRole(ctx context.Context, roleName string, users []string) error {
	fs := s.Firestore(ctx)

	doitRolesDocs, err := fs.Collection("app/doit-employees/doitRoles").Where("roleName", "==", roleName).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	if len(doitRolesDocs) == 0 {
		return fmt.Errorf("doitRole [%s] not found", roleName)
	}

	doc := doitRolesDocs[0]

	var role doitRole
	if err := doc.DataTo(&role); err != nil {
		return err
	}

	existingUsersMap := map[string]bool{}
	for _, existingUser := range role.Users {
		existingUsersMap[existingUser] = true
	}

	for _, user := range users {
		if !existingUsersMap[user] {
			role.Users = append(role.Users, user)
		}
	}

	_, err = doc.Ref.Set(ctx, role)

	return err
}

func (s *Service) IsDoitEmployee(ctx context.Context) bool {
	isDoitEmployee, ok := ctx.Value(common.DoitEmployee).(bool)
	if ok {
		return isDoitEmployee
	}

	return false
}
