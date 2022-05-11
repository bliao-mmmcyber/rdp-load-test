package guac

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/appaegis/golang-common/pkg/config"
	"github.com/appaegis/golang-common/pkg/dynamodbcli"
	"github.com/appaegis/golang-common/pkg/httpclient"
	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/logging"
)

const INVITEE_LIMIT = 4

type Command interface {
	Exec(instruction *Instruction, session *SessionCommonData, client *RdpClient) *Instruction
}

var mailService MailService = RdpMailService{}

var commands = make(map[string]Command)

func init() {
	commands[SHARE_SESSION] = RequestSharingCommand{}
	commands[DLP_DOWNLOAD] = DlpDownloadCommand{}
	commands[DLP_UPLOAD] = DlpUploadCommand{}
	commands[LOG_DOWNLOAD] = LogDownloadCommand{}
	commands[DOWNLOAD_CHECK] = DownloadCheckCommand{}
	commands[SET_PERMISSONS] = SetPermissions{}
	commands[SEARCH_USER] = SearchUserCommand{}
	commands[REMOVE_SHARE] = RemoveShareCommand{}
	commands[CHECK_USER] = CheckUserCommand{}
	commands[STOP_SHARE] = StopShareCommand{}
}

func GetCommandByOp(instruction *Instruction) (Command, error) {
	if len(instruction.Args) <= 0 {
		return nil, fmt.Errorf("invalid instruction")
	}
	if c, ok := commands[instruction.Args[1]]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("invalid op %s", instruction.Args[1])
}

func GetSharingUrl(sessionId string) string {
	url := fmt.Sprintf("https://%s/share_session?shareSessionId=%s", config.GetPortalHostName(), sessionId)
	return url
}

type StopShareCommand struct{}

func (c StopShareCommand) Exec(instruction *Instruction, session *SessionCommonData, client *RdpClient) *Instruction {
	requestId := instruction.Args[0]
	if client.Role != ROLE_ADMIN {
		logrus.Errorf("%s is not host user, cannot stop sharing", client.UserId)
		return getResponseCommand(requestId, "401")
	}
	if room, ok := GetRdpSessionRoom(session.RdpSessionId); ok {
		room.StopShare()
		return getResponseCommand(requestId, "200")
	} else {
		logrus.Errorf("cannot find room by session id %s", session.RdpSessionId)
		return getResponseCommand(requestId, "400")
	}
}

type CheckUserCommand struct{}

func (c CheckUserCommand) Exec(instruction *Instruction, session *SessionCommonData, client *RdpClient) *Instruction {
	userId := instruction.Args[2]
	u := dynamodbcli.Singleon().QueryUserById(userId)
	status := "200"
	if u == nil || u.ID == "" {
		status = "404"
	}
	return getResponseCommand(instruction.Args[0], status)
}

type RemoveShareCommand struct{}

func (c RemoveShareCommand) Exec(instruction *Instruction, session *SessionCommonData, client *RdpClient) *Instruction {
	room, ok := GetRdpSessionRoom(session.RdpSessionId)
	requestId := instruction.Args[0]
	if len(instruction.Args) < 3 || !ok {
		logrus.Errorf("args len %d, room exist %v", len(instruction.Args), ok)
		return getResponseCommand(requestId, "500")
	}
	var err error
	for _, u := range instruction.Args[2:] {
		logrus.Infof("remove user %s from session %s", u, session.RdpSessionId)
		if room.Creator == u {
			logrus.Errorf("cannot remove rdp host user %s from session %s", u, session.RdpSessionId)
			continue
		}

		// notify removed user
		if removedUser, ok := room.Users[u]; ok {
			removeCmd := NewInstruction(REMOVE_SHARE)
			removedUser.WriteMessage(removeCmd)
		}

		room.RemoveUser(u)
		if e := dbAccess.RemoveInvitee(session.RdpSessionId, u); e != nil {
			err = e
			logrus.Errorf("remove invitee failed %s %s, e %v", session.RdpSessionId, u, e)
		}

	}
	if r, ok := GetRdpSessionRoom(session.RdpSessionId); ok {
		members := r.GetMembersInstruction()
		for _, u := range r.Users {
			u.WriteMessage(members)
		}
	}
	status := "200"
	if err != nil {
		status = "500"
	}
	return getResponseCommand(requestId, status)
}

type SetPermissions struct{}

func (c SetPermissions) Exec(instruction *Instruction, session *SessionCommonData, client *RdpClient) *Instruction {
	if client.Role == ROLE_VIEWER {
		logrus.Errorf("user %s didn't have permission to set permissions", client.UserId)
		return getResponseCommand(instruction.Args[0], "403")
	}
	room, ok := GetRdpSessionRoom(session.RdpSessionId)
	if !ok {
		return getResponseCommand(instruction.Args[0], "404")
	}
	for _, str := range instruction.Args[2:] {
		userPermission := strings.Split(str, ":")
		if len(userPermission) != 2 {
			logrus.Errorf("incorrect permission format %s", str)
			continue
		}
		user := userPermission[0]
		permission := userPermission[1]
		logrus.Infof("set permissions %s for user %s", permission, user)
		for _, u := range room.Users {
			if u.UserId == user {
				role := ROLE_VIEWER
				if strings.Contains(permission, "admin") {
					role = ROLE_CO_HOST
				}
				u.Role = role
				u.Keyboard = strings.Contains(permission, "keyboard")
				u.Mouse = strings.Contains(permission, "mouse")
			}
		}

		for u := range room.Invitees {
			if u == user {
				logrus.Infof("update permission for invitee %s to %s", user, permission)
				room.Invitees[user] = permission
			}
		}
		e := dbAccess.ShareRdpSession(user, permission, room.SessionId)
		if e != nil {
			logrus.Errorf("update permission for %s failed %v", user, e)
		}
	}
	ins := room.GetMembersInstruction()
	for _, u := range room.Users {
		u.WriteMessage(ins)
	}

	return getResponseCommand(instruction.Args[0], "200")
}

type SearchUserResp struct {
	Users []string `json:"users"`
}

type SearchUserCommand struct{}

func (c SearchUserCommand) Exec(instruction *Instruction, session *SessionCommonData, client *RdpClient) *Instruction {
	if len(instruction.Args) < 3 {
		logrus.Infof("instruction args err")
		return nil
	}
	prefix := strings.TrimSpace(instruction.Args[2])
	logrus.Infof("search user %s %s", session.TenantID, prefix)
	var result []string
	users, e := dbAccess.QueryUsersByTenantAndUserPrefix(session.TenantID, prefix)
	if e != nil {
		return nil
	}
	for _, u := range users {
		result = append(result, u.ID)
	}
	data, e := json.Marshal(SearchUserResp{
		Users: result,
	})
	if e != nil {
		logrus.Errorf("marshall search user result failed %v", e)
	}
	ins := NewInstruction(APPAEGIS_RESP_OP, instruction.Args[0], string(data))
	return ins
}

type RequestSharingCommand struct{}

func (c RequestSharingCommand) Exec(instruction *Instruction, session *SessionCommonData, client *RdpClient) *Instruction {
	var err error
	status := "200"

	url := GetSharingUrl(session.RdpSessionId)
	for i := 2; i < len(instruction.Args); i++ {
		strs := strings.Split(instruction.Args[i], ":")
		if len(strs) != 2 {
			logrus.Errorf("incorrect format of sharing user %s", instruction.Args[1])
			continue
		}
		invitee := strs[0]
		permissions := strs[1]
		if invitee == "" {
			logrus.Errorf("invitee should not be empty")
			continue
		}
		logrus.Infof("add sharing %s %s", invitee, permissions)
		e := AddInvitee(session.RdpSessionId, invitee, permissions)
		if e != nil {
			logrus.Errorf("add invitee to room failed %v", e)
			err = e
			continue
		}
		e = dbAccess.ShareRdpSession(invitee, permissions, session.RdpSessionId)
		if e != nil {
			err = e
			logrus.Errorf("share rdp session to user %s, permission %s, stream %s, failed %v", invitee, permissions, session.RdpSessionId, e)
		}
		e = mailService.SendInvitation(invitee, session.Email, url, session.AppName)
		if e != nil {
			err = e
			logrus.Errorf("send invitation email to %s failed %v", invitee, e)
		}
	}

	if r, ok := GetRdpSessionRoom(session.RdpSessionId); ok {
		members := r.GetMembersInstruction()
		for _, u := range r.Users {
			u.WriteMessage(members)
		}
	}
	if err != nil {
		status = "500"
	}
	payload := make(map[string]string)
	payload["status"] = status
	payload["url"] = url
	data, e := json.Marshal(payload)
	if e != nil {
		logrus.Errorf("error marshall %v", e)
	}
	resp := NewInstruction(APPAEGIS_RESP_OP, instruction.Args[0], string(data))
	return resp
}

type DlpDownloadCommand struct{}

func (c DlpDownloadCommand) Exec(instruction *Instruction, ses *SessionCommonData, client *RdpClient) *Instruction {
	filePath := instruction.Args[2]
	logrus.Debug("dlp-download: ", filePath)
	fileTokens := strings.Split(filePath, "/")
	fileName := fileTokens[0]
	if len(fileTokens) > 0 {
		fileName = fileTokens[len(fileTokens)-1]
	}
	fullPath := fmt.Sprintf("%s%s", GetDrivePathInEFS(ses.TenantID, ses.AppID, ses.Email), filePath)
	if info, e := os.Stat(fullPath); e == nil {
		if info.Size() == 0 {
			logrus.Infof("file %s size is 0", filePath)
			result := J{
				"ok": true,
			}
			data, _ := json.Marshal(result)
			return NewInstruction(APPAEGIS_RESP_OP, instruction.Args[0], string(data))
		}
	}

	fetcher := httpclient.NewResponseParser("POST", fmt.Sprintf("http://%s/event", config.GetDlpClientHost()), map[string]string{
		"Content-Type": "application/json",
	}, map[string]interface{}{
		"appID":      ses.AppID,
		"tenantID":   ses.TenantID,
		"path":       fullPath,
		"user":       ses.Email,
		"service":    "rdp",
		"actionType": "download",
		"location":   ses.ClientIsoCountry,
		"appName":    ses.AppName,
		"fileName":   fileName,
		"timestamp":  time.Now().UnixNano() / 1000000,
	})
	fetcher.Do()
	result := J{
		"ok": true,
	}
	data, _ := json.Marshal(result)
	return NewInstruction(APPAEGIS_RESP_OP, instruction.Args[0], string(data))
}

type DlpUploadCommand struct{}

func (c DlpUploadCommand) Exec(instruction *Instruction, ses *SessionCommonData, client *RdpClient) *Instruction {
	fileName := instruction.Args[2]
	logrus.Debug("dlp-upload: ", fileName)

	fetcher := httpclient.NewResponseParser("POST", fmt.Sprintf("http://%s/event", config.GetDlpClientHost()), map[string]string{
		"Content-Type": "application/json",
	}, map[string]interface{}{
		"appID":      ses.AppID,
		"tenantID":   ses.TenantID,
		"path":       fmt.Sprintf("%s/%s", GetDrivePathInEFS(ses.TenantID, ses.AppID, ses.Email), fileName),
		"user":       ses.Email,
		"service":    "rdp",
		"actionType": "upload",
		"location":   ses.ClientIsoCountry,
		"appName":    ses.AppName,
		"fileName":   fileName,
		"timestamp":  time.Now().UnixNano() / 1000000,
	})
	fetcher.Do()

	result := J{
		"ok": true,
	}
	data, _ := json.Marshal(result)
	return NewInstruction(APPAEGIS_RESP_OP, instruction.Args[0], string(data))
}

type LogDownloadCommand struct{}

func (c LogDownloadCommand) Exec(instruction *Instruction, ses *SessionCommonData, client *RdpClient) *Instruction {
	fileCount, err := strconv.Atoi(instruction.Args[2])
	if err != nil {
		fileCount = 1
	}
	logging.Log(logging.Action{
		AppTag:       "guac.download",
		TenantID:     ses.TenantID,
		UserEmail:    ses.Email,
		AppID:        ses.AppID,
		RoleIDs:      ses.RoleIDs,
		FileCount:    fileCount,
		ClientIP:     ses.ClientIP,
		RdpSessionId: ses.RdpSessionId,
	})
	IncrAlertRuleSessionCountByNumber(ses, "download", fileCount)
	result := J{
		"ok":    true,
		"count": fileCount,
	}
	data, _ := json.Marshal(result)
	return NewInstruction(APPAEGIS_RESP_OP, instruction.Args[0], string(data))
}

type DownloadCheckCommand struct{}

func (c DownloadCheckCommand) Exec(instruction *Instruction, ses *SessionCommonData, client *RdpClient) *Instruction {
	fileCount, err := strconv.Atoi(instruction.Args[2])
	if err != nil {
		fileCount = 1
	}
	var result J
	ok, block := CheckAlertRule(ses, "download", fileCount)
	if !ok && block {
		result = J{
			"ok": false,
		}
	} else {
		result = J{
			"ok":     true,
			"prompt": !ok,
		}
	}
	data, _ := json.Marshal(result)
	return NewInstruction(APPAEGIS_RESP_OP, instruction.Args[0], string(data))
}

func getResponseCommand(requestId string, status string) *Instruction {
	payload := map[string]string{
		"status": status,
	}
	data, _ := json.Marshal(payload)
	return NewInstruction(APPAEGIS_RESP_OP, requestId, string(data))
}
