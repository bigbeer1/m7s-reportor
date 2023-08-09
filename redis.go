package reportor

import "github.com/go-redis/redis/v8"

func (p ReportorConfig) NewRedisClusterManager() *redis.ClusterClient {
	// 解析redis-config

	var addrs []string
	var passwd string
	addrs = append(addrs, p.Host)
	passwd = p.Pass

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,
		Password: passwd,
	})

	return rdb

}

func (p ReportorConfig) NewRedisManager() *redis.Client {
	// 解析redis-config
	var addrs string
	var passwd string
	addrs = p.Host
	passwd = p.Pass

	var rdb = redis.NewClient(&redis.Options{
		Addr:     addrs,
		Password: passwd,
		PoolSize: 100,
	})
	return rdb

}
