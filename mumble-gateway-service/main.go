package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"mumble-gateway-service/controller"
	grpcClients "mumble-gateway-service/grpc_clients"
	"mumble-gateway-service/rbmq"
	"mumble-gateway-service/redis"
	"mumble-gateway-service/s3Media"
	"mumble-gateway-service/types"
	ws "mumble-gateway-service/ws"
	"mumble-gateway-service/wsclients"

	"mumble-gateway-service/utils"

	"github.com/gorilla/websocket"
)

func main() {
	utils.InitLogger()
	utils.Hostname, _ = os.Hostname()

	// Connect to gRPC message service
	utils.Log.Println("gRPC stub connecting to message service...")
	if err := grpcClients.InitMsgStub(); err != nil {
		utils.Log.Fatalln("Error starting message service's gRPC client, err:", err)
	}
	defer grpcClients.CloseMsgStub()
	utils.Log.Println("Connected to gRPC message service")

	// Connect to gRPC user service
	utils.Log.Println("gRPC stub connecting to user service...")
	if err := grpcClients.InitUserStub(); err != nil {
		utils.Log.Fatalln("Error starting user service's gRPC client, err:", err)
	}
	defer grpcClients.CloseUserStub()
	utils.Log.Println("Connected to gRPC user service")

	// Connect and setup RabbitMQ
	utils.Log.Println("Connecting to RabbitMQ...")
	if err := rbmq.InitMq(); err != nil {
		utils.Log.Fatalln("Error connecting and initializing RabbitMQ, err:", err)
	}
	defer rbmq.CloseMq()
	utils.Log.Println("Connected to RabbitMQ")

	rand.Seed(time.Now().Unix())
	s3Media.InitS3()

	utils.Log.Println("Getting RSA keys from user service...", os.Getenv("USR_SRVC_ADDR"), os.Getenv("MSG_SRVC_ADDR"))
	privateKeyBytes, publicKeyBytes, err := grpcClients.GetRSAKeys()
	if err != nil {
		utils.Log.Fatalln("Failed to get RSA keys from user service, err:", err)
	}

	utils.Log.Println("Decoding private and public keys...")
	privateKeyPem, _ := pem.Decode(privateKeyBytes)
	publicKeyPem, _ := pem.Decode(publicKeyBytes)

	utils.Log.Println("Parsing private key...")
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyPem.Bytes)
	if err != nil {
		utils.Log.Fatalln("Error parsing private key PEM")
	}

	utils.Log.Println("Parsing public key...")
	publicKey, err := x509.ParsePKIXPublicKey(publicKeyPem.Bytes)
	if err != nil {
		utils.Log.Fatalln("Error parsing public key PEM")
	}

	utils.PrivKey = privateKey
	utils.PubKey = publicKey.(*rsa.PublicKey)

	utils.Log.Println("Connecting to Redis...")
	if err := redis.InitRedis(); err != nil {
		utils.Log.Fatalln("Error connecting to Redis, err:", err)
	}
	defer redis.CloseRedis()
	utils.Log.Println("Connected to Redis")

	wsclients.WsClients = &types.WsClients{ClientConns: make(map[int64]*websocket.Conn), RWMutex: sync.RWMutex{}}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc:  AllowOriginFunc,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}))
	r.Get("/media/{objFolder}/{objName}", controller.GetFromS3)
	r.Route("/api", func(r chi.Router) {

		r.Group(func(r chi.Router) {
			r.Use(utils.AuthMiddleware)

			r.Get("/messages/{id}", controller.GetMsgs)
			r.Get("/contacts", controller.GetContacts)
			r.Get("/ws", ws.Handler)
			r.Get("/searchUser", controller.SearchUser)

			r.Post("/addContact", controller.AddContact)

			r.Put("/changeDp", controller.ChangeDP)
			r.Put("/changePw", controller.ChangePassword)
		})

		r.Group(func(r chi.Router) {
			r.Post("/register", controller.RegisterUser)
			r.Post("/login", controller.LoginUser)

			r.Get("/auth/logout", controller.LogoutUser)
			r.Get("/auth/refresh_token", controller.RefreshAccToken)
		})
	})

	utils.Log.Println("Starting gateway on localhost:8080...")
	if err := http.ListenAndServe("0.0.0.0:8080", r); err != nil {
		log.Fatal("Error starting server: ", err.Error())
	}
}

func AllowOriginFunc(r *http.Request, origin string) bool {
	//if origin == "http://localhost:3000" || origin == "http://127.0.0.1:3000" {
	//	return true
	//}
	return true
}
