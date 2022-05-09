package guac

import "github.com/appaegis/golang-common/pkg/dynamodbcli"

var dbAccess DbAccess = DynamodbAccess{}

type DbAccess interface {
	SaveActiveRdpSession(session *dynamodbcli.ActiveRdpSession) error
	ShareRdpSession(invitee string, inviteePermissions string, sessionId string) error
	DeleteRdpSession(sessionId string) error
	GetInviteeByUserIdAndSessionId(userId, sessionId string) (*dynamodbcli.ActiveRdpSessionInvitee, error)
	QueryUsersByTenantAndUserPrefix(tenantId, userPrefix string) ([]dynamodbcli.UserEntry, error)
	RemoveInvitee(sessionId, user string) error
}

type DynamodbAccess struct{}

func (d DynamodbAccess) SaveActiveRdpSession(session *dynamodbcli.ActiveRdpSession) error {
	return dynamodbcli.SaveActiveRdpSession(session)
}

func (d DynamodbAccess) ShareRdpSession(invitee string, inviteePermissions string, sessionId string) error {
	return dynamodbcli.ShareRdpSession(invitee, inviteePermissions, sessionId)
}

func (d DynamodbAccess) DeleteRdpSession(sessionId string) error {
	return dynamodbcli.DeleteRdpSession(sessionId)
}

func (d DynamodbAccess) GetInviteeByUserIdAndSessionId(userId, sessionId string) (*dynamodbcli.ActiveRdpSessionInvitee, error) {
	return dynamodbcli.GetInviteeByUserIdAndSessionId(userId, sessionId)
}

func (d DynamodbAccess) QueryUsersByTenantAndUserPrefix(tenantId, userPrefix string) ([]dynamodbcli.UserEntry, error) {
	return dynamodbcli.Singleon().QueryUsersByTenantAndUserPrefix(tenantId, userPrefix)
}

func (d DynamodbAccess) RemoveInvitee(sessionId, user string) error {
	return dynamodbcli.RemoveInvitee(sessionId, user)
}
