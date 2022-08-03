package redis

import (
	"mumble-user-service/utils"
	"strconv"
	"time"
)

func SetUserOnline(userId int64) error {
	if err := redisConn.Set(ctx, strconv.FormatInt(userId, 10), "Online", 0).Err(); err != nil {
		return err
	}
 
	// if err := redisConn.Publish(ctx, fmt.Sprintf("userstatus %d", userId), "Online").Err(); err != nil {
	//	utils.Log.Printf("redis: Error publishing user as online, err: %v", err)
	// }
	return nil
}

func SetUserOffline(userId int64, time string) error {
	if err := redisConn.Set(ctx, strconv.FormatInt(userId, 10) + ":lst-sn", time, 0).Err(); err != nil {
		return err
	}

	// if err := redisConn.Publish(ctx, fmt.Sprintf("userstatus %d", userId), time).Err(); err != nil {
	// 	utils.Log.Printf("redis: Error publishing user's last seen, err: %v", err)
	// 	return err
	// }

	return nil
}

func SetRefToken(key, refreshToken string, expTime time.Duration) error {
	if err := redisConn.Set(ctx, key, refreshToken, expTime).Err(); err != nil {
		utils.Log.Printf("redis: Error setting refresh token, key: %s, err: %v", key, err, redisConn)
		return err
	}

	return nil
}
