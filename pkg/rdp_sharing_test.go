package guac

import (
	"fmt"
	"testing"

	"github.com/appaegis/golang-common/pkg/dynamodbcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wwt/guac/mocks"
)

func TestAuthShare(t *testing.T) {
	// user not shared
	db := new(mocks.DbAccess)
	dbAccess = db
	db.On("GetInviteeByUserIdAndSessionId", "user2", "sessionId").Return(nil, fmt.Errorf(""))
	result, _ := AuthShare("user2", "sessionId")
	assert.False(t, result)

	// normal case
	NewRdpSessionRoom("sessionId", "user1", nil, "connectionId", true, "appId", "", loggingInfo)
	db.On("GetInviteeByUserIdAndSessionId", "user3", "sessionId").Return(&dynamodbcli.ActiveRdpSessionInvitee{
		Permissions: "mouse",
	}, nil)
	result, permissions := AuthShare("user3", "sessionId")
	assert.True(t, result)
	assert.Equal(t, permissions, "mouse")

	// clear data
	delete(rdpRooms, "sessionId")
}

func TestSingleAdmin(t *testing.T) {
	db := new(mocks.DbAccess)
	dbAccess = db

	NewRdpSessionRoom("singleAdmin", "user1", nil, "", true, "appId", "appName", loggingInfo)
	db.On("DeleteRdpSession", mock.Anything).Return(nil)
	_ = LeaveRoom("singleAdmin", "user1")
	assert.Equal(t, 0, len(rdpRooms))
}

func TestTwoAdmin(t *testing.T) {
	ws1 := new(mocks.WriterCloser)
	ws1.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	NewRdpSessionRoom("1", "user1", ws1, "", true, "appId", "appName", loggingInfo)

	ws := new(mocks.WriterCloser)
	ws.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	_, _ = JoinRoom("1", "user2", ws, "admin")

	_ = LeaveRoom("1", "user1")
	assert.Equal(t, 1, len(rdpRooms))
	assert.Equal(t, 1, len(rdpRooms["1"].Users))
}

func TestSingleAdminLeave(t *testing.T) {
	ws1 := new(mocks.WriterCloser)
	ws1.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	NewRdpSessionRoom("1", "user1", ws1, "", true, "appId", "appName", loggingInfo)

	ws := new(mocks.WriterCloser)
	ws.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	_, _ = JoinRoom("1", "user2", ws, "mouse")

	ws.On("Close").Return(nil)
	_ = LeaveRoom("1", "user1")

	assert.Equal(t, 0, len(rdpRooms))
}

func TestNormalUserLeave(t *testing.T) {
	ws1 := new(mocks.WriterCloser)
	ws1.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	NewRdpSessionRoom("1", "user1", ws1, "", true, "appId", "appName", loggingInfo)

	ws2 := new(mocks.WriterCloser)
	ws2.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	_, _ = JoinRoom("1", "user2", ws2, "mouse")

	_ = LeaveRoom("1", "user2")
	assert.Equal(t, 1, len(rdpRooms))
	assert.Equal(t, 1, len(rdpRooms["1"].Users))
}

func TestAddSharing(t *testing.T) {
	sessionId := "testaddsharing"
	NewRdpSessionRoom(sessionId, "user1", nil, "", true, "", "", loggingInfo)
	e := AddInvitee(sessionId, "user2", "")
	if e != nil {
		t.Errorf("add sharing user failed %v", e)
	}
	r, ok := GetRdpSessionRoom(sessionId)
	if !ok || len(r.Invitees) != 2 {
		t.Error(e)
	}
	delete(rdpRooms, sessionId)
}

func TestGetRoomByAppIdAndCreator(t *testing.T) {
	sessionId := "TestGetRoomByAppIdAndCreator"

	ws1 := new(mocks.WriterCloser)
	ws1.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	NewRdpSessionRoom(sessionId, "user1", ws1, "", true, "appId", "", loggingInfo)

	ws2 := new(mocks.WriterCloser)
	ws2.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	_, _ = JoinRoom(sessionId, "user2", ws2, "admin")
	_ = LeaveRoom(sessionId, "user1")

	r, ok := GetRoomByAppIdAndCreator("appId", "user1")
	assert.True(t, ok)
	assert.NotNil(t, r)
	assert.Equal(t, r.Creator, "user1")
}
