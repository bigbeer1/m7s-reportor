package reportor

import (
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/denisbrodbeck/machineid"
	"github.com/go-redis/redis/v8"
	. "m7s.live/engine/v4"
	. "m7s.live/plugin/gb28181/v4"
	"time"
)

type ReportorConfig struct {
	MonibucaId      string // m7sId 唯一标识
	Host            string // redis地址
	Type            string `default:",default=node,options=node|cluster"` // redis类型
	Pass            string // redis密码
	SyncServiceTime int64  `default:"30"`  // 同步服务器信息在线状态时间
	SyncTime        int64  `default:"30"`  // 同步阻塞时间
	SyncSaveTime    int64  `default:"180"` // 同步数据有效期时间
	RedisCluster    *redis.ClusterClient
	Redis           *redis.Client
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
		p.MonibucaId = id
		fmt.Println(v)
		// 创建redis 连接 判断是集群还是 单体

		switch p.Type {
		case "node":
			//  单体redis
			rdb := p.NewRedisManager()
			p.Redis = rdb

		case "cluster":
			// 集群redis
			rdb := p.NewRedisClusterManager()
			p.RedisCluster = rdb
		default:
			reportorPlugin.Error(fmt.Sprintf("不支持redis类型:%s,请以node|cluster", p.Type))
			return
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
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)

		if p.Type == "node" {
			cmd := p.Redis.Set(ctx, publicKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
			}
			cmd = p.Redis.Set(ctx, privateKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
			}
		}

		if p.Type == "cluster" {
			cmd := p.RedisCluster.Set(ctx, publicKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
			}
			cmd = p.RedisCluster.Set(ctx, privateKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
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
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)

	if p.Type == "node" {
		cmd := p.Redis.Set(ctx, key, data, time.Second*time.Duration(p.SyncSaveTime))
		if cmd.Err() != nil {
			reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
		}
	}

	if p.Type == "cluster" {
		cmd := p.RedisCluster.Set(ctx, key, data, time.Second*time.Duration(p.SyncSaveTime))
		if cmd.Err() != nil {
			reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
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
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)

		if p.Type == "node" {
			cmd := p.Redis.Set(ctx, publicKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
			}

			cmd = p.Redis.Set(ctx, privateKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
			}
		}

		if p.Type == "cluster" {
			cmd := p.RedisCluster.Set(ctx, publicKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
			}

			cmd = p.RedisCluster.Set(ctx, privateKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				reportorPlugin.Error(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
			}

		}

		return
	})

}
