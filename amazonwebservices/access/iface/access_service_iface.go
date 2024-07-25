//go:generate mockery --name Access --output ../mocks --outpkg mocks --case=underscore
package iface

import "github.com/aws/aws-sdk-go/aws/session"

type Access interface {
	GetAWSSession(accountID, functionName string) (*session.Session, error)
}
