package service

import (
	"testing"
)

func TestValidateRecipientsAgainstDomains(t *testing.T) {
	validDomains := []string{"valid.com"}
	validEmails := []string{"test1@valid.com", "test2@valid.com", "test3@valid.slack.com", "test@doit.com", "name@doit-intl.com", "kkk@doitintl.slack.com", "kkk@bbbb.slack.com", "xxxx@bbbb.teams.ms"}
	invalidEmails := []string{"test", "test@", "test@va", "test@invalid.com", "test@valid.com1", "test@doit.com", "name@doit-intl.com"}

	for _, email := range validEmails {
		err := validateRecipientsAgainstDomains([]string{email}, validDomains, true)
		if err != "" {
			t.Errorf("Expected valid emails to be valid, got error %s for email %s", err, email)
		}
	}

	for _, email := range invalidEmails {
		err := validateRecipientsAgainstDomains([]string{email}, validDomains, false)
		if err == "" {
			t.Errorf("Expected email %s to be invalid, got no error", email)
		}
	}
}
