package controller

import (
	"context"
	"mumble-message-service/db"
	msp "mumble-message-service/message-service-proto"
	"mumble-message-service/utils"
)

func (s *MsgGrpcServer) GetMsgs(ctx context.Context, q *msp.MessageQuery) (*msp.Messages, error) {
	var (
		contact_id_1, contact_id_2, offset int64 = q.ContactId_1, q.ContactId_2, q.Offset
		messages                           *msp.Messages
		err                                error
	)

	messages, err = db.SelectMsgs(contact_id_1, contact_id_2, offset)
	if err != nil {
		utils.Log.Println("Error getting messages, err:", err)
		return nil, err
	}

	return messages, nil
}
