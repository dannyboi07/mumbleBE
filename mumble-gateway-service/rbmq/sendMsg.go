package rbmq

import (
	"encoding/json"
	"errors"
	"mumble-gateway-service/types"
	"mumble-gateway-service/utils"
	"mumble-gateway-service/wsclients"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// var conSendMsgCh *amqp.Channel

type sendMsg struct {
	consumeCh        *amqp.Channel
	isConReady       bool
	notifyConChClose chan *amqp.Error
	closeReconn      chan bool
}

var sendMsgClient sendMsg

func initSendMsg() error {
	sendMsgClient = sendMsg{}

	// utils.Log.Println("Initing initsendmsgcon")
	err := initSendMsgCon(false)
	if err != nil {
		return err
	}

	// utils.Log.Println("initing x and q")
	err = initSendMsgXAndQ()
	if err != nil {
		utils.Log.Println("Failed to initialize xchange and q's to send messages")
		return err
	}

	// utils.Log.Println("go-ing")
	go reconnSendMsg()

	// utils.Log.Println("returning")
	return nil
}

func reconnSendMsg() {
	for {
		time.Sleep(5 * time.Second)

		select {
		case <-sendMsgClient.notifyConChClose:
			err := initSendMsgCon(true)
			if err != nil {
				utils.Log.Println("Failed to reinitialize channel consuming msgs to be sent, err:", err)
			}
		case <-sendMsgClient.closeReconn:
			return
		}
	}
}

func initSendMsgCon(recovering bool) error {
	if !client.isConReady {
		return errors.New("RabbitMq client's connection for consumption is not ready")
	}
	sendMsgClient.isConReady = false

	// utils.Log.Println("opening channel for consumption")
	var err error
	sendMsgClient.consumeCh, err = client.consumeConn.Channel()
	if err != nil {
		utils.Log.Println("err opening channel to consume msgs", err)
		return err
	}

	// utils.Log.Println("setting up notifs")
	sendMsgClient.notifyConChClose = make(chan *amqp.Error)
	sendMsgClient.consumeCh.NotifyClose(sendMsgClient.notifyConChClose)
	sendMsgClient.isConReady = true

	if recovering {
		// utils.Log.Println("initing consumption")
		var msgsChan <-chan amqp.Delivery
		msgsChan, err = sendMsgClient.consumeCh.Consume(
			"q_send_msg_"+utils.Hostname,
			"",
			false,
			true,
			false,
			false,
			nil,
		)
		utils.Log.Println(err)
		if err != nil {
			return err
		}
		// utils.Log.Println("go-ing consumption")
		go consumeMsg(msgsChan)
	}

	return nil
}

func initSendMsgXAndQ() error {
	var err error
	// Consumption Handling
	if err = sendMsgClient.consumeCh.ExchangeDeclare(
		"x_send_msg",
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	// var sendWsMsgQ amqp.Queue
	qName := "q_send_msg_" + utils.Hostname
	if _, err = sendMsgClient.consumeCh.QueueDeclare(
		qName,
		false,
		true,
		true,
		false,
		nil,
	); err != nil {
		return err
	}

	if err = sendMsgClient.consumeCh.QueueBind(
		qName,          // q_send_msg_asdio128as
		utils.Hostname, // Binding key is the container hostname
		"x_send_msg",
		false,
		nil,
	); err != nil {
		return err
	}

	utils.Log.Println("initing consumption")
	var msgsChan <-chan amqp.Delivery
	msgsChan, err = sendMsgClient.consumeCh.Consume(
		"q_send_msg_"+utils.Hostname,
		"",
		false,
		true,
		false,
		false,
		nil,
	)
	utils.Log.Println(err)
	if err != nil {
		return err
	}
	utils.Log.Println("go-ing consumption")
	go consumeMsg(msgsChan)

	// var msgsChan <-chan amqp.Delivery
	// msgsChan, err = sendMsgClient.consumeCh.Consume(
	// 	sendWsMsgQ.Name,
	// 	"",
	// 	false,
	// 	true,
	// 	false,
	// 	false,
	// 	nil,
	// )
	// if err != nil {
	// 	return err
	// }
	// go consumeMsg(msgsChan)

	return nil
}

// Note: Func below which executes as goroutine will exit when an exception occurs
// on the channel from which it is consuming. No need to handle it's exit
func consumeMsg(msgsChan <-chan amqp.Delivery) {
	for msg := range msgsChan {

		go func(goMsg amqp.Delivery) {
			utils.Log.Println("Consuming msg")

			var mqMsg types.WsMsg
			err := json.Unmarshal(goMsg.Body, &mqMsg)
			if err != nil {
				utils.Log.Println("Unable to unmarshal msg to send")
			}

			if *mqMsg.Type == "delivery" {
				utils.Log.Println("Delivering msg")
				online, err := wsclients.WsClients.ExistsAndSendMsg(*mqMsg.To, mqMsg)
				if err != nil {
					goMsg.Nack(false, true)
					utils.Log.Println("Failed to deliver msg to ws connection, err:", err)
				} else if !online {
					// Post on queue to user service for notification
					utils.Log.Println("user offline", online)
				} else {
					goMsg.Ack(false)
				}

			} else if *mqMsg.Type == "msg_status" {
				utils.Log.Println("Delivering msg status", *mqMsg.Type, *mqMsg.MsgUUID)
				var (
					online bool
					err    error
				)
				// Send "saved" ack to sender
				online, err = wsclients.WsClients.ExistsAndSendMsg(*mqMsg.From, mqMsg)
				if err != nil {
					goMsg.Nack(false, true)
					utils.Log.Println("Failed to deliver status update on msg to ws connection, err:", err)
				} else if !online {
					// Post on queue to user service
					utils.Log.Println("user offline", online)
				} else {
					goMsg.Ack(false)
				}
			}
		}(msg)
	}
}

func closeSendMsg() {
	sendMsgClient.closeReconn <- true
	sendMsgClient.isConReady = false

	sendMsgClient.consumeCh.Close()
}

//|| *mqMsg.Type == "update_msg_status" {
// if *mqMsg.Type == "msg_status" {
// testBytes, _ := json.Marshal(mqMsg)
// utils.Log.Println(testBytes)
// Send msg status to sender, From and To fields are flipped around
// Going to phase this out
// }
// else if *mqMsg.Type == "update_msg_status" {
// 	online, err = wsclients.WsClients.ExistsAndSendMsg(*mqMsg.To, types.WsMsg{
// 		MsgId:  mqMsg.MsgId,
// 		Type:   mqMsg.Type,
// 		Status: mqMsg.Status,
// 		From:   mqMsg.To,
// 		To:     mqMsg.From,
// 	})
// }
