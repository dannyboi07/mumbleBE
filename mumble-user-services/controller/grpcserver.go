package controller

import usp "mumble-user-service/user_service_proto"

type UserGrpcServer struct {
	usp.UnimplementedUserServiceServer
}
