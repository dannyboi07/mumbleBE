package rbmq

import (
	"mumble-gateway-service/utils"
	"time"

	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

type clientType struct {
	publishConn *amqp.Connection
	consumeConn *amqp.Connection

	notifyPubConnClose chan *amqp.Error
	notifyConConnClose chan *amqp.Error

	// saveMsg
	// sendMsg
	// lastSeen
	// pubSaveMsgCh *amqp.Channel
	// conSendMsgCh *amqp.Channel
	// pubLSCh *amqp.Channel
	// conLSCh *amqp.Channel

	// done chan bool
	// isReady    bool
	isPubReady bool
	isConReady bool
}

var client clientType

// var publishConn *amqp.Connection
// var consumeConn *amqp.Connection

// Have to declare an exchange which routes messages that have to be saved to the message service
// Also have to create a listener which is listening on a queue meant for a particular instance, and when a message arrives
// which has to be delivered to a websocket connection (It is guaranteed in most cases that the user is on that particular
// instance, if not, then hand over to the queue on which the User Service is listening, it will deliver it through Service Workers)
// func InitMq() error {
// 	var err error
// 	if publishConn, err = amqp.Dial(os.Getenv("MQ_ADDR")); err != nil {
// 		utils.Log.Println("err opening rbmq connection for publishes")
// 		return err
// 	}
// 	if consumeConn, err = amqp.Dial(os.Getenv("MQ_ADDR")); err != nil {
// 		utils.Log.Println("err opening rbmq connection for consumption")
// 		return err
// 	}

// 	// Open publishing channels
// 	// if pubSaveMsgCh, err = publishConn.Channel(); err != nil {
// 	// 	utils.Log.Println("err opening channel for publishing msgs to be saved")
// 	// 	return err
// 	// }
// 	// if pubLSCh, err = publishConn.Channel(); err != nil {
// 	// 	utils.Log.Println("err opening channel for publishing user's last seen")
// 	// 	return err
// 	// }
// 	err = initLastSeen()
// 	if err != nil {
// 		utils.Log.Println("err initializing rbmq for last seens")
// 		return err
// 	}

// 	// Open consumption channels
// 	if conSendMsgCh, err = consumeConn.Channel(); err != nil {
// 		utils.Log.Println("err opening channel for consuming messages to be sent")
// 		return err
// 	}
// 	if conLSCh, err = consumeConn.Channel(); err != nil {
// 		utils.Log.Println("err opening channel for consuming last seen updates")
// 		return err
// 	}
// 	// err = initSendMsg()
// 	// if err != nil {
// 	// 	utils.Log.Println("err initializing rbmq for sending messages")
// 	// 	return err
// 	// }

// 	// err = initSaveMsg()
// 	// if err != nil {
// 	// 	utils.Log.Println("err initializing rbmq for saving messages")
// 	// 	return err
// 	// }

// 	// if err = initSaveMsgXAndQ(); err != nil {
// 	// 	utils.Log.Println("err initializing exchange & queues for saving msgs")
// 	// 	return err
// 	// }
// 	// if err = initSendMsgXAndQ(); err != nil {
// 	// 	utils.Log.Println("err initing exchange & queues for sending msgs")
// 	// 	return err
// 	// }
// 	// if err = initPubLastSeenX(); err != nil {
// 	// 	utils.Log.Println("err initing exchange for publishing last seen")
// 	// 	return err
// 	// }

// 	// go func() {
// 	// 	for {
// 	// 		time.Sleep(5 * time.Second)
// 	// 		var uuid int64 = 0
// 	// 		typeMess := "message"
// 	// 		var from int64 = 11
// 	// 		var to int64 = 12
// 	// 		text := "Testing RabbitMQ"
// 	// 		utils.Log.Println("Publishing message...")
// 	// 		err := PublishMsg(types.WsMsg{
// 	// 			MsgUUID: &uuid,
// 	// 			Type:    &typeMess,
// 	// 			From:    &from,
// 	// 			To:      &to,
// 	// 			Text:    &text,
// 	// 		})
// 	// 		if err != nil {
// 	// 			utils.Log.Println("Err publishing message err:", err)
// 	// 			return
// 	// 		}
// 	// 	}
// 	// }()

// 	return nil
// }

// func CloseMq() {
// 	publishConn.Close()
// 	consumeConn.Close()
// }

func InitMq() error {

	client = clientType{
		// publishConn: nil,
		// consumeConn: nil,
		// notifyPubConnClose: make(chan *amqp.Error),
		// notifyConConnClose: make(chan *amqp.Error),
		// done: make(chan bool),
		// isReady:            false,
		// isPubReady: false,
		// isConReady: false,
	}

	var err error
	// client.publishConn, err = amqp.Dial(os.Getenv("MQ_ADDR"))
	// if err != nil {
	// 	utils.Log.Println("err opening rbmq connection for publishes")
	// 	return err
	// }

	utils.Log.Println("Opening rabbitmq connection for publishes")
	err = initPubConn()
	if err != nil {
		return err
	}

	utils.Log.Println("Opening rabbitmq connection for consumption")
	err = initConConn()
	if err != nil {
		return err
	}

	utils.Log.Println("Spinning func for connection recovery")
	go reconnConns()

	utils.Log.Println("Initing last seen")
	err = initLastSeen()
	if err != nil {
		utils.Log.Println("err initializing rbmq for last seens")
		return err
	}

	utils.Log.Println("Initing save msg")
	err = initSaveMsg()
	if err != nil {
		utils.Log.Println("err initializing rbmq for saving messages")
		return err
	}

	utils.Log.Println("Initing send msg")
	err = initSendMsg()
	if err != nil {
		utils.Log.Println("err initializing rbmq for sending messages")
		return err
	}
	// utils.Log.Println("RabbitMq inited")

	return nil
}

func initPubConn() error {
	var err error
	client.isPubReady = false

	client.publishConn, err = amqp.Dial(os.Getenv("MQ_ADDR"))
	if err != nil {
		utils.Log.Println("err opening rbmq connection for publishes")
		return err
	}

	client.notifyPubConnClose = make(chan *amqp.Error)
	client.publishConn.NotifyClose(client.notifyPubConnClose)
	client.isPubReady = true

	return nil
}

func initConConn() error {
	var err error
	client.isConReady = false

	client.consumeConn, err = amqp.Dial(os.Getenv("MQ_ADDR"))
	if err != nil {
		utils.Log.Println("err opening rbmq connection for consumption")
		return err
	}

	client.notifyConConnClose = make(chan *amqp.Error)
	client.consumeConn.NotifyClose(client.notifyConConnClose)
	client.isConReady = true

	return nil
}

func reconnConns() {
	for {
		time.Sleep(3 * time.Second)

		select {
		case <-client.notifyPubConnClose:
			// close(client.notifyPubConnClose)
			err := initPubConn()
			if err != nil {
				utils.Log.Println("Failed to reinitialize connection for publishes", err)
			} else {
				err = initLastSeenCon()
				if err != nil {
					utils.Log.Println("Failed to reinitialize last seen consumption", err)
				}

				err = initSendMsgCon(true)
				if err != nil {
					utils.Log.Println("Failed to reinitialize send msg con", err)
				}
			}

		case <-client.notifyConConnClose:
			// close(client.notifyConConnClose)
			err := initConConn()
			if err != nil {
				utils.Log.Println("Failed to reinitialize connection for consumption", err)
			} else {
				err = initLastSeenPub()
				if err != nil {
					utils.Log.Println("Failed to reinitialize last seen publish", err)
				}

				err = initSaveMsgPub()
				if err != nil {
					utils.Log.Println("Failed to reinitialize save msg publish", err)
				}
			}
		}
	}
}

func CloseMq() error {
	// if !client.isReady {
	// 	return errors.New("rbmq client already closed")
	// }
	// close(client.done)

	if client.isPubReady {
		client.isPubReady = false
		err := client.publishConn.Close()
		if err != nil {
			return err
		}
	}

	if client.isConReady {
		client.isConReady = false
		err := client.consumeConn.Close()
		if err != nil {
			return err
		}
	}

	closeLastSeen()
	closeSaveMsg()
	closeSendMsg()

	// client.isReady = false
	return nil
}

// if err != nil && errors.Is(err, amqp.ErrClosed) {
// 	if retry < 3 {
// 		if err = InitMsgMq(); err != nil {
// 			utils.Log.Println("Err retrying to connect to RabbitMQ, err:", err)
// 		} else {
// 			utils.Log.Println("Retry successful")
// 			retry = 0
// 		}
// 	} else {
// 		utils.Log.Fatalln("Exhausted retries to connect to RabbitMQ, exiting..., err:", err)
// 	}
//
// return nil
// return publishCh.Publish(
// 	"x_save_msgs",
// 	saveMsgQPub.Name,
// 	false,
// 	false,
// 	amqp.Publishing{
// 		ContentType: "application/json",
// 		Body:        msgByte,
// 	},
// )
