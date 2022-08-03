package redis

import (
	"mumble-user-service/utils"
	"strconv"
)

func GetUserStatus(userId int64) (string, error) {
	lastSeen, err := redisConn.Get(ctx, strconv.FormatInt(userId, 10)+":lst-sn").Result()
	if err != nil {
		utils.Log.Printf("redis: Error getting user's status, err: %v", err)
		return "", err
	}
	return lastSeen, err
}
