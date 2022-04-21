package guac

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
	"sync"
)

var lock sync.Mutex
var rdpRooms = make(map[string]*RdpSessionRoom)

type WriterCloser interface {
	io.Closer
	WriteMessage(messageType int, data []byte) error
}

type RdpClient struct {
	Websocket WriterCloser
	UserId    string
	Admin     bool
	Mouse     bool
	Keyboard  bool
}

type RdpSessionRoom struct {
	SessionId       string
	RdpConnectionId string
	Users           map[string]*RdpClient
}

func (r *RdpSessionRoom) GetMembersInstruction() *Instruction {
	ins := &Instruction{
		Opcode: SESSION_SHARE_OP,
	}
	args := []string{MEMBERS}
	for _, u := range rdpRooms[r.SessionId].Users {
		var permissions []string
		if u.Mouse {
			permissions = append(permissions, "mouse")
		}
		if u.Keyboard {
			permissions = append(permissions, "keyboard")
		}
		if u.Admin {
			permissions = append(permissions, "admin")
		}
		args = append(args, fmt.Sprintf("%s:%s:1", u.UserId, strings.Join(permissions, ",")))
	}
	ins.Args = args
	return ins
}

func AuthShare(userId, shareSessionId string) (bool, string) {
	//return true, ""

	var permissions string
	user, e := dbAccess.GetInviteeByUserIdAndSessionId(userId, shareSessionId)
	if e != nil {
		logrus.Errorf("query invitee by user %s and session %s failed", userId, shareSessionId)
		return false, permissions
	}
	if e == nil {
		permissions = user.Permissions
	}
	room, ok := GetRdpSessionRoom(shareSessionId)
	if !ok {
		logrus.Errorf("room %s not found", shareSessionId)
		return false, ""
	}
	if _, ok := room.Users[userId]; ok {
		logrus.Errorf("user already join this session %s, u %s", shareSessionId, userId)
		return false, ""
	}
	return true, permissions
}

func (r *RdpSessionRoom) join(user string, ws WriterCloser, permissions string) *RdpClient {
	r.Users[user] = &RdpClient{
		UserId:    user,
		Websocket: ws,
		Admin:     strings.Contains(permissions, "admin"),
		Mouse:     strings.Contains(permissions, "mouse"),
		Keyboard:  strings.Contains(permissions, "keyboard"),
	}
	logrus.Infof("room %s, user size %d", r.SessionId, len(r.Users))
	return r.Users[user]
}

func (r *RdpSessionRoom) leave(user string) {
	delete(r.Users, user)
}

func GetRdpSessionRoom(sessionId string) (*RdpSessionRoom, bool) {
	result, ok := rdpRooms[sessionId]
	return result, ok
}

func NewRdpSessionRoom(sessionId string, user string, closer WriterCloser, connectionId string) *RdpClient {
	lock.Lock()
	defer lock.Unlock()

	room := &RdpSessionRoom{
		SessionId:       sessionId,
		Users:           make(map[string]*RdpClient),
		RdpConnectionId: connectionId,
	}
	room.Users[user] = &RdpClient{
		Websocket: closer,
		UserId:    user,
		Admin:     true,
		Mouse:     true,
		Keyboard:  true,
	}
	rdpRooms[sessionId] = room
	return room.Users[user]
}

func JoinRoom(sessionId string, user string, ws WriterCloser, permissions string) (*RdpClient, error) {
	lock.Lock()
	defer lock.Unlock()

	var result *RdpClient
	if room, ok := rdpRooms[sessionId]; ok {
		result = room.join(user, ws, permissions)
	} else {
		return nil, fmt.Errorf("cannot find rdp room by id %s", sessionId)
	}

	r := rdpRooms[sessionId]
	ins := r.GetMembersInstruction()
	for _, u := range rdpRooms[sessionId].Users {
		if err := u.Websocket.WriteMessage(websocket.TextMessage, ins.Byte()); err != nil {
			logrus.Errorf("send message %s to user %s failed", ins.String(), u.UserId)
		}
	}

	return result, nil
}

func LeaveRoom(sessionId, user string) error {
	lock.Lock()
	defer lock.Unlock()

	if room, ok := rdpRooms[sessionId]; ok {
		room.leave(user)

		hasAdmin := false
		for _, u := range room.Users {
			if u.Admin {
				hasAdmin = true
			}
		}
		if !hasAdmin {
			for _, u := range room.Users {
				logrus.Infof("disconnect user %s", u.UserId)
				u.Websocket.Close()
			}

			delete(rdpRooms, sessionId)
			SessionDataStore.Delete(sessionId)
			dbAccess.DeleteRdpSession(sessionId)
			logrus.Infof("remove session data %s, remaining size %d", sessionId, len(SessionDataStore.Data))
		}
	} else {
		return fmt.Errorf("cannot find rdp room by id %s", sessionId)
	}
	return nil

}
