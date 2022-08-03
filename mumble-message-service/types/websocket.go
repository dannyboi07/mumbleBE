package types

import (
	"time"
)

// type WsClients struct {
// 	ClientConns map[int64]*websocket.Conn
// 	sync.RWMutex
// }

// Comes as input from a websocket connection
// type WsMessage struct {
// 	MessageUUID int64  `json:"message_uuid"`
// 	HostSender  string `json:"-"`
// 	Type        string `json:"type"`
// 	From        int64  `json:"from"`
// 	To          int64  `json:"to"`
// 	Text        string `json:"text"`
// }

type WsMsg struct {
	// These fields will come in as input from a WS connection
	MsgUUID string `json:"msg_uuid"`
	Text    string `json:"text,omitempty"`

	// Type field will go both ways
	Type   string `json:"type"`
	From   int64  `json:"from"`
	To     int64  `json:"to"`
	MsgId  int64  `json:"msg_id,omitempty"`
	Status string `json:"status,omitempty"`

	// These fields will go out as output
	Time time.Time `json:"time,omitempty"`
}

// Message that's sent after saving the message in db
type Message struct {
	MessageId int64     `json:"message_id"`
	From      int64     `json:"from"`
	To        int64     `json:"to"`
	Text      string    `json:"text"`
	Time      time.Time `json:"time"`
}

type MqMsg struct {
	MqMsgType string `json:"mq_msg_type"`
	WsMsg
}
