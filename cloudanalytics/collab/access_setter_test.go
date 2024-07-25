package collab

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type testObject struct {
	Access
}

func (t *testObject) SetCollaborators(collaborators []Collaborator) {
	t.Collaborators = collaborators
}

func (t *testObject) SetPublic(public *PublicAccess) {
	t.Public = public
}

func TestSetInitialSharingOptions(t *testing.T) {
	type args struct {
		object *testObject
		email  string
		public *PublicAccess
	}

	const testEmail = "test@doit.com"

	viewerRole := PublicAccessView
	editorRole := PublicAccessEdit

	tests := []struct {
		name string
		args args
		want *testObject
	}{
		{
			name: "Test nil public access",
			args: args{
				object: &testObject{},
				email:  testEmail,
				public: nil,
			},
			want: &testObject{
				Access: Access{
					Collaborators: []Collaborator{
						{
							Email: testEmail,
							Role:  CollaboratorRoleOwner,
						},
					},
					Public: nil,
				},
			},
		},
		{
			name: "Test viewer public access",
			args: args{
				object: &testObject{},
				email:  testEmail,
				public: &viewerRole,
			},
			want: &testObject{
				Access: Access{
					Collaborators: []Collaborator{
						{
							Email: testEmail,
							Role:  CollaboratorRoleOwner,
						},
					},
					Public: &viewerRole,
				},
			},
		},
		{
			name: "Test editor public access",
			args: args{
				object: &testObject{},
				email:  testEmail,
				public: &editorRole,
			},
			want: &testObject{
				Access: Access{
					Collaborators: []Collaborator{
						{
							Email: testEmail,
							Role:  CollaboratorRoleOwner,
						},
					},
					Public: &editorRole,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetAccess(tt.args.object, tt.args.email, tt.args.public)

			if !cmp.Equal(tt.args.object, tt.want) {
				t.Errorf("SetAccess() got = %+v, want %+v", tt.args.object, tt.want)
			}
		})
	}
}
