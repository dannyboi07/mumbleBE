package controller

import (
	"context"
	"errors"
	"mumble-user-service/db"
	"mumble-user-service/redis"
	"mumble-user-service/types"
	usp "mumble-user-service/user_service_proto"
	"mumble-user-service/utils"
	"strconv"
	"time"

	"github.com/jackc/pgx/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *UserGrpcServer) RegisterUser(ctx context.Context, user *usp.RegisterReq) (*emptypb.Empty, error) {

	// Validate user's name
	if err := utils.ValidName(user.Name); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// Validate user's email
	if !utils.ValidEmail(user.Email) {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid email format")
	}

	// Check if user exists
	if exists, err := db.UserExistsByEmail(user.Email); err != nil {

		utils.Log.Printf("err checking if user exists by email, err: %v", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	} else if exists {
		return nil, status.Errorf(codes.AlreadyExists, "User already exists")
	}

	// Validate password
	if errString, ok := utils.ValidPassword(user.Password); !ok {
		return nil, status.Errorf(codes.InvalidArgument, errString)
	}

	hashedPw, err := utils.HashPassword(user.Password, 10)
	if err != nil {
		utils.Log.Printf("err: hashing user's pw, err: %v", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	if err := db.InsertUser(user, hashedPw); err != nil {
		utils.Log.Printf("err: inserting user into db, err: %v", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	return nil, nil
}

func (s *UserGrpcServer) LoginUser(ctx context.Context, user *usp.LoginReq) (*usp.LoginResp, error) {
	var (
		userDetails types.User
		err         error
	)

	userDetails, err = db.SelectUserByEmail(user.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "User doesn't exist")
		}

		utils.Log.Printf("err while retrieving user's details, err: %v", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	if !utils.AuthPassword(userDetails.PasswordHash, user.Password) {
		return nil, status.Errorf(codes.Unauthenticated, "Wrong password")
	}

	var (
		accTk            string
		accTkExp         int
		userTokenDetails = types.UserForToken{UserId: userDetails.UserId, Email: userDetails.Email}
	)
	accTk, accTkExp, err = utils.CreateJwt(userTokenDetails)
	if err != nil {
		utils.Log.Printf("err creating acc token, err: %v", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	var (
		refTk    string
		refTkExp time.Duration
	)
	refTk, refTkExp, err = utils.CreateRefreshJwt(userTokenDetails)
	if err != nil {
		utils.Log.Printf("err creating ref token, err: %v", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	err = redis.SetRefToken(strconv.FormatInt(userDetails.UserId, 10)+":refTk", refTk, refTkExp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	return &usp.LoginResp{
		UserId:          userDetails.UserId,
		Name:            userDetails.Name,
		Email:           userDetails.Email,
		ProfilePic:      userDetails.Profile_pic,
		AccessToken:     accTk,
		RefreshToken:    refTk,
		AccessTokenExp:  int64(accTkExp),
		RefreshTokenExp: int64(refTkExp.Seconds())}, nil
}

func (s *UserGrpcServer) AddContact(ctx context.Context, contactIds *usp.AddContactReq) (*emptypb.Empty, error) {
	err := db.InsertContact(contactIds.ContactId_1, contactIds.ContactId_2)
	if err != nil {
		if err.Error() == "Contact already exists" {
			return nil, status.Errorf(codes.AlreadyExists, err.Error())
		}

		utils.Log.Printf("err inserting new contact into db, err: %v", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	return nil, nil
}
