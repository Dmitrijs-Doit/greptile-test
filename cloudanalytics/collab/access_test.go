package collab

import "testing"

func TestAccess_CanView(t *testing.T) {
	publicAccessView := PublicAccessView
	tests := []struct {
		name  string
		a     Access
		email string
		want  bool
	}{
		{
			name: "can view with public access",
			a: Access{
				Public: &publicAccessView,
			},
			email: requesterEmail,
			want:  true,
		},
		{
			name: "can view with public access and collaborator",
			a: Access{
				Public: &publicAccessView,
				Collaborators: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleViewer,
					},
				},
			},
			email: requesterEmail,
			want:  true,
		},
		{
			name: "can view with collaborator",
			a: Access{
				Collaborators: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleViewer,
					},
				},
			},
			email: requesterEmail,
			want:  true,
		},
		{
			name: "can't view without public access and collaborator role",
			a: Access{
				Collaborators: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleViewer,
					},
				},
			},
			email: otherEmail,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.CanView(tt.email); got != tt.want {
				t.Errorf("CanView() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestAccess_CanEdit(t *testing.T) {
	publicAccessEdit := PublicAccessEdit
	publicAccessView := PublicAccessView
	tests := []struct {
		name  string
		a     Access
		email string
		want  bool
	}{
		{
			name: "can edit with public access",
			a: Access{
				Public: &publicAccessEdit,
			},
			email: requesterEmail,
			want:  true,
		},
		{
			name: "can edit with collaborator role editor public access view",
			a: Access{
				Collaborators: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleEditor,
					},
				},
				Public: &publicAccessView,
			},
			email: requesterEmail,
			want:  true,
		},
		{
			name: "can edit with collaborator role owner",
			a: Access{
				Collaborators: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
				},
			},
			email: requesterEmail,
			want:  true,
		},
		{
			name: "can't edit with collaborator role viewer",
			a: Access{
				Collaborators: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleViewer,
					},
				},
			},
			email: requesterEmail,
			want:  false,
		},
		{
			name: "can't edit without public access or collaborator role",
			a: Access{
				Collaborators: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleViewer,
					},
				},
			},
			email: otherEmail,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.CanEdit(tt.email); got != tt.want {
				t.Errorf("CanEdit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccess_IsOwner(t *testing.T) {
	tests := []struct {
		name  string
		a     Access
		email string
		want  bool
	}{
		{
			name: "is owner with collaborator role owner",
			a: Access{
				Collaborators: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
				},
			},
			email: requesterEmail,
			want:  true,
		},
		{
			name: "is not owner with collaborator role editor",
			a: Access{
				Collaborators: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleEditor,
					},
				},
			},
			email: requesterEmail,
			want:  false,
		},
		{
			name: "is not owner with collaborator role viewer",
			a: Access{
				Collaborators: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleViewer,
					},
				},
			},
			email: requesterEmail,
			want:  false,
		},
		{
			name: "is not owner without collaborator role",
			a: Access{
				Collaborators: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleViewer,
					},
				},
			},
			email: otherEmail,
			want:  false,
		},
		{
			name: "is not owner with empty collaborators list",
			a: Access{
				Collaborators: []Collaborator{},
			},
			email: requesterEmail,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.IsOwner(tt.email); got != tt.want {
				t.Errorf("IsOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}
