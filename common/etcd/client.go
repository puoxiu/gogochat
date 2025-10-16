
package etcd

import clientv3 "go.etcd.io/etcd/client/v3"

var etcdClient *clientv3.Client

// etcd客户端
func InitEtcd(addr string) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{addr},
	})
	if err != nil {
		panic(err)
	}

	etcdClient = cli
}

func GetEtcdClient() *clientv3.Client {
	return etcdClient
}

func CloseEtcdClient() {
	if etcdClient != nil {
		etcdClient.Close()
	}
}