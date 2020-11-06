package guac

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"go.etcd.io/etcd/clientv3"
)

// var etcdClient *EtcdClient = nil

type EtcdClient struct {
	Cli              *clientv3.Client
	OperationTimeout time.Duration
}

func NewEtcdClient() *EtcdClient {
	timeout := 10 * time.Second
	etcdUser := os.Getenv("ETCDCTL_USER")
	etcdUserToken := strings.Split(etcdUser, ":")
	if len(etcdUserToken) < 2 {
		log.Fatal("no provide etcd user")
		return nil
	}
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{os.Getenv("ETCDCTL_ENDPOINT"), "http://localhost:2379"},
		Username:    etcdUserToken[0],
		Password:    etcdUserToken[1],
		DialTimeout: timeout,
	})
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return &EtcdClient{
		Cli:              client,
		OperationTimeout: 3 * time.Second,
	}
}

func etcdget(client *EtcdClient, key string, opts ...clientv3.OpOption) *clientv3.GetResponse {
	ctx, cancel := context.WithTimeout(context.Background(), client.OperationTimeout)
	getResp, err := client.Cli.Get(ctx, key, opts...)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	return getResp
}

func EtcdGet(client *EtcdClient, key string, opts ...clientv3.OpOption) *clientv3.GetResponse {
	return etcdget(client, key, opts...)
}

func EtcdGetWithPrefix(client *EtcdClient, key string) *clientv3.GetResponse {
	return etcdget(client, key, clientv3.WithPrefix())
}

func EtcdPut(client *EtcdClient, key string, value string) *clientv3.PutResponse {
	ctx, cancel := context.WithTimeout(context.Background(), client.OperationTimeout)
	putResp, err := client.Cli.Put(ctx, key, value)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	return putResp
}

func EtcdDel(client *EtcdClient, key string) *clientv3.DeleteResponse {
	ctx, cancel := context.WithTimeout(context.Background(), client.OperationTimeout)
	delResp, err := client.Cli.Delete(ctx, key, clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	return delResp
}
