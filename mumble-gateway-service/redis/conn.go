package redis

import (
	"context"
	"os"
	"strconv"

	redisPkg "github.com/go-redis/redis/v9"
)

var redisConn *redisPkg.Client
var ctx context.Context

func InitRedis() error {
	ctx = context.Background()
	db, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		return err
	}

	redisConn = redisPkg.NewClient(&redisPkg.Options{
		Network:     "tcp",
		Addr:        os.Getenv("REDIS_ADDR"),
		Password:    os.Getenv("REDIS_PWD"),
		DB:          db,
		ReadTimeout: 0,
	})
	return nil
}

func CloseRedis() {
	redisConn.Close()
}
