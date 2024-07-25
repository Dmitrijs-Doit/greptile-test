package knownissues

import "testing"

func Test_getAwsKnownIssueLevel(t *testing.T) {
	knownIssueDescription := "Current severity level: Operating normally\n"
	expected := "Operating normally"
	result := getAwsKnownIssueLevel(knownIssueDescription)

	if result != expected {
		t.Errorf("Expected severity level %q, but got %q", expected, result)
	}
}
