package guac

import (
	"sync"

	"github.com/gorilla/websocket"
)

type WrappedWebSocket struct {
	*websocket.Conn
	mux sync.Mutex
}

func NewWrappedWebSocket(con *websocket.Conn) *WrappedWebSocket {
	r := WrappedWebSocket{
		Conn: con,
		mux:  sync.Mutex{},
	}
	return &r
}

func (s *WrappedWebSocket) WriteMessage(messageType int, data []byte) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.Conn.WriteMessage(messageType, data)
}
