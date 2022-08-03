package rbmq

import (
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

var consumeConn *amqp.Connection
var publishConn *amqp.Connection

func InitMq() error {
	var err error
	// Open connection for consumption
	if consumeConn, err = amqp.Dial(os.Getenv("MQ_ADDR")); err != nil {
		return err
	}
	// Open connection for publishes
	if publishConn, err = amqp.Dial(os.Getenv("MQ_ADDR")); err != nil {
		return err
	}

	// Open channel to consume msgs to be saved to db
	if consumeMsgToSaveCh, err = consumeConn.Channel(); err != nil {
		return err
	}
	// Open channel to publish msgs to be sent back
	if publishMsgToSendCh, err = publishConn.Channel(); err != nil {
		return err
	}

	// Declare exchange to publish msgs to be sent
	if err = initMsgToSendX(); err != nil {
		return err
	}

	// Declare exchange and queue to consume msgs to be saved to db
	if err = initSaveMsgXandQ(); err != nil {
		return err
	}

	return nil
}

func CloseMq() {
	consumeConn.Close()
	publishConn.Close()
}
