package googlecloud

import (
	"testing"

	"google.golang.org/api/cloudbilling/v1"
)

func TestRemoveAdminMembers(t *testing.T) {
	memberToRemove := "user:member_to_remove@doit.com"
	policy := cloudbilling.Policy{Bindings: []*cloudbilling.Binding{}}
	policy.Bindings = append(policy.Bindings, &cloudbilling.Binding{
		Role:    RoleBillingAdmin,
		Members: []string{memberToRemove, "user:test1@doit.com", "user:test2@doit.com"},
	}, &cloudbilling.Binding{
		Role:    RoleBillingUser,
		Members: []string{"test3@doit.com"},
	})

	removeAdminMembers(&policy, []string{memberToRemove})

	if len(policy.Bindings) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(policy.Bindings))
	}

	if len(policy.Bindings[0].Members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(policy.Bindings[0].Members))
	}

	// Check that the member was removed
	for _, member := range policy.Bindings[0].Members {
		if member == memberToRemove {
			t.Errorf("Expected member %s to be removed", memberToRemove)
		}
	}
}

func TestGetDoitAdminMembers(t *testing.T) {
	memberToGet := "user:member_to_get@doit.com"
	policy := cloudbilling.Policy{Bindings: []*cloudbilling.Binding{}}
	policy.Bindings = append(policy.Bindings, &cloudbilling.Binding{
		Role:    RoleBillingAdmin,
		Members: []string{memberToGet, "user:test1@test.com", "user:test2@test.com"},
	}, &cloudbilling.Binding{
		Role:    RoleBillingUser,
		Members: []string{"test3@doit.com"},
	})

	members := getDoitAdminMembers(&policy)

	if len(members) != 1 {
		t.Errorf("Expected 1 member, got %d", len(members))
	}

	if members[0] != memberToGet {
		t.Errorf("Expected member %s, got %s", memberToGet, members[0])
	}
}

func TestValidateEmailDomains(t *testing.T) {
	validDomains := []string{"valid.com"}
	validEmails := []string{"test1@valid.com", "test2@VALID.COM", "test3@ValiD.Com"}
	invalidEmails := []string{"test", "test@", "test@invalid.com"}

	err := validateEmailDomains(validEmails, validDomains)
	if err != nil {
		t.Errorf("Expected valid emails to be valid")
	}

	for _, email := range invalidEmails {
		err := validateEmailDomains([]string{email}, validDomains)
		if err == nil {
			t.Errorf("Expected email %s to be invalid", email)
		}
	}
}
