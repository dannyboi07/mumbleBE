package db

import "mumble-message-service/types"

func InsertMessage(insertMessage types.WsMsg) (types.WsMsg, error) {
	var message types.WsMsg
	row := db.QueryRow(dbContext, "INSERT INTO message (msg_from, msg_to, text, status) VALUES ($1, $2, $3, $4) RETURNING *", insertMessage.From, insertMessage.To, insertMessage.Text, "saved")
	err := row.Scan(&message.MsgId, &message.From, &message.To, &message.Text, &message.Time, &message.Status)
	if err != nil {
		return message, err
	}
	return message, nil
}
