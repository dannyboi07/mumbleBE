package grpc_clients

import (
	"context"
	msp "mumble-gateway-service/message_service_proto"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
)

func GetMsgsMethod(contactId1, contactId2, offset int64) ([]byte, error) {
	// ConnCheck()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	messageQueryPb := msp.MessageQuery{ContactId_1: contactId1, ContactId_2: contactId2, Offset: offset}
	messagesPb, err := msgClient.GetMsgs(ctx, &messageQueryPb)
	if err != nil {
		return nil, err
	}

	return protojson.MarshalOptions{UseProtoNames: true}.Marshal(messagesPb)
}
