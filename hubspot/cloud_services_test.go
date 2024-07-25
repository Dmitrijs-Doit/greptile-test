package hubspot

import (
	"testing"
)

func Test_awsServices(t *testing.T) {
	s := awsServices()
	if v, ok := s["Amazon Quantum Ledger Database"]; !ok {
		t.Errorf("Expected Amazon Quantum Ledger Database to be in the map")

		if v != struct{}{} {
			t.Errorf("unexpected value for Amazon Quantum Ledger Database")
		}
	}
}
