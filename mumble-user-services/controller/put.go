package controller

import (
	"context"
	"errors"
	"mumble-user-service/db"
	"mumble-user-service/types"
	usp "mumble-user-service/user_service_proto"
	"mumble-user-service/utils"

	"github.com/jackc/pgx/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *UserGrpcServer) ChangePwd(ctx context.Context, user *usp.ChangePwdReq) (*emptypb.Empty, error) {
	var (
		userDetails types.User
		err         error
	)
	userDetails, err = db.SelectUserById(user.UserId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "User does not exist")
		}

		utils.Log.Printf("err getting user from db, err: %v", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	if !utils.AuthPassword(userDetails.PasswordHash, user.OldPassword) {
		return nil, status.Errorf(codes.Unauthenticated, "Wrong password")

	} else if errString, ok := utils.ValidPassword(user.NewPassword); !ok {
		return nil, status.Errorf(codes.InvalidArgument, errString)
	}

	newPwdHash, err := utils.HashPassword(user.NewPassword, 10)
	if err != nil {
		utils.Log.Printf("err: hashing user's pw, err: %v", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	if err = db.UpdateUserPwd(user.UserId, newPwdHash); err != nil {
		utils.Log.Printf("err: updating user pwd in db, err: %v", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	return nil, nil
}

func (s *UserGrpcServer) ChangeDp(ctx context.Context, user *usp.ChangeDpReq) (*emptypb.Empty, error) {
	if err := db.UpdateUserDp(user.UserId, user.ProfilePic); err != nil {
		utils.Log.Printf("err updating user's dp in db, err: %v", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.Unauthenticated, "User doesn't exist")
		}

		return nil, status.Errorf(codes.Internal, "Internal server error")
	}
	return nil, nil
}
