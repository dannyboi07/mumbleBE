package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"mumble-user-service/controller"
	"mumble-user-service/db"
	"mumble-user-service/redis"
	usp "mumble-user-service/user_service_proto"
	"mumble-user-service/utils"
	"net"
	"os"

	"google.golang.org/grpc"
)

func main() {
	utils.InitLogger()

	if privKeyBytes, err := ioutil.ReadFile("private.pem"); err != nil {
		// Exit if the error is not a "File does not exist err"
		if !errors.Is(err, os.ErrNotExist) {
			utils.Log.Fatalf("Failed to open private pem key, err: %v", err)
		}

		utils.Log.Println("Creating priv and pub keys and pem files")

		var privateKey *rsa.PrivateKey
		privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			utils.Log.Fatalf("Failed to generate private key, err: %v", err)
		}

		utils.PrivKey = privateKey
		utils.PubKey = &privateKey.PublicKey

		utils.Log.Println("Marshalling public key")
		var pubKeyBytes []byte
		pubKeyBytes, err = x509.MarshalPKIXPublicKey(utils.PubKey)
		if err != nil {
			utils.Log.Fatalf("Failed to marshal public key, err: %v", err)
		}
		var publicKeyBlock *pem.Block = &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubKeyBytes,
		}

		// var wg sync.WaitGroup
		// wg.Add(2)

		// go func() {
		// 	defer wg.Done()

		utils.Log.Println("Creating public pem")
		var pubPemFile *os.File
		pubPemFile, err = os.Create("public.pem")
		if err != nil {
			utils.Log.Fatalf("Failed to create public pem file, err: %v", err)
		}

		if err = pem.Encode(pubPemFile, publicKeyBlock); err != nil {
			utils.Log.Fatalf("Failed to encode public key block to pem file, err: %v", err)
		}

		// }()

		// go func() {
		// 	defer wg.Done()

		utils.Log.Println("Marshalling private key")
		privKeyBytes = x509.MarshalPKCS1PrivateKey(privateKey)
		privateKeyBlock := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privKeyBytes,
		}
		utils.Log.Println("Creating private pem")
		var privPemFile *os.File
		privPemFile, err = os.Create("private.pem")
		if err != nil {
			utils.Log.Fatalf("Failed to create private pem file, err: %v", err)
		}

		if err := pem.Encode(privPemFile, privateKeyBlock); err != nil {
			utils.Log.Fatalf("Failed to encode private key block to pem file, err: %v", err)
		}
		// }()
		// wg.Wait()
	} else {
		// var wg sync.WaitGroup
		// wg.Add(2)

		// go func() {
		// 	defer wg.Done()

		var privKeyPem *pem.Block

		if privKeyPem, _ = pem.Decode(privKeyBytes); privKeyPem != nil {

			var privateKey *rsa.PrivateKey
			privateKey, err = x509.ParsePKCS1PrivateKey(privKeyPem.Bytes)
			if err != nil {
				utils.Log.Fatalf("Failed to parse private key, err: %v", err)
			}

			utils.PrivKey = privateKey
		} else {
			utils.Log.Fatalf("Failed to decode private key pem")
		}
		// }()

		// go func() {
		// 	defer wg.Done()

		var pubKeyBytes []byte
		pubKeyBytes, err = ioutil.ReadFile("public.pem")
		if err != nil {
			utils.Log.Fatalf("Failed to read public pem file, err: %v", err)
		}

		var pubKeyPem *pem.Block
		if pubKeyPem, _ = pem.Decode(pubKeyBytes); err == nil {
			// publicKey is type any
			publicKey, err := x509.ParsePKIXPublicKey(pubKeyPem.Bytes)
			if err != nil {
				utils.Log.Fatalf("Failed to parse public key, err: %v", err)
			}

			var ok bool
			if utils.PubKey, ok = publicKey.(*rsa.PublicKey); !ok {
				utils.Log.Fatalf("Failed to type assert public key to rsa public key, err: %v", err)
			}
		} else {
			utils.Log.Fatalf("Failed to decode public key, err: %v", err)
		}
		// }()

		// wg.Wait()
	}

	utils.Log.Println("Connecting to db...")
	// Connect to db instance
	if err := db.InitDb(); err != nil {
		utils.Log.Printf("Failed to connect to db, err: %v", err)
		return
	}
	utils.Log.Println("Connected to db")
	defer db.CloseDb()

	utils.Log.Println("Connecting to redis...")
	// Connect to redis
	if err := redis.InitRedis(); err != nil {
		utils.Log.Printf("Failed to connect to redis, err: %v", err)
		return
	}
	defer redis.CloseRedis()

	// Get TCP listener
	lis, err := net.Listen("tcp", os.Getenv("SRVR_ADDR"))
	if err != nil {
		utils.Log.Printf("Failed to listen on %s, err: %v", os.Getenv("SRVR_ADDR"), err)
		return
	}

	utils.Log.Println("Starting gRPC server at", os.Getenv("SRVR_ADDR"))
	// Begin gRPC server on TCP listener with options
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	usp.RegisterUserServiceServer(grpcServer, &controller.UserGrpcServer{})
	if err := grpcServer.Serve(lis); err != nil {
		utils.Log.Printf("Failed to start gRPC server, err: %v", err)
		return
	}
}
