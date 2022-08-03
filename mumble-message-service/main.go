package main

import (
	"context"
	"mumble-message-service/db"
	msp "mumble-message-service/message-service-proto"
	"mumble-message-service/rbmq"
	"mumble-message-service/redis"
	"mumble-message-service/utils"
	"net"
	"os"

	"google.golang.org/grpc"
)

type msgGrpcServer struct {
	msp.UnimplementedMessageServiceServer
}

func main() {
	utils.InitLogger()

	var err error

	// err = godotenv.Load()
	// if err != nil {
	// 	utils.Log.Fatalf("Failed to load env vars, err: %v", err)
	// }
	// utils.Log.Println("Env vars loaded...")

	if err = db.InitDB(); err != nil {
		utils.Log.Fatalln("Failed to connect to database, exiting, err:", err)
	}
	defer db.CloseDB()
	utils.Log.Println("Connected to DB...")

	if err = rbmq.InitMq(); err != nil {
		utils.Log.Fatalln("Failed to connect to mq, exiting, err:", err)
	}
	defer rbmq.CloseMq()
	utils.Log.Println("Connected to RabbitMQ...")

	utils.Log.Println("Connecting to Redis...")
	if err = redis.InitRedis(); err != nil {
		utils.Log.Fatalln("Failed to connect to redis, err:", err)
	}
	defer redis.CloseRedis()
	utils.Log.Println("Connected to Redis")

	var lis net.Listener
	lis, err = net.Listen("tcp", os.Getenv("SRVR_ADDR"))
	if err != nil {
		utils.Log.Fatalln("Failed to listen, err:", err)
	}

	grpcServer := grpc.NewServer()
	msp.RegisterMessageServiceServer(grpcServer, &msgGrpcServer{})

	utils.Log.Println("Starting gRPC server")
	if err = grpcServer.Serve(lis); err != nil {
		utils.Log.Fatalln("Failed to serve gRPC server, err:", err)
	}
}

func (s *msgGrpcServer) GetMsgs(ctx context.Context, q *msp.MessageQuery) (*msp.Messages, error) {
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
