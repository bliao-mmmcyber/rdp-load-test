package guac

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/logging"
)

var (
	lock     sync.Mutex
	rdpRooms = make(map[string]*RdpSessionRoom)
)

type UserList struct {
	Users []User `json:"users"`
}
type User struct {
	UserId     string `json:"userId"`
	Permission string `json:"permission"`
	Status     int    `json:"status"`
}

type WriterCloser interface {
	io.Closer
	WriteMessage(messageType int, data []byte) error
}

type RdpClient struct {
	Websocket WriterCloser
	UserId    string
	Role      string // admin or cohost or viewer
	Mouse     bool
	Keyboard  bool
}

func (c *RdpClient) SendPermission() {
	var permissions []string
	if c.Keyboard {
		permissions = append(permissions, "keyboard")
	}
	if c.Mouse {
		permissions = append(permissions, "mouse")
	}
	permissions = append(permissions, c.Role)
	permissionStr := strings.Join(permissions, ",")
	ins := NewInstruction(USER_PERMISSON, []string{permissionStr}...)
	e := c.Websocket.WriteMessage(websocket.TextMessage, ins.Byte())
	if e != nil {
		logrus.Errorf("write user-permission command to client failed %v", e)
	}
}

type RdpSessionRoom struct {
	Creator         string
	AppId           string
	TenantId        string
	SessionId       string
	RdpConnectionId string
	AllowSharing    bool
	Users           map[string]*RdpClient
	Invitees        map[string]string
	lock            *sync.Mutex
}

func (r *RdpSessionRoom) GetRdpClient(userId string) *RdpClient {
	r.lock.Lock()
	defer r.lock.Unlock()

	if u, ok := r.Users[userId]; ok {
		return u
	}
	return nil
}

func (r *RdpSessionRoom) GetMembersInstruction() *Instruction {
	room, ok := rdpRooms[r.SessionId]
	if !ok {
		return NewInstruction(MEMBERS)
	}
	var users []User
	for _, u := range room.Users {
		var permissions []string
		if u.Mouse {
			permissions = append(permissions, "mouse")
		}
		if u.Keyboard {
			permissions = append(permissions, "keyboard")
		}
		if u.Role != ROLE_VIEWER {
			permissions = append(permissions, "admin")
		}
		users = append(users, User{
			UserId:     u.UserId,
			Permission: strings.Join(permissions, ","),
			Status:     1,
		})
	}
	for u, permission := range room.Invitees {
		if _, ok := room.Users[u]; !ok {
			users = append(users, User{
				UserId:     u,
				Permission: permission,
				Status:     0,
			})
		}
	}
	data, e := json.Marshal(users)
	if e != nil {
		logrus.Infof("marshall failed %v", e)
	}
	return NewInstruction(MEMBERS, string(data))
}

func AuthShare(userId, shareSessionId string) (bool, string) {
	// return true, ""

	var permissions string
	user, e := dbAccess.GetInviteeByUserIdAndSessionId(userId, shareSessionId)
	if e != nil {
		logrus.Errorf("query invitee by user %s and session %s failed", userId, shareSessionId)
		return false, permissions
	} else {
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
	r.lock.Lock()
	defer r.lock.Unlock()

	role := ROLE_VIEWER
	if strings.Contains(permissions, "admin") {
		role = ROLE_CO_HOST
	}
	r.Users[user] = &RdpClient{
		UserId:    user,
		Websocket: ws,
		Role:      role,
		Mouse:     strings.Contains(permissions, "mouse"),
		Keyboard:  strings.Contains(permissions, "keyboard"),
	}
	logrus.Infof("room %s, user size %d", r.SessionId, len(r.Users))
	return r.Users[user]
}

func (r *RdpSessionRoom) leave(user string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	delete(r.Users, user)
}

func (r *RdpSessionRoom) RemoveUser(user string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	delete(r.Invitees, user)
	if u, ok := r.Users[user]; ok {
		e := u.Websocket.Close()
		if e != nil {
			logrus.Errorf("close client %s ws failed %v", user, e)
		}
		delete(r.Users, user)
	}
}

func (r *RdpSessionRoom) AddInvitee(user, permissions string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.Invitees[user] = permissions
}

func GetRdpSessionRoom(sessionId string) (*RdpSessionRoom, bool) {
	result, ok := rdpRooms[sessionId]
	return result, ok
}

func NewRdpSessionRoom(sessionId string, user string, closer WriterCloser, connectionId string, allowSharing bool, appId, tenantId string) *RdpClient {
	lock.Lock()
	defer lock.Unlock()

	room := &RdpSessionRoom{
		SessionId:       sessionId,
		AppId:           appId,
		TenantId:        tenantId,
		Users:           make(map[string]*RdpClient),
		RdpConnectionId: connectionId,
		Invitees:        make(map[string]string),
		AllowSharing:    allowSharing,
		lock:            &sync.Mutex{},
	}
	room.Invitees[user] = "admin,keyboard,mouse"
	room.Users[user] = &RdpClient{
		Websocket: closer,
		UserId:    user,
		Role:      ROLE_ADMIN,
		Mouse:     true,
		Keyboard:  true,
	}
	rdpRooms[sessionId] = room
	logrus.Infof("add rdp room, session id %s", sessionId)
	return room.Users[user]
}

func AddInvitee(sessionId string, user string, permissions string) error {
	lock.Lock()
	defer lock.Unlock()
	if room, ok := GetRdpSessionRoom(sessionId); ok {
		room.AddInvitee(user, permissions)
		return nil
	}
	return fmt.Errorf("room with session id %s not found", sessionId)
}

func JoinRoom(sessionId string, user string, ws WriterCloser, permissions string) (*RdpClient, error) {
	lock.Lock()
	defer lock.Unlock()

	var result *RdpClient
	if room, ok := GetRdpSessionRoom(sessionId); ok {
		result = room.join(user, ws, permissions)
		ins := room.GetMembersInstruction()
		for _, u := range rdpRooms[sessionId].Users {
			if err := u.Websocket.WriteMessage(websocket.TextMessage, ins.Byte()); err != nil {
				logrus.Errorf("send message %s to user %s failed", ins.String(), u.UserId)
			}
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("cannot find rdp room by id %s", sessionId)
	}
}

func LeaveRoom(sessionId, user string) error {
	lock.Lock()
	defer lock.Unlock()

	if room, ok := GetRdpSessionRoom(sessionId); ok {
		room.leave(user)

		hasAdmin := false
		for _, u := range room.Users {
			if u.Role != ROLE_VIEWER {
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
			e := dbAccess.DeleteRdpSession(sessionId)
			e2 := kv.Delete(fmt.Sprintf("guac-%s", sessionId))
			logrus.Infof("remove session data %s, remaining size %d, e %v, e2 %v", sessionId, len(SessionDataStore.Data), e, e2)

			logging.Log(logging.Action{
				AppTag:       "guac.exit",
				RdpSessionId: sessionId,
				UserEmail:    room.Creator,
				AppID:        room.AppId,
				TenantID:     room.TenantId,
			})
		}
	} else {
		return fmt.Errorf("cannot find rdp room by id %s", sessionId)
	}
	return nil
}
