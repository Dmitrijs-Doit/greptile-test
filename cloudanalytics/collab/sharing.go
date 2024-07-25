package collab

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AnalyticsSharer interface {
	Share(ctx context.Context, id string, collaborators []Collaborator, public *PublicAccess) error
}

type Collab struct {
}

func NewCollab() *Collab {
	return &Collab{}
}

func (*Collab) validateCollaborators(oldCollabs, newCollabs []Collaborator, requesterEmail string, oaAllowed bool) error {
	var (
		newOwner        string
		numOwners       int
		isAllowedToEdit bool
	)

	for _, c := range newCollabs {
		if c.Email == "" {
			return ErrInvalidCollaboratorEmail
		}

		switch c.Role {
		case CollaboratorRoleOwner:
			newOwner = c.Email
			numOwners++
		case CollaboratorRoleEditor, CollaboratorRoleViewer:
		default:
			return ErrInvalidCollaboratorRole
		}
	}

	// Resource must have exactly one owner
	if numOwners != 1 {
		return ErrInvalidAmountOfOwners
	}

	for _, c := range oldCollabs {
		switch c.Role {
		case CollaboratorRoleOwner:
			// Transfer of ownership can be done by the current owner or by user with Ownership assignment permissions
			if c.Email != newOwner && c.Email != requesterEmail && !oaAllowed {
				return ErrNotTheOwner
			}

			fallthrough
		case CollaboratorRoleEditor:
			if c.Email == requesterEmail {
				isAllowedToEdit = true
			}
		}
	}

	// Only editors/owner/owner admins may update collaborators
	if !isAllowedToEdit && !oaAllowed {
		return ErrUserNotEditor
	}

	return nil
}

func (*Collab) validatePublicAccess(public *PublicAccess) error {
	if public != nil && *public != PublicAccessEdit && *public != PublicAccessView {
		return ErrInvalidPublicAccessRole
	}

	return nil
}

func (c *Collab) ShareAnalyticsResource(ctx context.Context, oldCollabs, newCollabs []Collaborator, public *PublicAccess, resourceID, requesterEmail string, sharer AnalyticsSharer, isCAOwner bool) error {
	if err := c.validateCollaborators(oldCollabs, newCollabs, requesterEmail, isCAOwner); err != nil {
		return err
	}

	if err := c.validatePublicAccess(public); err != nil {
		return err
	}

	if err := sharer.Share(ctx, resourceID, newCollabs, public); err != nil {
		return err
	}

	return nil
}

type UpdateUserEmailInput struct {
	CustomerID  string
	OldEmail    string
	NewEmail    string
	Collections []string
}

func (c *Collab) UpdateCollabEmail(ctx context.Context, logger logger.ILogger, fs *firestore.Client, batch iface.Batch, input UpdateUserEmailInput) error {
	if input.NewEmail == input.OldEmail || len(input.Collections) == 0 {
		return nil
	}

	for _, col := range input.Collections {
		logger.Infoln("collection: ", col)

		query := fs.Collection(col).Where("collaborators", common.ArrayContainsAny, []Collaborator{
			{Email: input.OldEmail, Role: CollaboratorRoleOwner},
			{Email: input.OldEmail, Role: CollaboratorRoleEditor},
			{Email: input.OldEmail, Role: CollaboratorRoleViewer},
		})

		// If no customer provided, update for all customers
		if input.CustomerID != "" {
			customerRef := fs.Collection("customers").Doc(input.CustomerID)
			query.Where("customer", "==", customerRef)
		}

		docSnaps, err := query.Select("customer", "collaborators").Documents(ctx).GetAll()
		if err != nil {
			return err
		}

		if len(docSnaps) == 0 {
			continue
		}

		logger.Infoln("num objects: ", len(docSnaps))

		for _, docSnap := range docSnaps {
			var r Collaborators
			if err := docSnap.DataTo(&r); err != nil {
				return err
			}

			for _, oldCollab := range r.Collaborators {
				if oldCollab.Email == input.OldEmail {
					newCollab := Collaborator{Email: input.NewEmail, Role: oldCollab.Role}

					if err := batch.Update(ctx, docSnap.Ref, []firestore.Update{
						{FieldPath: []string{"collaborators"}, Value: firestore.ArrayUnion(newCollab)},
					}); err != nil {
						return err
					}

					if err := batch.Update(ctx, docSnap.Ref, []firestore.Update{
						{FieldPath: []string{"collaborators"}, Value: firestore.ArrayRemove(oldCollab)},
					}); err != nil {
						return err
					}

					break
				}
			}
		}

		if err := batch.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
}
