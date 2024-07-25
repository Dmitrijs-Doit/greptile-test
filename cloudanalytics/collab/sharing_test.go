package collab

import (
	"testing"
)

const (
	requesterEmail = "requester@example.com"
	otherEmail     = "other-user@example.com"
)

func TestCollab_validateCollaborators(t *testing.T) {
	type args struct {
		oldCollabs []Collaborator
		newCollabs []Collaborator
		email      string
		oaAllowed  bool
	}

	tests := []struct {
		name    string
		c       *Collab
		args    args
		wantErr bool
	}{
		{
			name: "Happy path",
			args: args{
				oldCollabs: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
				},
				newCollabs: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
					{
						Email: otherEmail,
						Role:  CollaboratorRoleEditor,
					},
				},
				email:     requesterEmail,
				oaAllowed: false,
			},
			wantErr: false,
		},
		{
			name: "error invalid collaborator email",
			args: args{
				oldCollabs: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
				},
				newCollabs: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
					{
						Role: CollaboratorRoleViewer,
					},
				},
				email:     requesterEmail,
				oaAllowed: false,
			},
			wantErr: true,
		},
		{
			name: "error invalid collaborator role",
			args: args{
				oldCollabs: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
				},
				newCollabs: []Collaborator{
					{
						Email: requesterEmail,
					},
				},
				email:     requesterEmail,
				oaAllowed: false,
			},
			wantErr: true,
		},
		{
			name: "error too many owners",
			args: args{
				oldCollabs: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
				},
				newCollabs: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
					{
						Email: otherEmail,
						Role:  CollaboratorRoleOwner,
					},
				},
				email:     requesterEmail,
				oaAllowed: false,
			},
			wantErr: true,
		},
		{
			name: "error no owners",
			args: args{
				oldCollabs: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
				},
				newCollabs: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleEditor,
					},
					{
						Email: otherEmail,
						Role:  CollaboratorRoleEditor,
					},
				},
				email:     requesterEmail,
				oaAllowed: false,
			},
			wantErr: true,
		},
		{
			name: "error user is not the owner",
			args: args{
				oldCollabs: []Collaborator{
					{
						Email: otherEmail,
						Role:  CollaboratorRoleOwner,
					},
				},
				newCollabs: []Collaborator{
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
				},
				email: requesterEmail,
			},
			wantErr: true,
		},
		{
			name: "error user is not allowed to edit",
			args: args{
				oldCollabs: []Collaborator{
					{
						Email: otherEmail,
						Role:  CollaboratorRoleOwner,
					},
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleViewer,
					},
				},
				newCollabs: []Collaborator{
					{
						Email: otherEmail,
						Role:  CollaboratorRoleOwner,
					},
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleEditor,
					},
				},
				email:     requesterEmail,
				oaAllowed: false,
			},
			wantErr: true,
		},
		{
			name: "error user does not have owner assigner permission",
			args: args{
				oldCollabs: []Collaborator{
					{
						Email: otherEmail,
						Role:  CollaboratorRoleOwner,
					},
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleViewer,
					},
				},
				newCollabs: []Collaborator{
					{
						Email: otherEmail,
						Role:  CollaboratorRoleEditor,
					},
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
				},
				email:     requesterEmail,
				oaAllowed: false,
			},
			wantErr: true,
		},
		{
			name: "happy path if user has permissions to assign owner role",
			args: args{
				oldCollabs: []Collaborator{
					{
						Email: otherEmail,
						Role:  CollaboratorRoleOwner,
					},
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleViewer,
					},
				},
				newCollabs: []Collaborator{
					{
						Email: otherEmail,
						Role:  CollaboratorRoleEditor,
					},
					{
						Email: requesterEmail,
						Role:  CollaboratorRoleOwner,
					},
				},
				email:     requesterEmail,
				oaAllowed: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Collab{}
			if err := c.validateCollaborators(tt.args.oldCollabs, tt.args.newCollabs, tt.args.email, tt.args.oaAllowed); (err != nil) != tt.wantErr {
				t.Errorf("Collab.validateCollaborators() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCollab_validatePublicAccess(t *testing.T) {
	type args struct {
		public *PublicAccess
	}

	invalidPublicValue := PublicAccess("invalid role")
	publicView := PublicAccessView
	publicEdit := PublicAccessEdit

	tests := []struct {
		name    string
		c       *Collab
		args    args
		wantErr bool
	}{
		{
			name: "Happy path no public",
			args: args{
				public: nil,
			},
			wantErr: false,
		},
		{
			name: "Happy path view",
			args: args{
				public: &publicView,
			},
			wantErr: false,
		},
		{
			name: "Happy path edit",
			args: args{
				public: &publicEdit,
			},
			wantErr: false,
		},
		{
			name: "error invalid public access value",
			args: args{
				public: &invalidPublicValue,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Collab{}
			if err := c.validatePublicAccess(tt.args.public); (err != nil) != tt.wantErr {
				t.Errorf("Collab.validatePublicAccess() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
