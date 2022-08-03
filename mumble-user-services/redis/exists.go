package redis

import (
	"errors"
	"mumble-user-service/utils"
	"strconv"

	"github.com/go-redis/redis/v9"
)

func IsUserOnline(userId int64) bool {
	// Change check to val == "Online" ? true : false after checking for code which depends on this
	return redisConn.Exists(ctx, strconv.FormatInt(userId, 10) + ":lst-sn").Val() == 0
}

func RefTokenExists(key, refTk string) (bool, error) {
	getVal := redisConn.Get(ctx, key)

	if getVal.Err() != nil {
		if errors.Is(getVal.Err(), redis.Nil) {
			return false, nil
		}

		utils.Log.Printf("err getting refTk from redis, key: %s, err: %v", key, getVal.Err())
		return false, getVal.Err()
	}

	return getVal.Val() == refTk, nil
}
