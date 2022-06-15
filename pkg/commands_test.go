package guac

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/appaegis/golang-common/pkg/config"
	"github.com/appaegis/golang-common/pkg/dynamodbcli"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wwt/guac/lib/logging"
	"github.com/wwt/guac/mocks"
)

var loggingInfo = logging.LoggingInfo{
	TenantId: "tenantId",
}

func TestGetSharingUrl(t *testing.T) {
	url := "qa.appaegistest.com"
	suffix := strings.Join(strings.Split(url, ".")[1:], ".")
	assert.Equal(t, suffix, "appaegistest.com")

	config.AddConfig(config.PORTAL_HOSTNAME, "dev.appaegistest.com")
	db := new(mocks.DbAccess)
	dbAccess = db // inject mock
	db.On("GetTenantById", "tenantId").Return(dynamodbcli.TenantEntry{
		IdpDomain: "kchung",
	})
	db.On("GetTenantById", "tenantId2").Return(dynamodbcli.TenantEntry{})
	result := GetSharingUrl("sessionId", "tenantId")
	assert.True(t, strings.HasPrefix(result, "https://kchung.appaegistest.com"))

	result = GetSharingUrl("sessionId", "tenantId2")
	logrus.Infof("result %s", result)
	assert.True(t, strings.HasPrefix(result, "https://dev.appaegistest.com"))
}

func TestStopShareCommand(t *testing.T) {
	sessionId := "TestStopShare"
	ws1 := new(mocks.WriterCloser)
	ws1.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	NewRdpSessionRoom(sessionId, "user1", ws1, "", true, "appId", "appName", loggingInfo)
	_ = AddInvitee(sessionId, "user2", "")

	ws2 := new(mocks.WriterCloser)
	ws2.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	ws2.On("Close").Return(nil)
	_, _ = JoinRoom(sessionId, "user2", ws2, "")

	ins := NewInstruction(APPAEGIS_RESP_OP, "requestId", STOP_SHARE)
	c, e := GetCommandByOp(ins)
	if e != nil {
		t.Errorf("cannot found command by %s", STOP_SHARE)
	}

	db := new(mocks.DbAccess)
	dbAccess = db
	db.On("RemoveInvitee", mock.Anything, mock.Anything).Return(nil)

	result := c.Exec(ins, &SessionCommonData{RdpSessionId: sessionId}, &RdpClient{UserId: "user1", Role: ROLE_ADMIN})
	room, _ := GetRdpSessionRoom(sessionId)
	assert.True(t, strings.Contains(result.String(), "200")) // check status
	assert.True(t, len(room.Users) == 1)
	assert.True(t, len(room.Invitees) == 1)

	delete(rdpRooms, sessionId)
}

func TestSharingAndRmoeveShareCommand(t *testing.T) {
	svc := new(mocks.MailService)
	mailService = svc
	svc.On("SendInvitation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ws1 := new(mocks.WriterCloser)
	ws1.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	_ = NewRdpSessionRoom("123", "test@appaegis.com", ws1, "connectionId", true, "appId", "appName", loggingInfo)

	i := Instruction{
		Args: []string{"requestId", SHARE_SESSION, "kchung@appaegis.com:mouse,keyboard,admin"},
	}
	c, e := GetCommandByOp(&i)
	if e != nil {
		t.Fatal("cannot get share-session command")
	}

	db := new(mocks.DbAccess)
	dbAccess = db // inject mock
	db.On("ShareRdpSession", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	db.On("GetTenantById", mock.Anything).Return(dynamodbcli.TenantEntry{
		IdpDomain: "qa-john",
	})

	result := c.Exec(&i, &SessionCommonData{RdpSessionId: "123"}, &RdpClient{})
	m := make(map[string]string)
	_ = json.Unmarshal([]byte(result.Args[1]), &m)

	assert.Equal(t, m["status"], "200", "incorrect status")
	assert.NotEmpty(t, m["url"], "url should not be empty")

	ins := NewInstruction(APPAEGIS_OP, "requestId", REMOVE_SHARE, "kchung@appaegis.com")
	c, e = GetCommandByOp(ins)
	if e != nil {
		t.Fatal("cannot get remove-share command")
	}
	db.On("RemoveInvitee", mock.Anything, mock.Anything).Return(nil)
	result = c.Exec(ins, &SessionCommonData{
		RdpSessionId: "123",
	}, &RdpClient{})
	logrus.Infof("result %s", result.String())
	_ = json.Unmarshal([]byte(result.Args[1]), &m)
	assert.Equal(t, m["status"], "200", "incorrect result status")

	delete(rdpRooms, "123")
}

func TestSearchUserCommand(t *testing.T) {
	i := NewInstruction(APPAEGIS_OP, "requestId", SEARCH_USER, "kchung")
	db := new(mocks.DbAccess)
	dbAccess = db
	user := dynamodbcli.UserEntry{
		ID: "kchung@appaegis.com",
	}
	db.On("QueryUsersByTenantAndUserPrefix", "tenantId", "kchung").Return([]dynamodbcli.UserEntry{user}, nil)
	c, e := GetCommandByOp(i)
	if e != nil {
		t.Fatal("cannot get share-session command")
	}
	result := c.Exec(i, &SessionCommonData{RdpSessionId: "123", TenantID: "tenantId"}, &RdpClient{})
	assert.Equal(t, result.Args[0], "requestId")
	var searchResult SearchUserResp
	_ = json.Unmarshal([]byte(result.Args[1]), &searchResult)
	assert.Equal(t, searchResult.Users[0], "kchung@appaegis.com")
}

func TestSetPermissionsCommand(t *testing.T) {
	i := &Instruction{
		Args: []string{"requestId", SET_PERMISSONS, "user2:admin,mouse,keyboard"},
	}
	c, e := GetCommandByOp(i)
	if e != nil {
		t.Fatal("cannot get set permissions command")
	}
	db := new(mocks.DbAccess)
	dbAccess = db
	db.On("ShareRdpSession", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ws1 := new(mocks.WriterCloser)
	ws1.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	NewRdpSessionRoom("1", "user1", ws1, "", true, "appId", "appName", loggingInfo)

	ws2 := new(mocks.WriterCloser)
	ws2.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	_, _ = JoinRoom("1", "user2", ws2, "")
	r, ok := GetRdpSessionRoom("1")
	if !ok {
		t.Fatal("cannot get rdp session room")
	}
	user2 := r.Users["user2"]

	_ = c.Exec(i, &SessionCommonData{RdpSessionId: "1"}, &RdpClient{Role: ROLE_CO_HOST})
	assert.Equal(t, user2.Role, ROLE_CO_HOST)
	assert.True(t, user2.Keyboard)
	assert.True(t, user2.Mouse)

	delete(rdpRooms, "1")
}
