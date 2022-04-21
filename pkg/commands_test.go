package guac

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wwt/guac/mocks"
	"testing"
)

func TestSharingCommand(t *testing.T) {
	i := Instruction{
		Args: []string{SHARE_SESSION, "requestId", "kchung@appaegis.com", "mouse,keyboard,admin"},
	}
	c, e := GetCommandByOp(&i)
	if e != nil {
		t.Fatal("cannot get share-session command")
	}
	db := new(mocks.DbAccess)
	dbAccess = db //inject mock
	db.On("ShareRdpSession", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result := c.Exec(&i, &SessionCommonData{RdpSessionId: "123"}, &RdpClient{})
	assert.Equal(t, result.Args[2], "200", "incorrect status")
	assert.NotEmpty(t, result.Args[3], "url should not be empty")

}

func TestSetPermissionsCommand(t *testing.T) {
	i := &Instruction{
		Args: []string{SET_PERMISSONS, "user2:admin,mouse,keyboard"},
	}
	c, e := GetCommandByOp(i)
	if e != nil {
		t.Fatal("cannot get set permissions command")
	}
	db := new(mocks.DbAccess)
	dbAccess = db

	ws1 := new(mocks.WriterCloser)
	ws1.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	NewRdpSessionRoom("1", "user1", ws1, "")

	ws2 := new(mocks.WriterCloser)
	ws2.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	JoinRoom("1", "user2", ws2, "")
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

}
