// Code generated by mockery v2.15.0. DO NOT EDIT.

package mocks

import (
	costandusagereportservice "github.com/aws/aws-sdk-go/service/costandusagereportservice"

	iam "github.com/aws/aws-sdk-go/service/iam"

	mock "github.com/stretchr/testify/mock"

	s3 "github.com/aws/aws-sdk-go/service/s3"
)

// IAWSDal is an autogenerated mock type for the IAWSDal type
type IAWSDal struct {
	mock.Mock
}

// DescribeReportDefinitions provides a mock function with given fields: accountID
func (_m *IAWSDal) DescribeReportDefinitions(accountID string) (*costandusagereportservice.DescribeReportDefinitionsOutput, error) {
	ret := _m.Called(accountID)

	var r0 *costandusagereportservice.DescribeReportDefinitionsOutput
	if rf, ok := ret.Get(0).(func(string) *costandusagereportservice.DescribeReportDefinitionsOutput); ok {
		r0 = rf(accountID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*costandusagereportservice.DescribeReportDefinitionsOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(accountID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPolicy provides a mock function with given fields: accountID, policyArn
func (_m *IAWSDal) GetPolicy(accountID string, policyArn string) (*iam.GetPolicyOutput, error) {
	ret := _m.Called(accountID, policyArn)

	var r0 *iam.GetPolicyOutput
	if rf, ok := ret.Get(0).(func(string, string) *iam.GetPolicyOutput); ok {
		r0 = rf(accountID, policyArn)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*iam.GetPolicyOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accountID, policyArn)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPolicyVersion provides a mock function with given fields: accountID, policyArn, versionID
func (_m *IAWSDal) GetPolicyVersion(accountID string, policyArn string, versionID string) (*iam.GetPolicyVersionOutput, error) {
	ret := _m.Called(accountID, policyArn, versionID)

	var r0 *iam.GetPolicyVersionOutput
	if rf, ok := ret.Get(0).(func(string, string, string) *iam.GetPolicyVersionOutput); ok {
		r0 = rf(accountID, policyArn, versionID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*iam.GetPolicyVersionOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, string) error); ok {
		r1 = rf(accountID, policyArn, versionID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRole provides a mock function with given fields: accountID, roleArn
func (_m *IAWSDal) GetRole(accountID string, roleArn string) (*iam.GetRoleOutput, error) {
	ret := _m.Called(accountID, roleArn)

	var r0 *iam.GetRoleOutput
	if rf, ok := ret.Get(0).(func(string, string) *iam.GetRoleOutput); ok {
		r0 = rf(accountID, roleArn)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*iam.GetRoleOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accountID, roleArn)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListObjectsV2 provides a mock function with given fields: accountID, s3Bucket
func (_m *IAWSDal) ListObjectsV2(accountID string, s3Bucket string) (*s3.ListObjectsV2Output, error) {
	ret := _m.Called(accountID, s3Bucket)

	var r0 *s3.ListObjectsV2Output
	if rf, ok := ret.Get(0).(func(string, string) *s3.ListObjectsV2Output); ok {
		r0 = rf(accountID, s3Bucket)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*s3.ListObjectsV2Output)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accountID, s3Bucket)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewIAWSDal interface {
	mock.TestingT
	Cleanup(func())
}

// NewIAWSDal creates a new instance of IAWSDal. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIAWSDal(t mockConstructorTestingTNewIAWSDal) *IAWSDal {
	mock := &IAWSDal{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}