package guac

import (
	"github.com/appaegis/golang-common/pkg/db_data/adaptor"
	"github.com/appaegis/golang-common/pkg/db_data/schema"
)

var dbAccess DbAccess = DynamodbAccess{}

type DbAccess interface {
	SaveActiveRdpSession(session *schema.ActiveRdpSession) error
	ShareRdpSession(invitee string, inviteePermissions string, sessionId string) error
	DeleteRdpSession(sessionId string) error
	GetInviteeByUserIdAndSessionId(userId, sessionId string) (*schema.ActiveRdpSessionInvitee, error)
	QueryUsersByTenantAndUserPrefix(tenantId, userPrefix string) ([]schema.UserEntry, error)
	RemoveInvitee(sessionId, user string) error
	GetTenantById(tenantId string) schema.TenantEntry
}

type DynamodbAccess struct{}

func (d DynamodbAccess) SaveActiveRdpSession(session *schema.ActiveRdpSession) error {
	return adaptor.GetDefaultDaoClient().SaveActiveRdpSession(session)
}

func (d DynamodbAccess) ShareRdpSession(invitee string, inviteePermissions string, sessionId string) error {
	return adaptor.GetDefaultDaoClient().ShareRdpSession(invitee, inviteePermissions, sessionId)
}

func (d DynamodbAccess) DeleteRdpSession(sessionId string) error {
	return adaptor.GetDefaultDaoClient().DeleteRdpSession(sessionId)
}

func (d DynamodbAccess) GetInviteeByUserIdAndSessionId(userId, sessionId string) (*schema.ActiveRdpSessionInvitee, error) {
	return adaptor.GetDefaultDaoClient().GetInviteeByUserIdAndSessionId(userId, sessionId)
}

func (d DynamodbAccess) QueryUsersByTenantAndUserPrefix(tenantId, userPrefix string) ([]schema.UserEntry, error) {
	return adaptor.GetDefaultDaoClient().QueryUsersByTenantAndUserPrefix(tenantId, userPrefix)
}

func (d DynamodbAccess) RemoveInvitee(sessionId, user string) error {
	return adaptor.GetDefaultDaoClient().RemoveInvitee(sessionId, user)
}

func (d DynamodbAccess) GetTenantById(tenantId string) schema.TenantEntry {
	return adaptor.GetDefaultDaoClient().GetTenantById(tenantId)
}
