package controller

import (
	msp "mumble-message-service/message-service-proto"
)

type MsgGrpcServer struct {
	msp.UnimplementedMessageServiceServer
}
