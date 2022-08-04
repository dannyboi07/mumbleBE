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

func InitMq() error {

	client = clientType{}

	var err error

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

	utils.Log.Println("rabbitmq publishing connection re/opened")

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

	utils.Log.Println("rabbitmq consumption connection re/opened")

	return nil
}

func reconnConns() {
	for {
		time.Sleep(3 * time.Second)

		select {
		case <-client.notifyPubConnClose:
			utils.Log.Println("Close listener: Rabbitmq publish connection closed")
			err := initPubConn()
			if err != nil {
				utils.Log.Println("Failed to reinitialize connection for publishes", err)
			} else {
				err = initLastSeenPub()
				if err != nil {
					utils.Log.Println("Failed to reinitialize last seen consumption", err)
				}

				err = initSaveMsgPub()
				if err != nil {
					utils.Log.Println("Failed to reinitialize send msg con", err)
				}
			}

		case <-client.notifyConConnClose:
			utils.Log.Println("Close listener: Rabbitmq consumption connection closed")
			err := initConConn()
			if err != nil {
				utils.Log.Println("Failed to reinitialize connection for consumption", err)
			} else {
				err = initLastSeenCon()
				if err != nil {
					utils.Log.Println("Failed to reinitialize last seen publish", err)
				}

				err = initSendMsgCon(true)
				if err != nil {
					utils.Log.Println("Failed to reinitialize save msg publish", err)
				}
			}
		}
	}
}

func CloseMq() error {
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

	return nil
}
