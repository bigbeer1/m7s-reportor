package reportor

import (
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/denisbrodbeck/machineid"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
	. "m7s.live/engine/v4"
	. "m7s.live/plugin/gb28181/v4"
	"time"
)

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

type VideoChannel struct {
	StreamPath       string `json:"stream_path"`        // 流通道地址
	MonibucaId       string `json:"monibuca_id"`        // 服务器ID
	MonibucaIp       string `json:"monibuca_ip"`        // 服务器IP
	StreamState      int64  `json:"stream_state"`       // 流状态
	StreamCreateTime int64  `json:"stream_create_time"` // 流拉取时间
	StreamType       string `json:"stream_type"`        // 流格式
}

var reportorPlugin = InstallPlugin(new(ReportorConfig))

func (p *ReportorConfig) OnEvent(event any) {

	switch v := event.(type) {
	case FirstConfig:
		id, _ := machineid.ProtectedID("monibuca")
		if id == "" {
			id = uuid.NewString()
		}
		p.MonibucaId = id
		fmt.Println(v)
		// 创建redis 连接 判断是集群还是 单体

		if len(p.RedisHost) > 0 {
			switch p.RedisType {
			case "node":
				//  单体redis
				p.Redis = p.NewRedisManager()

			case "cluster":
				// 集群redis
				p.RedisCluster = p.NewRedisClusterManager()

			}
		}

		if len(p.EtcdHost) > 0 {
			// etcd客户端
			p.Etcd = p.NewEtcdManager()

		}

		// 同步服务器状态
		go p.SyncServiceWorker()
		go p.SyncWorker()
	}
}

// 开启同步任务
func (p *ReportorConfig) SyncWorker() {
	// GB28181设备信息
	for {
		time.Sleep(time.Second * time.Duration(p.SyncTime))
		p.SyncGBDevices()
		p.SyncVideoChannels()
	}

}

func (p *ReportorConfig) SyncServiceWorker() {
	for {
		// GB28181设备信息
		p.SyncService()
		time.Sleep(time.Second * time.Duration(p.SyncServiceTime))
	}

}

// 同步GB设备表
func (p *ReportorConfig) SyncGBDevices() {
	Devices.Range(func(key, value interface{}) bool {
		publicKey := fmt.Sprintf("gbDevices:%v", key)
		privateKey := fmt.Sprintf("m7s:%v:gbDevices:%v", p.MonibucaId, key)
		// 反序列化
		data, err := sonic.Marshal(value)
		if err != nil {
			reportorPlugin.Error(fmt.Sprintf("gbDevices设备数据反序列化失败:%s", err.Error()))
			return true
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if p.Redis != nil {
			cmd := p.Redis.Set(ctx, publicKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
				return true
			}
			cmd = p.Redis.Set(ctx, privateKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
				return true
			}
		}

		if p.RedisCluster != nil {
			cmd := p.RedisCluster.Set(ctx, publicKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
				return true
			}
			cmd = p.RedisCluster.Set(ctx, privateKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
				return true
			}
		}

		if p.Etcd != nil {
			leaseResp, err := p.Etcd.Grant(ctx, p.SyncSaveTime)
			if err != nil {
				reportorPlugin.Error(fmt.Sprintf("etcd创建lease失败:%s", err.Error()))
				return true
			}
			// 写入键值对
			_, err = p.Etcd.Put(ctx, publicKey, string(data), clientv3.WithLease(leaseResp.ID))
			if err != nil {
				reportorPlugin.Error(fmt.Sprintf("etcd存储失败:%s", err.Error()))
				return true
			}

			// 写入键值对
			_, err = p.Etcd.Put(ctx, privateKey, string(data), clientv3.WithLease(leaseResp.ID))
			if err != nil {
				reportorPlugin.Error(fmt.Sprintf("etcd存储失败:%s", err.Error()))
				return true
			}
		}

		return true
	})
}

// 同步m7s服务端信息
func (p *ReportorConfig) SyncService() {
	key := fmt.Sprintf("m7sService:%v", p.MonibucaId)
	data, err := sonic.Marshal(SysInfo)
	if err != nil {
		reportorPlugin.Error(fmt.Sprintf("m7sService设备数据反序列化失败:%s", err.Error()))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if p.Redis != nil {
		cmd := p.Redis.Set(ctx, key, data, time.Second*time.Duration(p.SyncSaveTime))
		if cmd.Err() != nil {
			reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
			return
		}
	}

	if p.RedisCluster != nil {
		cmd := p.RedisCluster.Set(ctx, key, data, time.Second*time.Duration(p.SyncSaveTime))
		if cmd.Err() != nil {
			reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
			return
		}
	}

	if p.Etcd != nil {
		leaseResp, err := p.Etcd.Grant(ctx, p.SyncSaveTime)
		if err != nil {
			reportorPlugin.Error(fmt.Sprintf("etcd创建lease失败:%s", err.Error()))
			return
		}
		// 写入键值对
		_, err = p.Etcd.Put(ctx, key, string(data), clientv3.WithLease(leaseResp.ID))
		if err != nil {
			reportorPlugin.Error(fmt.Sprintf("etcd存储失败:%s", err.Error()))
			return
		}
	}

}

// 同步流通道
func (p *ReportorConfig) SyncVideoChannels() {

	Streams.Range(func(streamPath string, stream *Stream) {
		publicKey := fmt.Sprintf("streamPath:%v", streamPath)
		privateKey := fmt.Sprintf("m7s:%v:streamPath:%v", p.MonibucaId, streamPath)

		videoChannel := &VideoChannel{
			StreamPath:       streamPath,
			MonibucaId:       p.MonibucaId,
			MonibucaIp:       SysInfo.LocalIP,
			StreamState:      int64(stream.State),
			StreamCreateTime: stream.StartTime.UnixMilli(),
			StreamType:       stream.GetType(),
		}
		// 反序列化
		data, err := sonic.Marshal(videoChannel)
		if err != nil {
			reportorPlugin.Error(fmt.Sprintf("SyncVideoChannel设备数据反序列化失败:%s", err.Error()))
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if p.Redis != nil {
			cmd := p.Redis.Set(ctx, publicKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
				return
			}

			cmd = p.Redis.Set(ctx, privateKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
				return
			}
		}

		if p.RedisCluster != nil {
			cmd := p.RedisCluster.Set(ctx, publicKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
				return
			}

			cmd = p.RedisCluster.Set(ctx, privateKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
				return
			}

		}

		if p.Etcd != nil {
			leaseResp, err := p.Etcd.Grant(ctx, p.SyncSaveTime)
			if err != nil {
				reportorPlugin.Error(fmt.Sprintf("etcd创建lease失败:%s", err.Error()))
				return
			}
			// 写入键值对
			_, err = p.Etcd.Put(ctx, publicKey, string(data), clientv3.WithLease(leaseResp.ID))
			if err != nil {
				reportorPlugin.Error(fmt.Sprintf("etcd存储失败:%s", err.Error()))
				return
			}

			// 写入键值对
			_, err = p.Etcd.Put(ctx, privateKey, string(data), clientv3.WithLease(leaseResp.ID))
			if err != nil {
				reportorPlugin.Error(fmt.Sprintf("etcd存储失败:%s", err.Error()))
				return
			}
		}

		return
	})

}
