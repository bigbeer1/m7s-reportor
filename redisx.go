package reportor

import "github.com/go-redis/redis/v8"

func (p ReportorConfig) NewRedisClusterManager() *redis.ClusterClient {
	// 解析redis-config

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    p.RedisHost,
		Password: p.RedisPass,
	})

	return rdb

}

func (p ReportorConfig) NewRedisManager() *redis.Client {
	// 解析redis-config

	if len(p.RedisHost) > 1 {
		var rdb = redis.NewClient(&redis.Options{
			Addr:     p.RedisHost[0],
			Password: p.RedisPass,
			PoolSize: 100,
		})
		return rdb
	}

	return nil

}
