package redis

func DelRefToken(key string) error {
	return redisConn.Del(ctx, key).Err()
}
