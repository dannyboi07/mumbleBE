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
	lastSeenClient = lastSeen{}

	utils.Log.Println("Initing publish channel for last seen")
	err := initLastSeenPub()
	if err != nil {
		return err
	}

	utils.Log.Println("Initing consumption channel for last seen")
	err = initLastSeenCon()
	if err != nil {
		return err
	}

	utils.Log.Println("Initing last seen topology")
	err = initPubLastSeenX()
	if err != nil {
		utils.Log.Println("Failed to initialize exchange to publish last seen")
		return err
	}

	utils.Log.Println("Go-oing consumption of last seens")
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

	utils.Log.Println("last seen pub re/opened")
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

	utils.Log.Println("last seen con re/opened")
	return nil
}

func reconnLastSeenCh() {
	for {
		time.Sleep(5 * time.Second)

		select {
		case <-lastSeenClient.notifyPubChClose:
			utils.Log.Println("Close listener: Last seen publishing channel closed")
			err := initLastSeenPub()
			if err != nil {
				utils.Log.Println("Failed to reinitialize channel for last seen publishes", err)
			}

		case <-lastSeenClient.notifyConChClose:
			utils.Log.Println("Close listener: Last seen consumption channel closed")
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
		utils.Log.Println("Err declaring exchange for last seens")
		return err
	}

	return nil
}

func SubToLastSeenQ(subberUserId int64, subToUserId int64) (<-chan amqp.Delivery, error) {

	utils.Log.Println("subbing to last seen", subberUserId, subToUserId)
	utils.Log.Println("chan status", lastSeenClient.consumeCh.IsClosed())
	if !lastSeenClient.isConReady {
		return nil, errors.New("Last seen consumption channel not ready yet")
	}

	qAndConsumerName := strconv.FormatInt(subberUserId, 10)
	var err error

	// Using subscriber's userId for making a new separate queue
	if _, err = lastSeenClient.consumeCh.QueueDeclare(
		qAndConsumerName,
		false,
		true,
		true,
		true, // Using noWait true, arbitrary q
		nil,
	); err != nil {
		utils.Log.Println("err declaring ls queue")
		return nil, err
	}

	// Use routing key of the userId to subscribe to, bind to topic exchange
	if err = lastSeenClient.consumeCh.QueueBind(
		qAndConsumerName,
		strconv.FormatInt(subToUserId, 10),
		"x_last_seen",
		true, // Using noWait true, arbitrary queue
		nil,
	); err != nil {
		utils.Log.Println("err binding ls queue")
		return nil, err
	}

	var lastSeenChan <-chan amqp.Delivery
	lastSeenChan, err = lastSeenClient.consumeCh.Consume(
		qAndConsumerName,
		qAndConsumerName,
		true,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		utils.Log.Println("err consuming ls queue")
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
	if !lastSeenClient.isConReady {
		return errors.New("last seen consumption channel not ready yet")
	}
	return lastSeenClient.consumeCh.Cancel(strconv.FormatInt(subberUserId, 10), true)
}

func closeLastSeen() {
	lastSeenClient.closeReconn <- true
	lastSeenClient.isPubReady = false
	lastSeenClient.isConReady = false

	lastSeenClient.publishCh.Close()
	lastSeenClient.consumeCh.Close()
}
