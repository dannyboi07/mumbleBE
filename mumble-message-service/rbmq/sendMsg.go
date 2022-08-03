package rbmq

import amqp "github.com/rabbitmq/amqp091-go"

var publishMsgToSendCh *amqp.Channel

func initMsgToSendX() error {
	return publishMsgToSendCh.ExchangeDeclare(
		"x_send_msg",
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
}
