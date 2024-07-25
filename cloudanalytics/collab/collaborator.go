package collab

import (
	"errors"
)

// When null, this will only be visible to the collaborators.
// Do NOT set the budgets public field field to owner. Leave it unset if it's not meant to be public.
// When set to "viewer", it's visible to all users in the user domain.
// When set to "editor", every user in the user domain can edit.
// swagger:enum CollaboratorRole
type CollaboratorRole string

type PublicAccess string

const (
	CollaboratorRoleOwner  CollaboratorRole = "owner"
	CollaboratorRoleEditor CollaboratorRole = "editor"
	CollaboratorRoleViewer CollaboratorRole = "viewer"
	PublicAccessView       PublicAccess     = "viewer"
	PublicAccessEdit       PublicAccess     = "editor"
)

type Collaborators struct {
	Collaborators []Collaborator `firestore:"collaborators"`
}

type Collaborator struct {
	Email string           `json:"email" firestore:"email"`
	Role  CollaboratorRole `json:"role" firestore:"role"`
}

var (
	ErrInvalidCollaboratorEmail = errors.New("invalid collaborator email")
	ErrInvalidCollaboratorRole  = errors.New("invalid collaborator role")
	ErrInvalidPublicAccessRole  = errors.New("invalid public access role")
	ErrInvalidAmountOfOwners    = errors.New("resource must have exactly one owner")
	ErrNotTheOwner              = errors.New("only the owner himself may transfer ownership")
	ErrUserNotEditor            = errors.New("only editors may update sharing settings")
	ErrNoCollaborators          = errors.New("collaborators are missing")
)
