package rbmq

import (
	"encoding/json"
	"errors"
	"mumble-gateway-service/types"
	"mumble-gateway-service/utils"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Channel for publishing msgs to be saved to db
// var pubSaveMsgCh *amqp.Channel

type saveMsg struct {
	publishCh        *amqp.Channel
	isPubReady       bool
	notifyPubChClose chan *amqp.Error
	closeReconn      chan bool
}

var saveMsgClient saveMsg

func initSaveMsg() error {
	saveMsgClient = saveMsg{}

	err := initSaveMsgPub()
	if err != nil {
		return err
	}

	err = initSaveMsgXAndQ()
	if err != nil {
		utils.Log.Println("Failed to initialize xchange and q's to save messages")
		return err
	}

	go reconnSaveMsg()

	return nil
}

func initSaveMsgPub() error {
	if !client.isPubReady {
		return errors.New("RabbitMq client publish connection not ready")
	}
	saveMsgClient.isPubReady = false

	var err error
	saveMsgClient.publishCh, err = client.publishConn.Channel()
	if err != nil {
		utils.Log.Println("err opening channel to publish msgs", err)
		return err
	}

	saveMsgClient.notifyPubChClose = make(chan *amqp.Error)
	saveMsgClient.publishCh.NotifyClose(saveMsgClient.notifyPubChClose)
	saveMsgClient.isPubReady = true

	return nil
}

func initSaveMsgXAndQ() error {
	var err error

	if saveMsgClient.publishCh, err = client.publishConn.Channel(); err != nil {
		utils.Log.Println("err opening channel for publishing msgs to be saved")
		return err
	}

	if err = saveMsgClient.publishCh.ExchangeDeclare(
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

	var saveMsgQ amqp.Queue
	// Declaring queue, incase it doesn't exist, it's idempotent anyways
	if saveMsgQ, err = saveMsgClient.publishCh.QueueDeclare(
		"q_save_msg",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	// saveMsgQName = saveMsgQ.Name
	// Bind declared queue to the declared exchange
	if err = saveMsgClient.publishCh.QueueBind(
		saveMsgQ.Name, // Name
		"",            // saveMsgQPub.Name, // Key
		"x_save_msg",  // XChange
		false,         // noWait
		nil,           // args
	); err != nil {
		return err
	}

	return nil
}

func reconnSaveMsg() {
	for {
		time.Sleep(5 * time.Second)

		select {
		case <-saveMsgClient.notifyPubChClose:
			err := initSaveMsgPub()
			if err != nil {
				utils.Log.Println("Failed to reinitialize channel publishing msgs to be saved, err:", err)
			}
		case <-saveMsgClient.closeReconn:
			return
		}
	}
}

func PublishMsg(msg types.WsMsg) error {
	var (
		msgByte []byte
		err     error
	)

	if msgByte, err = json.Marshal(msg); err != nil {
		return err
	}

	return saveMsgClient.publishCh.Publish(
		"x_save_msg", // XChange
		"",           // saveMsgQName, // Key
		false,        // Mandatory
		false,        // Immediate
		amqp.Publishing{ // Msg
			DeliveryMode: amqp.Persistent, // Messages that are q-ed to be saved to DB, should be persistent
			ContentType:  "application/json",
			Body:         msgByte,
		},
	)
}

func closeSaveMsg() {
	saveMsgClient.closeReconn <- true
	saveMsgClient.isPubReady = false

	saveMsgClient.publishCh.Close()
}
