package rbmq

import (
	"encoding/json"
	"errors"
	"mumble-gateway-service/types"
	"mumble-gateway-service/utils"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Channel for publishing last seen updates
// var pubLSCh *amqp.Channel

// Channel for consuming last seen updates
// var conLSCh *amqp.Channel

type lastSeen struct {
	publishCh        *amqp.Channel
	consumeCh        *amqp.Channel
	isPubReady       bool
	isConReady       bool
	notifyPubChClose chan *amqp.Error
	notifyConChClose chan *amqp.Error
	closeReconn      chan bool
}

var lastSeenClient lastSeen

func initLastSeen() error {
	// if !client.isReady {
	// 	return errors.New("RabbitMq client not ready")
	// }
	// publishCh: nil,
	// consumeCh: nil,
	// isPubReady: false,
	// isConReady: false,
	// notifyPubChClose: make(chan *amqp.Error),
	// notifyConChClose: make(chan *amqp.Error),

	lastSeenClient = lastSeen{}

	err := initLastSeenPub()
	if err != nil {
		return err
	}

	err = initLastSeenCon()
	if err != nil {
		return err
	}

	err = initPubLastSeenX()
	if err != nil {
		utils.Log.Println("Failed to initialize exchange to publish last seen")
		return err
	}
	go reconnLastSeenCh()

	return nil
}

func initLastSeenPub() error {
	if !client.isPubReady {
		return errors.New("RabbitMq client publish connection not ready")
	}

	lastSeenClient.isPubReady = false

	var err error
	lastSeenClient.publishCh, err = client.publishConn.Channel()
	if err != nil {
		utils.Log.Println("err opening channel to publish last seen", err)
		return err
	}

	lastSeenClient.notifyPubChClose = make(chan *amqp.Error)
	lastSeenClient.publishCh.NotifyClose(lastSeenClient.notifyPubChClose)
	lastSeenClient.isPubReady = true

	return nil
}

func initLastSeenCon() error {
	if !client.isConReady {
		return errors.New("RabbitMq client consumption connection not ready")
	}

	lastSeenClient.isConReady = false

	var err error
	lastSeenClient.consumeCh, err = client.consumeConn.Channel()
	if err != nil {
		utils.Log.Println("err opening channel to consume last seen", err)
		return err
	}

	lastSeenClient.notifyConChClose = make(chan *amqp.Error)
	lastSeenClient.consumeCh.NotifyClose(lastSeenClient.notifyConChClose)
	lastSeenClient.isConReady = true

	return nil
}

func reconnLastSeenCh() {
	for {
		time.Sleep(5 * time.Second)

		select {
		case <-lastSeenClient.notifyPubChClose:
			// close(lastSeenClient.notifyPubChClose)
			utils.Log.Println("reopening ls pub channel")
			err := initLastSeenPub()
			if err != nil {
				utils.Log.Println("Failed to reinitialize channel for last seen publishes", err)
			}

		case <-lastSeenClient.notifyConChClose:
			// close(lastSeenClient.notifyConChClose)
			utils.Log.Println("opening ls sub channel")
			err := initLastSeenCon()
			if err != nil {
				utils.Log.Println("Failed to reinitialize channel for last seen consumption", err)
			}
		case <-lastSeenClient.closeReconn:
			return
		}
	}
}

func initPubLastSeenX() error {

	if !lastSeenClient.isPubReady {
		return errors.New("last seen publish channel not ready yet")
	}
	var err error

	if err = lastSeenClient.publishCh.ExchangeDeclare(
		"x_last_seen",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	return nil
}

func SubToLastSeenQ(subberUserId int64, subToUserId int64) (<-chan amqp.Delivery, error) {
	utils.Log.Println("chan status", lastSeenClient.consumeCh.IsClosed())
	if !lastSeenClient.isConReady {
		return nil, errors.New("Last seen consumption channel not ready yet")
	}

	qAndConsumerName := strconv.FormatInt(subberUserId, 10)
	var err error

	// Dirty q disconnect, noWait is true, low priority msgs, ignoring them if any in-flight
	// err = lastSeenClient.consumeCh.Cancel(qAndConsumerName, true)
	// if err != nil {
	// 	utils.Log.Println("err cancelling last seen q consumption, err:", err)
	// }

	// Using subscriber's userId to ensure a new separate queue is created,
	// else a single queue would be created for n + 1 users interested in a particular
	// user's last seen update and only 1 from n users will receive the update msg
	if _, err = lastSeenClient.consumeCh.QueueDeclare(
		qAndConsumerName,
		false,
		true,
		true,
		true, // Using noWait true, arbitrary q
		nil,
	); err != nil {
		return nil, err
	}

	if err = lastSeenClient.consumeCh.QueueBind(
		qAndConsumerName,
		strconv.FormatInt(subToUserId, 10),
		"x_last_seen",
		true, // Using noWait true, arbitrary queue
		nil,
	); err != nil {
		return nil, err
	}

	var lastSeenChan <-chan amqp.Delivery
	lastSeenChan, err = lastSeenClient.consumeCh.Consume(
		qAndConsumerName, // qAndConsumerName
		qAndConsumerName, // consumer name
		true,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return lastSeenChan, nil
}

func PubLastSeen(subToUserId int64, lastSeen types.UserLastSeen) error {
	if !lastSeenClient.isPubReady {
		return errors.New("last seen publish channel not ready yet")
	}

	var (
		msgByte []byte
		err     error
	)

	if msgByte, err = json.Marshal(lastSeen); err != nil {
		return err
	}

	return lastSeenClient.publishCh.Publish(
		"x_last_seen",
		strconv.FormatInt(subToUserId, 10),
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msgByte,
		},
	)
}

func CancelSubToLastSeen(subberUserId int64) error {
	// _, err := conLSCh.QueueDelete(strconv.FormatInt(subberUserId, 10), false, false, true)
	// return err
	if !lastSeenClient.isConReady {
		return errors.New("last seen consumption channel not ready yet")
	}
	// utils.Log.Println("chan status", lastSeenClient.consumeCh.IsClosed())
	return lastSeenClient.consumeCh.Cancel(strconv.FormatInt(subberUserId, 10), false)
}

func closeLastSeen() {
	lastSeenClient.closeReconn <- true
	lastSeenClient.isPubReady = false
	lastSeenClient.isConReady = false

	lastSeenClient.publishCh.Close()
	lastSeenClient.consumeCh.Close()
}

// func SubToLastSeenQ(subberUserId int64, subToUserId int64) (<-chan amqp.Delivery, error) {

// 	qAndConsumerName := strconv.FormatInt(subberUserId, 10)
// 	var err error

// 	// Dirty q disconnect, noWait is true, low priority msgs, ignoring them if any in-flight
// 	err = conLSCh.Cancel(qAndConsumerName, true)
// 	if err != nil {
// 		utils.Log.Println("err cancelling last seen q consumption, err:", err)
// 	}

// 	// Using subscriber's userId to ensure a new separate queue is created,
// 	// else a single queue would be created for n + 1 users interested in a particular
// 	// user's last seen update and only 1 from n users will receive the update msg
// 	if _, err = conLSCh.QueueDeclare(
// 		qAndConsumerName,
// 		false,
// 		true,
// 		true,
// 		true, // Using noWait true, arbitrary q
// 		nil,
// 	); err != nil {
// 		return nil, err
// 	}

// 	if err = conLSCh.QueueBind(
// 		qAndConsumerName,
// 		strconv.FormatInt(subToUserId, 10),
// 		"x_last_seen",
// 		true, // Using noWait true, arbitrary queue
// 		nil,
// 	); err != nil {
// 		return nil, err
// 	}

// 	var lastSeenChan <-chan amqp.Delivery
// 	lastSeenChan, err = conLSCh.Consume(
// 		qAndConsumerName, // qAndConsumerName
// 		qAndConsumerName, // consumer name
// 		true,
// 		true,
// 		false,
// 		false,
// 		nil,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return lastSeenChan, nil
// }
