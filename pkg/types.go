package guac

import (
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var counter = 0

type Message struct {
	counter     int
	messageType int
	data        []byte
}
type WrappedWebSocket struct {
	*websocket.Conn
	sendCh chan Message
}

func NewWrappedWebSocket(con *websocket.Conn) *WrappedWebSocket {
	r := WrappedWebSocket{
		Conn:   con,
		sendCh: make(chan Message, 1024),
	}
	go r.send()
	return &r
}

func (s *WrappedWebSocket) WriteMessage(messageType int, data []byte) error {
	counter++
	logrus.Infof("push message %d", counter)
	s.sendCh <- Message{
		counter:     counter,
		messageType: messageType,
		data:        data,
	}
	return nil
	//return s.Conn.WriteMessage(messageType, data)
}

func (s *WrappedWebSocket) send() {
	logrus.Infof("start send")
	for {
		m, ok := <-s.sendCh
		if !ok {
			logrus.Infof("send chan closed")
			return
		}
		logrus.Infof("send message %d", m.counter)
		if e := s.Conn.WriteMessage(m.messageType, m.data); e != nil {
			logrus.Errorf("write message failed %v", e)
		}
	}
}
