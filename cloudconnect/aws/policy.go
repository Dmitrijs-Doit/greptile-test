package aws

import "encoding/json"

type PolicyPermissions struct {
	Version   string `json:"Version"`
	Statement []struct {
		Sid      string          `json:"Sid"`
		Effect   string          `json:"Effect"`
		Action   []string        `json:"Action"`
		Resource json.RawMessage `json:"Resource"`
	} `json:"Statement"`
}
