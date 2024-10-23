package main

import "github.com/xstarnet/go-infra/etcd"

func main() {
	etcd.Hello()
	etcd.Test([]string{"127.0.0.1:2379"}, "greet.rpc")
}
