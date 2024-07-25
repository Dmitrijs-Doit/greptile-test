package collab

type AccessSetter interface {
	SetCollaborators(collaborators []Collaborator)
	SetPublic(public *PublicAccess)
}

// SetAccess sets the collaborators and public access sharing options
// for the given cloud analytics resource.
func SetAccess(object AccessSetter, email string, public *PublicAccess) {
	object.SetPublic(public)
	object.SetCollaborators([]Collaborator{
		{
			Email: email,
			Role:  CollaboratorRoleOwner,
		},
	})
}
