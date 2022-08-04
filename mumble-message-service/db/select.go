package db

import (
	msp "mumble-message-service/message-service-proto"
	"time"

	protoTStamp "google.golang.org/protobuf/types/known/timestamppb"
)

func SelectMsgs(contactId1, contactId2, offset int64) (*msp.Messages, error) {
	var messages msp.Messages = msp.Messages{}
	rows, err := db.Query(dbContext, `SELECT * FROM message WHERE
										msg_from = $1 AND msg_to = $2
										OR msg_from = $2 AND msg_to = $1
										ORDER BY time DESC OFFSET $3 LIMIT 20`, contactId1, contactId2, offset)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var (
			message  msp.Message
			tempTime time.Time
		)

		err := rows.Scan(&message.MsgId, &message.From, &message.To, &message.Text, &tempTime, &message.Status)
		if err != nil {
			return nil, err
		}

		message.Time = protoTStamp.New(tempTime)
		messages.Messages = append(messages.Messages, &message)
	}

	return &messages, nil
}
