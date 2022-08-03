package types

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WsClients struct {
	ClientConns map[int64]*websocket.Conn
	sync.RWMutex
}

func (c *WsClients) AddConn(userid int64, conn *websocket.Conn) {
	c.RWMutex.Lock()
	c.ClientConns[userid] = conn
	c.RWMutex.Unlock()
}

func (c *WsClients) DelConn(userId int64) {
	c.RWMutex.Lock()
	delete(c.ClientConns, userId)
	c.RWMutex.Unlock()
}

func (c *WsClients) Exists(userId int64) bool {
	c.RWMutex.RLock()
	_, online := c.ClientConns[userId]
	c.RWMutex.RUnlock()

	return online
}

func (c *WsClients) ExistsAndSendLastSeen(userId int64, lastSeen UserLastSeen) (bool, error) {
	var (
		conn   *websocket.Conn
		online bool
	)
	c.RWMutex.RLock()
	conn, online = c.ClientConns[userId]
	c.RWMutex.RUnlock()
	if !online {
		return false, nil
	}

	c.RWMutex.Lock()
	err := conn.WriteJSON(lastSeen)
	c.RWMutex.Unlock()
	if err != nil {
		return true, err
	}

	return true, nil
}

// bool return indicates if the connection to send to is present
func (c *WsClients) ExistsAndSendMsg(userId int64, msg WsMsg) (bool, error) {
	if msg.Text != nil {
		fmt.Println("msg text", *msg.Text)
	}

	var (
		conn   *websocket.Conn
		online bool
	)
	c.RWMutex.Lock()
	conn, online = c.ClientConns[userId]
	c.RWMutex.Unlock()
	if !online {
		return false, nil
	}

	c.RWMutex.RLock()
	err := conn.WriteJSON(msg)
	c.RWMutex.RUnlock()
	if err != nil {
		return true, err
	}

	return true, nil
}

// Comes as input from a websocket connection
type WsMsg struct {
	// These fields will come in as input from a WS connection
	MsgUUID *string `json:"msg_uuid"`
	Text    *string `json:"text,omitempty"`

	// These fields will go both ways
	Type   *string `json:"type"`
	From   *int64  `json:"from"`
	To     *int64  `json:"to"`
	MsgId  *int64  `json:"msg_id"`
	Status *string `json:"status"`

	// These fields will go out as output
	Time time.Time `json:"time,omitempty"`
}

// Will come from the "x_send_msg" message queue
type MqSendMsg struct {
	MqMsgType string `json:"mq_msg_type"`
	WsMsg
}

type WsMsgOut struct {
	MsgId   int64     `json:"msg_id"`
	MsgUUID int64     `json:"msg_uuid"`
	Type    string    `json:"type"`
	Status  string    `json:"status"`
	From    int64     `json:"from,omitempty"`
	To      int64     `json:"to,omitempty"`
	Text    string    `json:"text,omitempty"`
	Time    time.Time `json:"time,omitempty"`
}

type WsAck struct {
	MsgUUID int64  `json:"msg_uuid"`
	MsgId   int64  `json:"msg_id"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	From    int64  `json:"from"`
	To      int64  `json:"to"`
}

// type WsMessage struct {
// 	Type *string `json:"type"`
// 	From *int64  `json:"from"`
// 	To   *int64  `json:"to"`
// 	Text *string `json:"text"`
// }

// Message that's sent after saving the message in db
type Message struct {
	MessageId int64     `json:"message_id"`
	From      int64     `json:"from"`
	To        int64     `json:"to"`
	Text      string    `json:"text"`
	Time      time.Time `json:"time"`
}
