package etcd

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func Hello() {
	fmt.Println("hello world")
}

func Test(hosts []string, key string) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   hosts,
		DialTimeout: 5 * time.Second,
	})
	fmt.Println("cli created")
	if err != nil {
		fmt.Println("clientv3.New err: ", err)
	}
	val, err := cli.Get(context.TODO(), key, clientv3.WithPrefix())
	if err != nil {
		fmt.Println("cli.get err: ", err)
	}
	for _, KV := range val.Kvs {
		fmt.Printf("key: %s, value: %s\n", string(KV.Key), string(KV.Value))
	}
	defer cli.Close()
}
