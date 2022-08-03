package grpc_clients

import (
	msp "mumble-gateway-service/message_service_proto"
	usp "mumble-gateway-service/user_service_proto"
	"mumble-gateway-service/utils"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var msgConn *grpc.ClientConn
var msgClient msp.MessageServiceClient

var userConn *grpc.ClientConn
var userClient usp.UserServiceClient

func InitMsgStub() error {
	var (
		opts []grpc.DialOption
		err  error
	)
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	msgConn, err = grpc.Dial(os.Getenv("MSG_SRVC_ADDR"), opts...)
	if err == nil {
		msgClient = msp.NewMessageServiceClient(msgConn)
	}
	return err
}

func InitUserStub() error {
	var (
		opts []grpc.DialOption
		err  error
	)
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	userConn, err = grpc.Dial(os.Getenv("USR_SRVC_ADDR"), opts...)
	if err == nil {
		userClient = usp.NewUserServiceClient(userConn)
	}
	return err
}

func CloseMsgStub() {
	if err := msgConn.Close(); err != nil {
		utils.Log.Println("Error closing message gRPC connection, err:", err)
	}
}

func CloseUserStub() {
	if err := userConn.Close(); err != nil {
		utils.Log.Println("Error closing user gRPC connection, err:", err)
	}
}

// func ConnCheck() {
// 	if conn == nil {
// 		var (
// 			count int = 0
// 			err   error
// 		)
// 		for count < 3 {
// 			if err = InitGRPCStub(); err != nil {
// 				utils.Log.Println("Failed to reinitiate gRPC connection, try", count+1)
// 			}
// 		}
// 		if err != nil {
// 			utils.Log.Fatalln("Failed to resurrect gRPC connection, closing")
// 		}
// 	}
// 	return
// }
