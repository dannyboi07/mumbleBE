package grpc_clients

import (
	"context"
	"errors"
	"mumble-gateway-service/types"
	usp "mumble-gateway-service/user_service_proto"
	"mumble-gateway-service/utils"
	"time"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func RegisterUserMethod(userDetails types.RegisterUser) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := userClient.RegisterUser(ctx, &usp.RegisterReq{
		Email:      *userDetails.Email,
		Name:       *userDetails.Name,
		Password:   *userDetails.Password,
		ProfilePic: *userDetails.Profile_Pic,
	})

	if err != nil {
		grpcErr, _ := status.FromError(err)
		return utils.MapGrpcErrors(grpcErr.Code()), errors.New(grpcErr.Message())
	}

	return 0, nil
}

func LoginUserMethod(userDetails types.LoginUser) (*usp.LoginResp, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	loginResp, err := userClient.LoginUser(ctx, &usp.LoginReq{
		Email:    *userDetails.Email,
		Password: *userDetails.Password,
	})
	if err != nil {
		grpcErr, _ := status.FromError(err)
		return nil, utils.MapGrpcErrors(grpcErr.Code()), errors.New(grpcErr.Message())
	}

	return loginResp, 0, nil
}

func GetContactsMethod(userId int64) (*usp.GetContactsResp, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	contacts, err := userClient.GetContacts(ctx, &usp.GetContactsReq{UserId: userId})
	if err != nil {
		grpcErr, _ := status.FromError(err)
		return nil, utils.MapGrpcErrors(grpcErr.Code()), errors.New(grpcErr.Message())
	}

	return contacts, 0, nil
}

func LogoutMethod(userId int64) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := userClient.LogoutUser(ctx, &usp.LogOutReq{UserId: userId})
	if err != nil {
		grpcErr, _ := status.FromError(err)
		return utils.MapGrpcErrors(grpcErr.Code()), errors.New(grpcErr.Message())
	}

	return 0, nil
}

func ChangePwdMethod(userInput *usp.ChangePwdReq) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := userClient.ChangePwd(ctx, userInput)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		return utils.MapGrpcErrors(grpcErr.Code()), errors.New(grpcErr.Message())
	}

	return 0, nil
}

func ChangeDpMethod(userId int64, profilePic string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := userClient.ChangeDp(ctx, &usp.ChangeDpReq{
		UserId:     userId,
		ProfilePic: profilePic,
	})
	if err != nil {
		grpcErr, _ := status.FromError(err)
		return utils.MapGrpcErrors(grpcErr.Code()), errors.New(grpcErr.Message())
	}

	return 0, nil
}

func SearchUserMethod(userId int64, email string) (*usp.SearchUserResp, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userResult, err := userClient.SearchUser(ctx, &usp.SearchUserReq{
		UserId: userId,
		Email:  email,
	})
	if err != nil {
		grpcErr, _ := status.FromError(err)
		return nil, utils.MapGrpcErrors(grpcErr.Code()), errors.New(grpcErr.Message())
	}

	return userResult, 0, nil
}

func AddContactMethod(contactId1, contactId2 int64) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := userClient.AddContact(ctx, &usp.AddContactReq{
		ContactId_1: contactId1,
		ContactId_2: contactId2,
	})
	if err != nil {
		grpcErr, _ := status.FromError(err)
		return utils.MapGrpcErrors(grpcErr.Code()), errors.New(grpcErr.Message())
	}

	return 0, nil
}

func RefreshAccTokenMethod(userId int64, refTk string) (*usp.RefreshAccTokenResp, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	accAndRefTk, err := userClient.RefreshAccToken(ctx, &usp.RefreshAccTokenReq{
		RefreshToken: refTk,
		UserId:       userId,
	})

	if err != nil {
		grpcErr, _ := status.FromError(err)
		return nil, utils.MapGrpcErrors(grpcErr.Code()), errors.New(grpcErr.Message())
	}

	return accAndRefTk, 0, nil
}

func GetRSAKeys() ([]byte, []byte, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rsaKeysQuery, err := userClient.GetRSAKeys(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, nil, err
	}

	return rsaKeysQuery.PrivateKey, rsaKeysQuery.PublicKey, nil
}
