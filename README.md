# plugin-reportor
数据同步上报插件——使得m7s实例可作把内存中信息同步到其他数据库中去：目前支持reids,redis-clusters



## 插件地址
https://github.com/bigbeer1/m7s-reportor

## 插件引入

```go
import (
   _ "github.com/bigbeer1/m7s-reportor"
)
```

## 配置

```yaml

reportor:
  enable: true
  redishost:
    - "redis-1:6381"
  redistype: "cluster"
  redispass: "G62m5301234567"

  etcdhost:
    - "127.0.0.1:30000"
  etcdusername: ""
  etcdpassword: ""
  etcddialtimeout: 10

  syncservicetime: 10
  synctime: 10
  syncsavetime: 180

```

origin代表源服务器拉流地址前缀，可以由如下几种格式：
```
type ReportorConfig struct {
    MonibucaId string   // m7sId 唯一标识
    RedisHost  []string // redis地址
    RedisType  string   `default:",default=node,options=node|cluster"` // redis类型
    RedisPass  string   // redis密码

    EtcdHost        []string // etcd地址
    EtcdUsername    string   // etcd用户名
    EtcdPassword    string   // etcdPassword
    EtcdDialTimeout int64    `default:"10"` // 通讯超时时间  秒

    SyncServiceTime int64 `default:"30"`  // 同步服务器信息在线状态时间
    SyncTime        int64 `default:"30"`  // 同步阻塞时间
    SyncSaveTime    int64 `default:"180"` // 同步数据有效期时间

    RedisCluster *redis.ClusterClient // redisCluster客户端
    Redis        *redis.Client        // redis客户端

    Etcd *clientv3.Client // etcd客户端
}
```

## 使用

如果不存在redis  可以通过文件中 docker-composer-redis.yml  启动
如果不存在redis  可以通过文件中 docker-composer-redis-cluters.yml  启动

注 cluters 启动后需要使用命令将cluters创建
$ docker exec -it redis-1 redis-cli --cluster create 172.20.99.11:6381 172.20.99.12:6382 172.20.99.13:6383 172.20.99.14:6384 172.20.99.15:6385 172.20.99.16:6386 --cluster-replicas 1


redis默认密码为 G62m5301234567  请自行修改

### reids 单体配置示例
```
reportor:
  redishost: 
    - "127.0.0.1:6379"
  redistype: "node"
  redispass: "G62m5301234567"
```


### reids 集群配置示例
```
reportor:
  redishost: 
   - "redis-1:6379"
  redistype: "cluster"
  redispass: "G62m5301234567"
```
如果在docker 内 需要修改 本地hosts文件  

```
127.0.0.1 redis-1
127.0.0.1 redis-2
127.0.0.1 redis-3
127.0.0.1 redis-4
127.0.0.1 redis-5
127.0.0.1 redis-6
```


### etcd 集群配置示例
```
reportor:
  etcdhost:
    - "127.0.0.1:30000"
  etcdusername: ""
  etcdpassword: ""
  etcddialtimeout: 10
```