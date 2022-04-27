package guac

import (
	"testing"

	"github.com/appaegis/golang-common/pkg/dynamodbcli"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wwt/guac/mocks"
)

func TestSharingAndRmoeveShareCommand(t *testing.T) {
	svc := new(mocks.MailService)
	mailService = svc
	svc.On("SendInvitation", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_ = NewRdpSessionRoom("123", "test@appaegis.com", nil, "connectionId", true)

	i := Instruction{
		Args: []string{SHARE_SESSION, "requestId", "kchung@appaegis.com", "mouse,keyboard,admin"},
	}
	c, e := GetCommandByOp(&i)
	if e != nil {
		t.Fatal("cannot get share-session command")
	}

	db := new(mocks.DbAccess)
	dbAccess = db // inject mock
	db.On("ShareRdpSession", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result := c.Exec(&i, &SessionCommonData{RdpSessionId: "123"}, &RdpClient{})
	assert.Equal(t, result.Args[2], "200", "incorrect status")
	assert.NotEmpty(t, result.Args[3], "url should not be empty")

	ins := NewInstruction(APPAEGIS_OP, REMOVE_SHARE, "requestId", "kchung@appaegis.com")
	c, e = GetCommandByOp(ins)
	if e != nil {
		t.Fatal("cannot get remove-share command")
	}
	db.On("RemoveInvitee", mock.Anything, mock.Anything).Return(nil)
	result = c.Exec(ins, &SessionCommonData{
		RdpSessionId: "123",
	}, &RdpClient{})
	logrus.Infof("result %s", result.String())
	assert.Equal(t, result.Args[2], "200", "incorrect result status")

	delete(rdpRooms, "123")
}

func TestSearchUserCommand(t *testing.T) {
	i := NewInstruction(APPAEGIS_OP, SEARCH_USER, "requestId", "kchung")
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
	assert.Equal(t, result.Args[0], SEARCH_USER_ACK)
	assert.Equal(t, result.Args[1], "requestId")
	assert.Equal(t, result.Args[2], "kchung@appaegis.com")
}

func TestSetPermissionsCommand(t *testing.T) {
	i := &Instruction{
		Args: []string{SET_PERMISSONS, "requestId", "user2:admin,mouse,keyboard"},
	}
	c, e := GetCommandByOp(i)
	if e != nil {
		t.Fatal("cannot get set permissions command")
	}
	db := new(mocks.DbAccess)
	dbAccess = db

	ws1 := new(mocks.WriterCloser)
	ws1.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	NewRdpSessionRoom("1", "user1", ws1, "", true)

	ws2 := new(mocks.WriterCloser)
	ws2.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	_, _ = JoinRoom("1", "user2", ws2, "")
	r, ok := GetRdpSessionRoom("1")
	if !ok {
		t.Fatal("cannot get rdp session room")
	}
	user2 := r.Users["user2"]

	result := c.Exec(i, &SessionCommonData{RdpSessionId: "1"}, &RdpClient{Admin: true})
	assert.True(t, user2.Admin)
	assert.True(t, user2.Keyboard)
	assert.True(t, user2.Mouse)
	assert.Nil(t, result)

	delete(rdpRooms, "1")
}
