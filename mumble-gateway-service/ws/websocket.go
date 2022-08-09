package ws

import (
	"encoding/json"
	"mumble-gateway-service/rbmq"
	"mumble-gateway-service/redis"
	"mumble-gateway-service/types"
	"mumble-gateway-service/utils"
	"mumble-gateway-service/wsclients"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
	amqp "github.com/rabbitmq/amqp091-go"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// origin := r.Header.Get("Origin")
		return r.Header.Get("Origin") == "https://mumble.daniel-dev.tech"
		// return origin == "http://localhost:3000" || origin == "http://localhost:8080"
	},
}

func Handler(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value("userDetails").(jwt.MapClaims)["UserId"].(int64)

	if userPresent := wsclients.WsClients.Exists(userId); userPresent {
		http.Error(w, "User already connected", http.StatusForbidden)
		utils.Log.Println("client error: user exists in connection list", r.RemoteAddr)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Error establishing Websocket connection", http.StatusInternalServerError)
		utils.Log.Println("ws cntrl error: upgrading WS connection", err)
		return
	}

	wsclients.WsClients.AddConn(userId, conn)

	err = redis.SetUserOnline(userId)
	if err != nil {
		utils.Log.Println("ws cntrl error: setting user online in redis", err)
	}

	err = rbmq.PubLastSeen(userId, types.UserLastSeen{UserLastSeenTime: "Online"})
	if err != nil {
		utils.Log.Println("err publishing user's status, err:", err)
	}

	conn.SetReadLimit(4096)

	utils.Log.Println("Client connected, User Id:", userId)
	go wsConnHandler(conn, userId)
}

func wsConnHandler(conn *websocket.Conn, userId int64) {
	defer conn.Close()
	contactIdChan := make(chan int64, 1)
	closeChildChan := make(chan bool)
	closeParentChan := make(chan bool)
	go wsPubSubHandler(userId, contactIdChan, closeChildChan, closeParentChan)

forLoop:
	for {
		select {
		case <-closeParentChan:
			utils.Log.Println("ws: parent goroutine rcvd error from sub handling-child")
			go wsPubSubHandler(userId, contactIdChan, closeChildChan, closeParentChan)
			continue forLoop
		default:
			var message types.WsMsg
			err := conn.ReadJSON(&message)
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseGoingAway) {
					utils.Log.Println("err reading msg from client, err:", err)
				}
				break forLoop
			} else if message.Type == nil || message.To == nil || message.From == nil {
				utils.Log.Println("ws-client: missing fields in ws message", conn.RemoteAddr())
				break forLoop
			} else {
				// Verifying whether the "from" field in WS message sent from client side actually exists in the connection list.
				if senderIsPresent := wsclients.WsClients.Exists(*message.From); !senderIsPresent {
					utils.Log.Println("ws-client: Sender missing from list of connections, sent uId", message.From, conn.RemoteAddr())
					break forLoop
				}

				switch *message.Type {
				case "msg":
					if message.MsgUUID == nil || message.Text == nil {
						utils.Log.Println("ws-client: missing fields in ws message", conn.RemoteAddr())
						break forLoop
					}
					utils.Log.Println("Publishing message to be saved...", *message.Text)

					// Send message to message service to be saved
					go func(goMessage types.WsMsg) {
						if err := rbmq.PublishMsg(goMessage); err != nil {

							// Notify client to drop the message in case of an error
							// Or retry sending it, if the feature is implemented
							errType := "msg_status"
							errStatus := "save_err"
							wsclients.WsClients.ExistsAndSendMsg(*goMessage.From, types.WsMsg{
								MsgUUID: goMessage.MsgUUID,
								From:    goMessage.From,
								To:      goMessage.To,
								Type:    &errType,
								Status:  &errStatus,
							})
							utils.Log.Println("err publishing message to be saved, err:", err)
						}
					}(message)
				case "msg_status":
					if message.MsgId == nil || message.Status == nil {
						utils.Log.Println("ws-client: missing fields in ws message, userId:", userId, conn.RemoteAddr())
						break forLoop
					} else if *message.Status != "read" && *message.Status != "del" {
						utils.Log.Println("ws-client: received invalid message status value, userId:", userId, conn.RemoteAddr())
						break forLoop
					}
					utils.Log.Println("Publishing message update...")

					go func(goMessage types.WsMsg) {

						if err := rbmq.PublishMsg(goMessage); err != nil {

							// Notify client to drop the message in case of an error
							// Or retry sending it, if the feature is implemented
							errType := "msg_status"
							errStatus := "save_err"
							wsclients.WsClients.ExistsAndSendMsg(*goMessage.From, types.WsMsg{
								MsgId:  goMessage.MsgId,
								Type:   &errType,
								Status: &errStatus,
								From:   goMessage.From,
								To:     goMessage.To,
							})
							utils.Log.Println("err publishing message status update, err:", err)
						}
					}(types.WsMsg{
						MsgId:  message.MsgId,
						Type:   message.Type,
						Status: message.Status,
						From:   message.From,
						To:     message.To})

				case "get_lst_sn":

					if contactStatus, err := redis.GetUserStatus(*message.To); err == nil {
						wsclients.WsClients.Lock()
						conn.WriteJSON(types.UserLastSeen{UserLastSeenTime: contactStatus})
						wsclients.WsClients.Unlock()
					}

					contactIdChan <- *message.To
					continue forLoop
				default:
					utils.Log.Println("Unrecognized type value in ws msg field")
					break forLoop
				}
			}
		}
	}
	closeChildChan <- true
	utils.Log.Println("Client going offline, User Id:", userId)
	wsclients.WsClients.DelConn(userId)

	userOfflineTime := time.Now().Format("2006-01-02T15:04:05Z07:00")
	err := redis.SetUserOffline(userId, userOfflineTime)
	if err != nil {
		utils.Log.Println("err setting user as offline in redis, err:", err)
	}

	err = rbmq.PubLastSeen(userId, types.UserLastSeen{UserLastSeenTime: userOfflineTime})
	if err != nil {
		utils.Log.Println("err publishing userLastSeen, err:", err)
	}
}

func wsPubSubHandler(userId int64, contactIdChan <-chan int64, closeChildChan <-chan bool, closeParentChan chan<- bool) {
	utils.Log.Println("last seen handler started...", userId)
	resubscribe := false

	var globalContactId int64
	var lastSeenChan <-chan amqp.Delivery
forLoop:
	for {
		if resubscribe {
			utils.Log.Println("last seen handler resubscribing", userId, globalContactId)
			var err error
			lastSeenChan, err = rbmq.SubToLastSeenQ(userId, globalContactId)
			if err != nil {
				utils.Log.Println("err subbing to last seen q, err:", err)
			}
			resubscribe = false
		}

		select {
		case contactId := <-contactIdChan:
			if contactId != globalContactId {
				utils.Log.Println("last seen handler changing contactId", contactId)
				err := rbmq.CancelSubToLastSeen(userId)
				if err != nil {
					utils.Log.Println("err cancelling to sub to last seen, err:", err)
				}
				globalContactId = contactId
				resubscribe = true
				// Fix for amqp.Delivery corruption in chan consumption
				continue forLoop
			}

		case lastSeenMsg := <-lastSeenChan:
			if !resubscribe {
				var (
					lastSeen types.UserLastSeen
					err      error
				)

				err = json.Unmarshal(lastSeenMsg.Body, &lastSeen)
				if err != nil {
					utils.Log.Println("err unmarshalling last seen, contactIds:", userId, globalContactId, "err:", err)
					break forLoop
				} else {
					if online, err := wsclients.WsClients.ExistsAndSendLastSeen(userId, lastSeen); err != nil {
						utils.Log.Println("err sending last seen to user, err:", err)
					} else if !online {
						utils.Log.Println("user not present in list of connections, disconnecting...")
						break forLoop
					}
				}
			}
		case <-closeChildChan:
			utils.Log.Println("last seen handler notified to close...")
			break forLoop
		}
	}
	go func() {
		if err := rbmq.CancelSubToLastSeen(userId); err != nil {
			utils.Log.Println("err cancelling sub, err:", err)
		}
	}()

	utils.Log.Println("last seen handler closing...")
	closeParentChan <- true
}
