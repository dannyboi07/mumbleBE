package controller

import (
	"context"
	"errors"
	"io/ioutil"
	"mumble-user-service/db"
	"mumble-user-service/redis"
	"mumble-user-service/types"
	usp "mumble-user-service/user_service_proto"
	"mumble-user-service/utils"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *UserGrpcServer) GetContacts(ctx context.Context, userId *usp.GetContactsReq) (*usp.GetContactsResp, error) {

	contacts, err := db.SelectContacts(userId.UserId)
	if err != nil {
		utils.Log.Printf("err getting user's contacts, userid: %d, err: %v", userId.UserId, err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	return contacts, nil
}

func (s *UserGrpcServer) SearchUser(ctx context.Context, query *usp.SearchUserReq) (*usp.SearchUserResp, error) {
	var (
		user types.UserSearch
		err  error
	)
	user, err = db.SelectUserByEmailSearch(query.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "User doesn't exist")
		}

		utils.Log.Printf("err searching for user in db, email: %s, err: %v", query.Email, err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	isContact, err := db.ContactExists(query.UserId, user.UserId)
	if err != nil {
		utils.Log.Printf("err getting contact exists from db, err: %v", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	return &usp.SearchUserResp{
		UserId:     user.UserId,
		Name:       user.Name,
		ProfilePic: user.Profile_pic,
		IsFriend:   isContact,
	}, nil
}

func (s *UserGrpcServer) LogoutUser(ctx context.Context, userId *usp.LogOutReq) (*emptypb.Empty, error) {
	err := redis.DelRefToken(strconv.FormatInt(userId.UserId, 10) + ":refTk")
	if err != nil {
		utils.Log.Printf("err deleting refTk from redis, userId: %d, err: %v", userId.UserId, err)
	}

	return new(emptypb.Empty), nil
}

func (s *UserGrpcServer) RefreshAccToken(ctx context.Context, refTkReq *usp.RefreshAccTokenReq) (*usp.RefreshAccTokenResp, error) {

	// utils.Log.Println("received req to ref token")
	var (
		claims    jwt.MapClaims
		statusInt codes.Code
		err       error
	)
	claims, statusInt, err = utils.VerifyJwt(refTkReq.RefreshToken)
	if err != nil {
		return nil, status.Error(statusInt, err.Error())
	}

	var (
		exists bool
	)
	exists, err = redis.RefTokenExists(strconv.FormatInt(refTkReq.UserId, 10)+":refTk", refTkReq.RefreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}
	if !exists {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthorized refresh token")
	}

	var (
		accTk    string
		accTkExp int
		userId   int64
		email    string
		ok       bool
	)
	if userIdFloat, uIdOk := claims["UserId"].(float64); !uIdOk {
		return nil, status.Error(codes.PermissionDenied, "Invalid token claims")
	} else {
		userId = int64(userIdFloat)
	}
	if email, ok = claims["Email"].(string); !ok {
		return nil, status.Error(codes.PermissionDenied, "Invalid token claims")
	}

	userForToken := types.UserForToken{
		UserId: userId,
		Email:  email,
	}

	accTk, accTkExp, err = utils.CreateJwt(userForToken)
	if err != nil {
		utils.Log.Printf("err creating new accTk, userId: %d, err: %v", userId, err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	var (
		refTk    string
		refTkExp time.Duration
	)
	refTk, refTkExp, err = utils.CreateRefreshJwt(userForToken)
	if err != nil {
		utils.Log.Printf("err creating new refTk, userId: %d, err: %v", userId, err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	err = redis.SetRefToken(strconv.FormatInt(userId, 10)+":refTk", refTk, refTkExp)
	if err != nil {
		utils.Log.Printf("err setting refTk in redis, userId: %d, err: %v", userId, err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	// utils.Log.Println("response:", accTk, refTk, int64(accTkExp))

	return &usp.RefreshAccTokenResp{
		AccessToken:     accTk,
		RefreshToken:    refTk,
		AccessTokenExp:  int64(accTkExp),
		RefreshTokenExp: int64(refTkExp.Seconds()),
	}, nil
}

func (s *UserGrpcServer) GetRSAKeys(ctx context.Context, null *emptypb.Empty) (*usp.GetRSAKeysResp, error) {
	privKeyBytes, err := ioutil.ReadFile("private.pem")
	if err != nil {
		utils.Log.Printf("Failed to read private pem file, err: %v", err)
		return nil, status.Error(codes.Internal, "Failed to read private key file")
	}

	pubKeyBytes, err := ioutil.ReadFile("public.pem")
	if err != nil {
		utils.Log.Printf("Failed to read public pem file, err: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to read public key file")
	}

	return &usp.GetRSAKeysResp{
		PrivateKey: privKeyBytes,
		PublicKey:  pubKeyBytes,
	}, nil
}
