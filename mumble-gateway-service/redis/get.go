package redis

import (
	"errors"
	"mumble-gateway-service/utils"
	"strconv"

	redisPkg "github.com/go-redis/redis/v9"
)

func GetUserStatus(userId int64) (string, error) {
	lastSeen, err := redisConn.Get(ctx, strconv.FormatInt(userId, 10)+":lst-sn").Result()

	if err != nil {
		if errors.Is(err, redisPkg.Nil) {
			return "", nil
		}

		utils.Log.Printf("redis: Error getting user's status, err: %v", err)
		return "", err
	}
	return lastSeen, nil
}
