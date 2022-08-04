package db

import (
	"errors"
	"mumble-message-service/utils"
)

func UpdateMessageStatus(msgId int64, status string) error {
	utils.Log.Println(msgId, status)
	commandTag, err := db.Exec(dbContext, "UPDATE message SET status = $1 WHERE message_id = $2", status, msgId)
	if err == nil && commandTag.RowsAffected() != 1 {
		return errors.New("Error updating message status at DB")
	}
	return err
}
