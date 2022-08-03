package redis

import (
	"mumble-gateway-service/utils"
	"strconv"
	"time"
)

func SetUserOnline(userId int64) error {
	userIdStr := strconv.FormatInt(userId, 10)
	expiry := 24 * time.Hour
	if err := redisConn.Set(ctx, userIdStr+":lst-sn", "Online", expiry).Err(); err != nil {
		utils.Log.Println("Failed to set user as offline in redis, err:", err)
		// return err
	}

	err := redisConn.Set(ctx, userIdStr+":hstnm", utils.Hostname, expiry).Err()
	if err != nil {
		utils.Log.Println("Failed to set user's host machine key, err:", err)
	}

	return err
}

func SetUserOffline(userId int64, time string) error {
	if err := redisConn.Set(ctx, strconv.FormatInt(userId, 10)+":lst-sn", time, 0).Err(); err != nil {
		utils.Log.Println("Failed to user as offline, err:", err)
		// return err
	}

	err := redisConn.Del(ctx, strconv.FormatInt(userId, 10)+":hstnm").Err()
	if err != nil {
		utils.Log.Println("Failed to delete user's host machine key, err:", err)
	}

	return err
}

func SetRefToken(key, refreshToken string, expTime time.Duration) error {

	err := redisConn.Set(ctx, key, refreshToken, expTime).Err()
	if err != nil {
		utils.Log.Printf("redis: Error setting refresh token, key: %s, err: %v", key, err)
	}

	return err
}
