package cmn

import (
	"github.com/gorilla/websocket"
	"time"
	"w2w.io/mux"
)

type OnlineUser struct {
	WsConn *websocket.Conn

	Account  string `json:"account,omitempty"`
	NickName string `json:"nickname,omitempty"`
	Avatar   []byte `json:"avatar,omitempty"`

	ID int64

	LastActivity time.Time `json:"lastActivity,omitempty"`
	LastAction   string    `json:"lastAction,omitempty"`

	Status int8 `json:"status,omitempty"` //0: offline, 2: online, 4: transferring
}

type Online struct {
	Users map[string]*OnlineUser

	MsgTo      func(account string, message string) error
	Broadcast  func(message string) error
	OnlineList func() ([]byte, error)

	Mux *mux.Router
}

var OnlineUsers Online
