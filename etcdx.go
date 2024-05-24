package reportor

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func (p *ReportorConfig) NewEtcdManager() *clientv3.Client {

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   p.EtcdHost,
		DialTimeout: time.Duration(p.EtcdDialTimeout) * time.Second,
		Username:    p.EtcdUsername,
		Password:    p.EtcdPassword,
	})
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = cli.Get(ctx, "test_key") // 尝试获取一个键值
	if err != nil {
		// 处理获取键值时的错误
		reportorPlugin.Error(fmt.Sprintf("etcd 连接失败:%s", err.Error()))
		return nil
	}

	return cli
}
