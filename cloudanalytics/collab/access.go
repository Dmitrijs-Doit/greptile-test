package collab

type Access struct {
	Collaborators []Collaborator `firestore:"collaborators"`
	Public        *PublicAccess  `firestore:"public"`
}

func (c Access) CanView(email string) bool {
	if c.Public != nil {
		return true
	}

	for _, collaborator := range c.Collaborators {
		if collaborator.Email == email {
			return true
		}
	}

	return false
}

func (c Access) CanEdit(email string) bool {
	if c.Public != nil {
		if *c.Public == PublicAccessEdit {
			return true
		}
	}

	for _, collaborator := range c.Collaborators {
		if collaborator.Email == email {
			return collaborator.Role == CollaboratorRoleEditor || collaborator.Role == CollaboratorRoleOwner
		}
	}

	return false
}

func (c Access) IsOwner(email string) bool {
	for _, collaborator := range c.Collaborators {
		if collaborator.Email == email {
			return collaborator.Role == CollaboratorRoleOwner
		}
	}

	return false
}

func (c Access) GetOwner() string {
	for _, collaborator := range c.Collaborators {
		if collaborator.Role == CollaboratorRoleOwner {
			return collaborator.Email
		}
	}

	return ""
}
