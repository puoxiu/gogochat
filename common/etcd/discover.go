package etcd

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// GetServiceAddr 获取某个服务的所有实例地址
func GetServiceAddr(serviceName string) ([]string, error) {
	client := GetEtcdClient()
	if client == nil {
		return nil, fmt.Errorf("etcd client not initialized")
	}

	prefix := fmt.Sprintf("/services/%s/", serviceName)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get service address: %v", err)
	}

	addrs := []string{}
	for _, kv := range resp.Kvs {
		addrs = append(addrs, string(kv.Value))
	}

	log.Printf("[etcd] discovered %s -> %v\n", serviceName, addrs)
	return addrs, nil
}
