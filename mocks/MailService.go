// Code generated by mockery v2.10.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// MailService is an autogenerated mock type for the MailService type
type MailService struct {
	mock.Mock
}

// SendInvitation provides a mock function with given fields: to, inviter, link, appName
func (_m *MailService) SendInvitation(to string, inviter string, link string, appName string) error {
	ret := _m.Called(to, inviter, link, appName)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, string) error); ok {
		r0 = rf(to, inviter, link, appName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
