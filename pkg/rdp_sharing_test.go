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
	NewRdpSessionRoom("sessionId", "user1", nil, "connectionId", true)
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

	NewRdpSessionRoom("singleAdmin", "user1", nil, "", true)
	db.On("DeleteRdpSession", mock.Anything).Return(nil)
	_ = LeaveRoom("singleAdmin", "user1")
	assert.Equal(t, 0, len(rdpRooms))
}

func TestTwoAdmin(t *testing.T) {
	ws1 := new(mocks.WriterCloser)
	ws1.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	NewRdpSessionRoom("1", "user1", ws1, "", true)

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
	NewRdpSessionRoom("1", "user1", ws1, "", true)

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
	NewRdpSessionRoom("1", "user1", ws1, "", true)

	ws2 := new(mocks.WriterCloser)
	ws2.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	_, _ = JoinRoom("1", "user2", ws2, "mouse")

	_ = LeaveRoom("1", "user2")
	assert.Equal(t, 1, len(rdpRooms))
	assert.Equal(t, 1, len(rdpRooms["1"].Users))
}
