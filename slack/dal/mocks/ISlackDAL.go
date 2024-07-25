// Code generated by mockery v2.35.1. DO NOT EDIT.

package mocks

import (
	context "context"

	domain "github.com/doitintl/hello/scheduled-tasks/slack/domain"

	mock "github.com/stretchr/testify/mock"

	slack "github.com/slack-go/slack"
)

// ISlackDAL is an autogenerated mock type for the ISlackDAL type
type ISlackDAL struct {
	mock.Mock
}

// CreateChannel provides a mock function with given fields: channelName
func (_m *ISlackDAL) CreateChannel(channelName string) (*slack.Channel, error) {
	ret := _m.Called(channelName)

	var r0 *slack.Channel
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*slack.Channel, error)); ok {
		return rf(channelName)
	}
	if rf, ok := ret.Get(0).(func(string) *slack.Channel); ok {
		r0 = rf(channelName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*slack.Channel)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(channelName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateChannelWithFallback provides a mock function with given fields: ctx, channelName
func (_m *ISlackDAL) CreateChannelWithFallback(ctx context.Context, channelName string) (*slack.Channel, error) {
	ret := _m.Called(ctx, channelName)

	var r0 *slack.Channel
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*slack.Channel, error)); ok {
		return rf(ctx, channelName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *slack.Channel); ok {
		r0 = rf(ctx, channelName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*slack.Channel)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, channelName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAllChannelMemberEmails provides a mock function with given fields: ctx, workspaceID, channelID
func (_m *ISlackDAL) GetAllChannelMemberEmails(ctx context.Context, workspaceID string, channelID string) ([]string, error) {
	ret := _m.Called(ctx, workspaceID, channelID)

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) ([]string, error)); ok {
		return rf(ctx, workspaceID, channelID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) []string); ok {
		r0 = rf(ctx, workspaceID, channelID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, workspaceID, channelID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAllCustomerChannels provides a mock function with given fields: ctx, customerID
func (_m *ISlackDAL) GetAllCustomerChannels(ctx context.Context, customerID string) ([]slack.Channel, error) {
	ret := _m.Called(ctx, customerID)

	var r0 []slack.Channel
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]slack.Channel, error)); ok {
		return rf(ctx, customerID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []slack.Channel); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]slack.Channel)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetChannelInfo provides a mock function with given fields: ctx, workspaceID, channelID
func (_m *ISlackDAL) GetChannelInfo(ctx context.Context, workspaceID string, channelID string) (*slack.Channel, error) {
	ret := _m.Called(ctx, workspaceID, channelID)

	var r0 *slack.Channel
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*slack.Channel, error)); ok {
		return rf(ctx, workspaceID, channelID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *slack.Channel); ok {
		r0 = rf(ctx, workspaceID, channelID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*slack.Channel)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, workspaceID, channelID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetChannelMembers provides a mock function with given fields: channelID
func (_m *ISlackDAL) GetChannelMembers(channelID string) ([]string, error) {
	ret := _m.Called(channelID)

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(string) ([]string, error)); ok {
		return rf(channelID)
	}
	if rf, ok := ret.Get(0).(func(string) []string); ok {
		r0 = rf(channelID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(channelID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerPrivateChannelsForUser provides a mock function with given fields: ctx, customerID, userEmail
func (_m *ISlackDAL) GetCustomerPrivateChannelsForUser(ctx context.Context, customerID string, userEmail string) ([]slack.Channel, error) {
	ret := _m.Called(ctx, customerID, userEmail)

	var r0 []slack.Channel
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) ([]slack.Channel, error)); ok {
		return rf(ctx, customerID, userEmail)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) []slack.Channel); ok {
		r0 = rf(ctx, customerID, userEmail)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]slack.Channel)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, customerID, userEmail)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetInternalChannelInfo provides a mock function with given fields: channelID
func (_m *ISlackDAL) GetInternalChannelInfo(channelID string) (*slack.Channel, error) {
	ret := _m.Called(channelID)

	var r0 *slack.Channel
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*slack.Channel, error)); ok {
		return rf(channelID)
	}
	if rf, ok := ret.Get(0).(func(string) *slack.Channel); ok {
		r0 = rf(channelID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*slack.Channel)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(channelID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetInternalUserByEmail provides a mock function with given fields: ctx, email
func (_m *ISlackDAL) GetInternalUserByEmail(ctx context.Context, email string) (*slack.User, error) {
	ret := _m.Called(ctx, email)

	var r0 *slack.User
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*slack.User, error)); ok {
		return rf(ctx, email)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *slack.User); ok {
		r0 = rf(ctx, email)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*slack.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, email)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUser provides a mock function with given fields: ctx, workspaceID, ID
func (_m *ISlackDAL) GetUser(ctx context.Context, workspaceID string, ID string) (*slack.User, error) {
	ret := _m.Called(ctx, workspaceID, ID)

	var r0 *slack.User
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*slack.User, error)); ok {
		return rf(ctx, workspaceID, ID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *slack.User); ok {
		r0 = rf(ctx, workspaceID, ID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*slack.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, workspaceID, ID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUserByEmail provides a mock function with given fields: ctx, customerID, email
func (_m *ISlackDAL) GetUserByEmail(ctx context.Context, customerID string, email string) (*slack.User, error) {
	ret := _m.Called(ctx, customerID, email)

	var r0 *slack.User
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*slack.User, error)); ok {
		return rf(ctx, customerID, email)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *slack.User); ok {
		r0 = rf(ctx, customerID, email)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*slack.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, customerID, email)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUserEmail provides a mock function with given fields: ctx, customerID, userID
func (_m *ISlackDAL) GetUserEmail(ctx context.Context, customerID string, userID string) (string, error) {
	ret := _m.Called(ctx, customerID, userID)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (string, error)); ok {
		return rf(ctx, customerID, userID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) string); ok {
		r0 = rf(ctx, customerID, userID)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, customerID, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// InviteUsersToChannel provides a mock function with given fields: channelID, users
func (_m *ISlackDAL) InviteUsersToChannel(channelID string, users ...string) (*slack.Channel, error) {
	_va := make([]interface{}, len(users))
	for _i := range users {
		_va[_i] = users[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, channelID)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *slack.Channel
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...string) (*slack.Channel, error)); ok {
		return rf(channelID, users...)
	}
	if rf, ok := ret.Get(0).(func(string, ...string) *slack.Channel); ok {
		r0 = rf(channelID, users...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*slack.Channel)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...string) error); ok {
		r1 = rf(channelID, users...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PublishAppHome provides a mock function with given fields: ctx, workspaceID, userID, viewRequest
func (_m *ISlackDAL) PublishAppHome(ctx context.Context, workspaceID string, userID string, viewRequest slack.HomeTabViewRequest) error {
	ret := _m.Called(ctx, workspaceID, userID, viewRequest)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, slack.HomeTabViewRequest) error); ok {
		r0 = rf(ctx, workspaceID, userID, viewRequest)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SendEphemeral provides a mock function with given fields: ctx, channelID, userID, blocks
func (_m *ISlackDAL) SendEphemeral(ctx context.Context, channelID string, userID string, blocks *slack.MsgOption) (string, error) {
	ret := _m.Called(ctx, channelID, userID, blocks)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *slack.MsgOption) (string, error)); ok {
		return rf(ctx, channelID, userID, blocks)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *slack.MsgOption) string); ok {
		r0 = rf(ctx, channelID, userID, blocks)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, *slack.MsgOption) error); ok {
		r1 = rf(ctx, channelID, userID, blocks)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendInternalMessage provides a mock function with given fields: ctx, channelID, blocks
func (_m *ISlackDAL) SendInternalMessage(ctx context.Context, channelID string, blocks *slack.MsgOption) (string, error) {
	ret := _m.Called(ctx, channelID, blocks)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *slack.MsgOption) (string, error)); ok {
		return rf(ctx, channelID, blocks)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *slack.MsgOption) string); ok {
		r0 = rf(ctx, channelID, blocks)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *slack.MsgOption) error); ok {
		r1 = rf(ctx, channelID, blocks)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendMessage provides a mock function with given fields: ctx, customerID, channelID, blocks
func (_m *ISlackDAL) SendMessage(ctx context.Context, customerID string, channelID string, blocks *slack.MsgOption) (string, error) {
	ret := _m.Called(ctx, customerID, channelID, blocks)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *slack.MsgOption) (string, error)); ok {
		return rf(ctx, customerID, channelID, blocks)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *slack.MsgOption) string); ok {
		r0 = rf(ctx, customerID, channelID, blocks)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, *slack.MsgOption) error); ok {
		r1 = rf(ctx, customerID, channelID, blocks)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendResponse provides a mock function with given fields: ctx, workspaceID, channelID, responseURL, blocks
func (_m *ISlackDAL) SendResponse(ctx context.Context, workspaceID string, channelID string, responseURL string, blocks []slack.Block) error {
	ret := _m.Called(ctx, workspaceID, channelID, responseURL, blocks)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, []slack.Block) error); ok {
		r0 = rf(ctx, workspaceID, channelID, responseURL, blocks)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SendUnfurl provides a mock function with given fields: ctx, unfurlPayload
func (_m *ISlackDAL) SendUnfurl(ctx context.Context, unfurlPayload *domain.UnfurlPayload) error {
	ret := _m.Called(ctx, unfurlPayload)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *domain.UnfurlPayload) error); ok {
		r0 = rf(ctx, unfurlPayload)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SendUnfurlWithEphemeral provides a mock function with given fields: ctx, unfurlPayload
func (_m *ISlackDAL) SendUnfurlWithEphemeral(ctx context.Context, unfurlPayload *domain.UnfurlPayload) error {
	ret := _m.Called(ctx, unfurlPayload)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *domain.UnfurlPayload) error); ok {
		r0 = rf(ctx, unfurlPayload)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewISlackDAL creates a new instance of ISlackDAL. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewISlackDAL(t interface {
	mock.TestingT
	Cleanup(func())
}) *ISlackDAL {
	mock := &ISlackDAL{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}