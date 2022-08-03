package redis

import (
	"context"
	"mumble-user-service/utils"
	"os"
	"strconv"
	"time"

	redis "github.com/go-redis/redis/v9"
)

var redisConn *redis.Client
var ctx context.Context

func InitRedis() error {
	ctx = context.Background()
	db, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		return err
	}

	redisConn = redis.NewClient(&redis.Options{
		Addr:        os.Getenv("REDIS_ADDR"),
		Password:    os.Getenv("REDIS_PWD"),
		DB:          db,
		ReadTimeout: time.Second * 5, // Using default of 3 secs
	})
	utils.Log.Println("Connected to redis")
	return nil
}

func CloseRedis() {
	redisConn.Close()
}
