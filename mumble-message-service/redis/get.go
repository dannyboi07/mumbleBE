package redis

import (
	"errors"
	"strconv"

	"github.com/go-redis/redis/v9"
)

func GetUserHost(userId int64) (bool, string, error) {
	result, err := redisConn.Get(ctx, strconv.FormatInt(userId, 10)+":hstnm").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, "", nil
		}

		return false, "", err
	}

	return true, result, nil
}
