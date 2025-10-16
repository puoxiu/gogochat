package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/puoxiu/gogochat/pkg/zlog"
)

// Register 上送服务地址
// 参数:
//   - serviceName: 服务名称：例如 "user_service"
//   - addr: 服务实例地址：例如 "127.0.0.1:9001"
func Register(serviceName, addr string) error {
	client := GetEtcdClient()
	if client == nil {
		return fmt.Errorf("etcd client not initialized")
	}

	key := fmt.Sprintf("/services/%s/%s", serviceName, addr)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.Put(ctx, key, addr)
	if err != nil {
		return fmt.Errorf("failed to register service: %v", err)
	}

	zlog.Info(fmt.Sprintf("registered service: %s -> %s", serviceName, addr))
	return nil
}
