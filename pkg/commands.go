package guac

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/appaegis/golang-common/pkg/config"
	"github.com/appaegis/golang-common/pkg/httpclient"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/wwt/guac/lib/logging"
)

type Command interface {
	Exec(instruction *Instruction, session *SessionCommonData, client *RdpClient) *Instruction
}

var mailService MailService = RdpMailService{}

var commands = make(map[string]Command)

const (
	APPAEGIS_RESP_OP = "appaegis-resp"
	SESSION_SHARE_OP = "session-sharing"
)

func init() {
	commands[SHARE_SESSION] = RequestSharingCommand{}
	commands[DLP_DOWNLOAD] = DlpDownloadCommand{}
	commands[DLP_UPLOAD] = DlpUploadCommand{}
	commands[LOG_DOWNLOAD] = LogDownloadCommand{}
	commands[DOWNLOAD_CHECK] = DownloadCheckCommand{}
	commands[SET_PERMISSONS] = SetPermissions{}
	commands[SEARCH_USER] = SearchUserCommand{}
}

func GetCommandByOp(instruction *Instruction) (Command, error) {
	if len(instruction.Args) <= 0 {
		return nil, fmt.Errorf("invalid instruction")
	}
	if c, ok := commands[instruction.Args[0]]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("invalid op %s", instruction.Args[0])
}

func GetSharingUrl(sessionId string) string {
	url := fmt.Sprintf("https://%s/share_session?sessionId=%s", config.GetPortalHostName(), sessionId)
	return url
}

type SetPermissions struct{}

func (c SetPermissions) Exec(instruction *Instruction, session *SessionCommonData, client *RdpClient) *Instruction {
	if !client.Admin {
		logrus.Errorf("user %s didn't allow to set permissions", client.UserId)
		return nil
	}
	room, ok := GetRdpSessionRoom(session.RdpSessionId)
	if !ok {
		return nil
	}
	for _, str := range instruction.Args[1:] {
		userPermission := strings.Split(str, ":")
		if len(userPermission) != 2 {
			logrus.Errorf("incorrect permission format %s", str)
			continue
		}
		for _, u := range room.Users {
			if u.UserId == userPermission[0] {
				u.Admin = strings.Contains(userPermission[1], "admin")
				u.Keyboard = strings.Contains(userPermission[1], "keyboard")
				u.Mouse = strings.Contains(userPermission[1], "mouse")
			}
		}
	}
	ins := room.GetMembersInstruction()
	for _, u := range room.Users {
		if err := u.Websocket.WriteMessage(websocket.TextMessage, ins.Byte()); err != nil {
			logrus.Errorf("send message %s to user %s failed", ins.String(), u.UserId)
		}
	}
	return nil
}

type SearchUserCommand struct{}

func (c SearchUserCommand) Exec(instruction *Instruction, session *SessionCommonData, client *RdpClient) *Instruction {
	if len(instruction.Args) < 3 {
		logrus.Infof("instruction args err")
		return nil
	}
	prefix := instruction.Args[2]
	users, e := dbAccess.QueryUsersByTenantAndUserPrefix(session.TenantID, prefix)
	if e != nil {
		return nil
	}
	ins := NewInstruction(SESSION_SHARE_OP, SEARCH_USER_ACK, instruction.Args[1])
	for _, u := range users {
		ins.Args = append(ins.Args, u.ID)
	}
	return ins
}

type RequestSharingCommand struct{}

func (c RequestSharingCommand) Exec(instruction *Instruction, session *SessionCommonData, client *RdpClient) *Instruction {
	var e error
	status := "200"
	if e != nil {
		status = "500"
	}

	url := GetSharingUrl(session.RdpSessionId)
	for i := 2; i < len(instruction.Args); i++ {
		var invitee, permissions string
		invitee = instruction.Args[i]
		if i+1 < len(instruction.Args) {
			permissions = instruction.Args[i+1]
		}
		e := AddSharingUser(session.RdpSessionId, invitee, permissions)
		if e != nil {
			logrus.Errorf("add invitee to room failed %v", e)
			continue
		}
		e = dbAccess.ShareRdpSession(invitee, permissions, session.RdpSessionId)
		if e != nil {
			logrus.Errorf("share rdp session to user %s, permission %s, stream %s, failed %v", invitee, permissions, session.RdpSessionId, e)
		}
		e = mailService.SendInvitation(invitee, session.Email, url)
		if e != nil {
			logrus.Errorf("send invitation email to %s failed %v", invitee, e)
		}
	}

	resp := NewInstruction(SESSION_SHARE_OP, "share-session-ack", instruction.Args[1], status, url, "")
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
	fetcher := httpclient.NewResponseParser("POST", fmt.Sprintf("http://%s/event", config.GetDlpClientHost()), map[string]string{
		"Content-Type": "application/json",
	}, map[string]interface{}{
		"appID":      ses.AppID,
		"tenantID":   ses.TenantID,
		"path":       fmt.Sprintf("%s%s", GetDrivePathInEFS(ses.TenantID, ses.AppID, ses.Email), filePath),
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
	return NewInstruction(APPAEGIS_RESP_OP, instruction.Args[1], string(data))
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
	return NewInstruction(APPAEGIS_RESP_OP, instruction.Args[1], string(data))
}

type LogDownloadCommand struct{}

func (c LogDownloadCommand) Exec(instruction *Instruction, ses *SessionCommonData, client *RdpClient) *Instruction {
	fileCount, err := strconv.Atoi(instruction.Args[2])
	if err != nil {
		fileCount = 1
	}
	logging.Log(logging.Action{
		AppTag:    "guac.download",
		TenantID:  ses.TenantID,
		UserEmail: ses.Email,
		AppID:     ses.AppID,
		RoleIDs:   ses.RoleIDs,
		FileCount: fileCount,
		ClientIP:  ses.ClientIP,
	})
	IncrAlertRuleSessionCountByNumber(ses, "download", fileCount)
	result := J{
		"ok":    true,
		"count": fileCount,
	}
	data, _ := json.Marshal(result)
	return NewInstruction(APPAEGIS_RESP_OP, instruction.Args[1], string(data))
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
	return NewInstruction(APPAEGIS_RESP_OP, instruction.Args[1], string(data))
}
