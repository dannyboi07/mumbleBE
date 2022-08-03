package rbmq

import (
	"encoding/json"
	"mumble-message-service/db"
	"mumble-message-service/redis"
	"mumble-message-service/types"
	"mumble-message-service/utils"

	amqp "github.com/rabbitmq/amqp091-go"
)

var consumeMsgToSaveCh *amqp.Channel

func initSaveMsgXandQ() error {

	var err error
	if err = consumeMsgToSaveCh.ExchangeDeclare(
		"x_save_msg",
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	// Limiting prefetch size to 2000 for caution, this service will be limited by Postgres write speed,
	// supposedly around 3000/sec
	if err = consumeMsgToSaveCh.Qos(
		2000,
		0,
		false,
	); err != nil {
		return err
	}

	var saveMsgQ amqp.Queue
	if saveMsgQ, err = consumeMsgToSaveCh.QueueDeclare(
		"q_save_msg",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	if err = consumeMsgToSaveCh.QueueBind(
		saveMsgQ.Name,
		"",
		"x_save_msg",
		false,
		nil,
	); err != nil {
		return err
	}

	var msgs <-chan amqp.Delivery
	if msgs, err = consumeMsgToSaveCh.Consume(
		saveMsgQ.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}
	go consumeMsgsToSave(msgs)

	// for i := 0; i < 10; i++ {
	// 	go func(i int) {
	// 		utils.Log.Println(i)
	// 	}(i)
	// }

	return nil
}

func consumeMsgsToSave(msgs <-chan amqp.Delivery) {
	utils.Log.Println("Consuming msgs")

	for msg := range msgs {

		go func(goMsg amqp.Delivery, publishMsgChannel *amqp.Channel) {
			utils.Log.Println("Consuming msg")
			var origMsg types.WsMsg

			err := json.Unmarshal(goMsg.Body, &origMsg)
			if err != nil {
				utils.Log.Println("err unmarshalling rbmq msg, err:", err)
				goMsg.Nack(false, true)
				return
			}

			utils.Log.Println("Received msg", origMsg.MsgUUID, origMsg.Type, origMsg.From, origMsg.To, origMsg.Text)

			if origMsg.Type == "msg" {

				if savedMsg, err := db.InsertMessage(origMsg); err != nil {
					goMsg.Nack(false, true)
					utils.Log.Println("failed to save message to db, err:", err)
				} else {
					goMsg.Ack(false)

					// Testing sharing channel across threads here, not thread safe acc to main docs
					// Note: Channel's source code contains mutexes...? Seems safe?
					// Turns out to be thread safe: https://github.com/streadway/amqp/issues/77
					go func(goSavedWsMsg types.WsMsg, goPubMsgToSendCh *amqp.Channel) {

						receiverMsgBytes, err := json.Marshal(types.WsMsg{
							// MsgUUID: goSavedWsMsg.MsgUUID,
							MsgId:  goSavedWsMsg.MsgId,
							Type:   "delivery",
							Status: "",
							From:   goSavedWsMsg.From,
							To:     goSavedWsMsg.To,
							Text:   goSavedWsMsg.Text,
							Time:   goSavedWsMsg.Time,
						})
						if err != nil {
							utils.Log.Println("Failed to marshal msg to receiver, err:", err)
						} else {
							// Queue msg to receiver
							if present, receiverHost, err := redis.GetUserHost(goSavedWsMsg.To); present {
								goPubMsgToSendCh.Publish(
									"x_send_msg",
									receiverHost,
									false,
									false,
									amqp.Publishing{
										ContentType: "application/json",
										Body:        receiverMsgBytes,
									},
								)
							} else if err != nil {
								utils.Log.Println("err getting user's host, err:", err)
								return
							} else if !present {
								// Implement Queue route to user service
								utils.Log.Println("Receiver offline")
							}
						}
					}(savedMsg, publishMsgChannel)

					go func(goSavedWsMsg types.WsMsg, goPubMsgToSendCh *amqp.Channel) {

						senderMsgBytes, err := json.Marshal(types.WsMsg{
							MsgUUID: goSavedWsMsg.MsgUUID,
							MsgId:   goSavedWsMsg.MsgId,
							Type:    "msg_status",
							Status:  "saved",
							From:    goSavedWsMsg.From, // goSavedWsMsg.From,
							To:      goSavedWsMsg.To,   // goSavedWsMsg.To,
							Time:    goSavedWsMsg.Time,
						})
						if err != nil {
							utils.Log.Println("Failed to marshal msg to sender, err:", err)
							return
						}
						// Send "sent" acknowledgement to sender
						if present, senderHost, err := redis.GetUserHost(goSavedWsMsg.From); err != nil {
							utils.Log.Println("err getting user's host, err:", err)
							return
						} else if present {
							if err := goPubMsgToSendCh.Publish(
								"x_send_msg",
								senderHost,
								false,
								false,
								amqp.Publishing{
									ContentType: "application/json",
									Body:        senderMsgBytes,
								},
							); err != nil {
								utils.Log.Println("Failed to publish msg, err:", err)
							}
						} else if !present {
							// Implement Queue route to user service
							utils.Log.Println("Sender offline")
						}
					}(types.WsMsg{
						MsgUUID: origMsg.MsgUUID,
						MsgId:   savedMsg.MsgId,
						From:    savedMsg.From,
						To:      savedMsg.To,
						Time:    savedMsg.Time,
					}, publishMsgChannel)
				}
			} else if origMsg.Type == "msg_status" {
				if err := db.UpdateMessageStatus(origMsg.MsgId, origMsg.Status); err != nil {
					goMsg.Nack(false, true)
					utils.Log.Println("failed to update message status in db, err:", err)
				} else {
					goMsg.Ack(false)

					var (
						senderMsgBytes []byte
						err            error
					)
					if origMsg.Status != "saved" {
						senderMsgBytes, err = json.Marshal(types.WsMsg{
							MsgId:  origMsg.MsgId,
							Type:   "msg_status", // Type: "update_msg_status",
							Status: origMsg.Status,
							From:   origMsg.To, // origMsg.From,
							To:     origMsg.From,
						})
					} else {
						senderMsgBytes, err = json.Marshal(types.WsMsg{
							MsgUUID: string(origMsg.MsgId),
							Type:    "msg_status", // Type: "update_msg_status",
							Status:  origMsg.Status,
							From:    origMsg.To, // origMsg.From,
							To:      origMsg.From,
						})
					}
					if err != nil {
						utils.Log.Fatalln("Failed to marshal status update to sender, err:", err)
						return
					}

					if present, senderHost, err := redis.GetUserHost(origMsg.To); err != nil {
						utils.Log.Println("err getting user's host, err:", err)
						return
					} else if present {
						if err := publishMsgChannel.Publish(
							"x_send_msg",
							senderHost,
							false,
							false,
							amqp.Publishing{
								ContentType: "application/json",
								Body:        senderMsgBytes,
							},
						); err != nil {
							utils.Log.Println("Failed to publish msg update, err:", err)
						}
					} else if !present {
						utils.Log.Println("Sender offline")
					}
				}
			}

		}(msg, publishMsgToSendCh)

	}
	utils.Log.Println("Consumer exiting")
}
