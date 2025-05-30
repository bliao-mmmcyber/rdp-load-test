package guac

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/appaegis/golang-common/pkg/db_data/schema"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/logging"
	"github.com/wwt/guac/pkg/session"
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
	Role       string `json:"role"`
	Permission string `json:"permission"`
	Status     int    `json:"status"`
}

type WriterCloser interface {
	io.Closer
	WriteMessage(messageType int, data []byte) error
}

type RdpClient struct {
	Websocket WriterCloser
	UserAgent schema.UserAgent
	UserId    string
	Role      string // admin or cohost or viewer
	Mouse     bool
	Keyboard  bool
	lock      sync.Mutex
}

func (c *RdpClient) WriteMessage(ins *Instruction) {
	// logrus.Debug("appaegis cmd send: ", ins.String())
	e := c.Websocket.WriteMessage(websocket.TextMessage, ins.Byte())
	if e != nil {
		logrus.Errorf("write message to %s failed %v", c.UserId, e)
	}
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
	c.WriteMessage(ins)
}

type RdpSessionRoom struct {
	Creator         string
	AppId           string
	SessionId       string
	RdpConnectionId string
	AllowSharing    bool
	Users           map[string]*RdpClient
	Invitees        map[string]string
	lock            *sync.Mutex
	loggingInfo     *logging.LoggingInfo
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
			Role:       u.Role,
			Permission: strings.Join(permissions, ","),
			Status:     1,
		})
	}
	for u, permission := range room.Invitees {
		role := ROLE_VIEWER
		if strings.Contains(permission, "admin") {
			role = ROLE_CO_HOST
		}
		if _, ok := room.Users[u]; !ok {
			users = append(users, User{
				UserId:     u,
				Role:       role,
				Permission: permission,
				Status:     0,
			})
		}
	}
	data, e := json.Marshal(users)
	if e != nil {
		logrus.Infof("marshall failed %v", e)
	}
	result := NewInstruction(MEMBERS, string(data))
	logrus.Infof("members %s", result.String())
	return result
}

func AuthShare(userId, shareSessionId string) (bool, string) {
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
	if r.Creator == user {
		role = ROLE_ADMIN
	} else if strings.Contains(permissions, "admin") {
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

func (r *RdpSessionRoom) StopShare() {
	r.lock.Lock()
	defer r.lock.Unlock()

	for u := range r.Invitees {
		if u == r.Creator {
			continue
		}
		delete(r.Invitees, u)
		_ = dbAccess.RemoveInvitee(r.SessionId, u)
	}

	for _, c := range r.Users {
		if c.UserId == r.Creator {
			continue
		}
		e := c.Websocket.Close()
		if e != nil {
			logrus.Errorf("close %s ws failed %v", c.UserId, e)
		}
		delete(r.Users, c.UserId)
	}

	var users []User
	if data, e := json.Marshal(users); e == nil {
		emptyMembers := NewInstruction(MEMBERS, string(data))
		r.Users[r.Creator].WriteMessage(emptyMembers)
	}
}

func GetRdpSessionRoom(sessionId string) (*RdpSessionRoom, bool) {
	result, ok := rdpRooms[sessionId]
	return result, ok
}

func NewRdpSessionRoom(sessionId string, user string, closer WriterCloser, connectionId string, allowSharing bool, appId, appName string, loggingInfo logging.LoggingInfo) *RdpClient {
	lock.Lock()
	defer lock.Unlock()

	room := &RdpSessionRoom{
		Creator:         user,
		SessionId:       sessionId,
		AppId:           appId,
		loggingInfo:     &loggingInfo,
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
		if len(room.Invitees) >= INVITEE_LIMIT {
			return fmt.Errorf("total invitee reach the limit")
		}
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
			u.WriteMessage(ins)
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("cannot find rdp room by id %s", sessionId)
	}
}

func LeaveRoom(session *session.SessionCommonData, sessionId, user, clientIp, clientPrivateIp string) error {
	lock.Lock()
	defer lock.Unlock()

	if user != session.Email { // only log leave event for joined session user
		logging.Log(logging.Action{
			Session:         session,
			AppTag:          "rdp.leave",
			UserEmail:       user,
			ClientIP:        clientIp,
			ClientPrivateIp: clientPrivateIp,
			Destination:     session.ServerName,
		})
	}

	if room, ok := GetRdpSessionRoom(sessionId); ok {
		room.leave(user)

		hasAdmin := false
		for _, u := range room.Users {
			if u.Role != ROLE_VIEWER {
				hasAdmin = true
			}
			if len(room.Invitees) > 1 {
				members := room.GetMembersInstruction()
				u.WriteMessage(members)
			}
		}
		if !hasAdmin {
			closeRoom(room)
		}
	} else {
		return fmt.Errorf("cannot find rdp room by id %s", sessionId)
	}
	return nil
}

func GetRoomByAppIdAndCreator(appId, creator string) (*RdpSessionRoom, bool) {
	for _, r := range rdpRooms {
		_, exist := r.Users[creator]
		if r.Creator == creator && r.AppId == appId && !exist {
			return r, true
		}
	}
	return nil, false
}

func closeRoom(room *RdpSessionRoom) {
	for _, u := range room.Users {
		logrus.Infof("disconnect user %s", u.UserId)
		u.Websocket.Close()
	}
	ses, _ := SessionDataStore.Get(room.SessionId).(*session.SessionCommonData)

	delete(rdpRooms, room.SessionId)
	SessionDataStore.Delete(room.SessionId)
	e := dbAccess.DeleteRdpSession(room.SessionId)
	e2 := kv.Delete(fmt.Sprintf("guac-%s", room.SessionId))
	logrus.Infof("remove session data %s, room size %d, session store size %d, e %v, e2 %v", room.SessionId, len(rdpRooms), len(SessionDataStore.Data), e, e2)
	room.loggingInfo.SessionId = ses.RdpSessionId
	AddEncodeRecoding(*room.loggingInfo)

	if ses.Auth {
		go SendEvent("exit", logging.Action{
			Session:         ses,
			UserEmail:       room.Creator,
			ClientIP:        ses.ClientIP,
			ClientPrivateIp: ses.ClientPrivateIp,
			Destination:     ses.ServerName,
		})
	}
}
