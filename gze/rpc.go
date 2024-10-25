package gze

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/x/errors"
	mvccpb "go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

type (
	emptyGrpcClientConn struct{}
	emptyClient         struct{}
	zRpcClientConn      struct {
		zc zrpc.Client
	}
)

type (
	RpcClientConn interface {
		Conn() grpc.ClientConnInterface
	}
)

func (emptyGrpcClientConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	return errors.New(50404, "RpcServerNotFound")
}

func (emptyGrpcClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New(50404, "RpcServerNotFound")
}

func (c *emptyClient) Conn() grpc.ClientConnInterface {
	return emptyGrpcClientConn{}
}

func (c zRpcClientConn) Conn() grpc.ClientConnInterface {
	return c.zc.Conn()
}

type RpcStubCreator[T interface{}] func(cli RpcClientConn) T

type RpcServerKey struct {
	Key string
	Env string
	ID  string
}

func NewRpcServerKey(from *mvccpb.KeyValue) (RpcServerKey, bool) {
	out := strings.Split(string(from.Key), "/")
	if len(out) == 2 {
		key := strings.Split(out[0], "@")
		env := "default"
		if len(key) >= 2 {
			env = key[1]
		}
		return RpcServerKey{
			Key: key[0],
			Env: env,
			ID:  out[1],
		}, true
	}
	return RpcServerKey{}, false
}

type RpcClientRouter[T interface{}] struct {
	creator    RpcStubCreator[T]
	hosts      []string
	key        string
	envStubMap map[string]T
	etcdCli    *clientv3.Client
}

func NewRpcClientRouter[T interface{}](EtcdHosts []string, RpcKey string, creator func(cli RpcClientConn) T) *RpcClientRouter[T] {
	return &RpcClientRouter[T]{
		creator:    creator,
		envStubMap: make(map[string]T),
		key:        RpcKey,
		hosts:      EtcdHosts,
	}
}

func (r *RpcClientRouter[T]) GetEnvKey(env string) string {
	if env == "" {
		env = "default"
	}
	return fmt.Sprintf("%v@%v", r.key, env)
}

func (r *RpcClientRouter[T]) Stub(ctx context.Context) T {
	var env string
	meta, ok := ctx.Value(CtxMeta).(ContextMeta)
	if ok {
		env = meta.Env
	}

	if stub, ok := r.envStubMap[env]; ok {
		return stub
	}

	defaultStub, ok := r.envStubMap["default"]
	if !ok {
		return r.creator(&emptyClient{})
	}
	return defaultStub
}

func (r *RpcClientRouter[T]) AddRpcClient(ctx context.Context, KV *mvccpb.KeyValue) {
	rpcKey, ok := NewRpcServerKey(KV)
	if !ok {
		logc.Errorf(ctx, "Unknown Key: %v", string(KV.Key))
		return
	}
	if _, ok := r.envStubMap[rpcKey.Env]; !ok {
		client, err := zrpc.NewClient(zrpc.RpcClientConf{
			Etcd: discov.EtcdConf{
				Hosts: r.hosts,
				Key:   r.GetEnvKey(rpcKey.Env),
			},
		})
		if err != nil {
			logc.Errorf(ctx, "Get Key[%v] Error: %v", r.key, err)
			return
		}
		logc.Infof(ctx, "Add Client: %v", rpcKey.Env)
		r.envStubMap[rpcKey.Env] = r.creator(zRpcClientConn{zc: client})
	}
}

func (r *RpcClientRouter[T]) DeleteRpcClient(ctx context.Context, KV *mvccpb.KeyValue) {
	rpcKey, ok := NewRpcServerKey(KV)
	if !ok {
		logc.Errorf(ctx, "Unknown Key: %v", string(KV.Key))
		return
	}
	if _, ok := r.envStubMap[rpcKey.Env]; ok {
		logc.Infof(ctx, "Delete Client: %v", rpcKey.Env)
		delete(r.envStubMap, rpcKey.Env)
	}
}

func (r *RpcClientRouter[T]) Start(ctx context.Context, etcdCli *clientv3.Client) {
	r.etcdCli = etcdCli
	go r.sync(ctx)
	go r.watch(ctx)
}

func (r *RpcClientRouter[T]) watch(ctx context.Context) {
	watcherCtx := clientv3.WithRequireLeader(ctx)
	watcher := r.etcdCli.Watch(watcherCtx, r.key, clientv3.WithPrefix(), clientv3.WithCreatedNotify())

	for res := range watcher {
		if res.Canceled {
			logc.Errorf(watcherCtx, "Watcher[%v] canceled", r.key)
		}
		if res.Created {
			logc.Infof(watcherCtx, "Watcher[%v] created", r.key)
		}
		for _, event := range res.Events {
			if event.Type == clientv3.EventTypePut {
				r.AddRpcClient(watcherCtx, event.Kv)
			}
			if event.Type == clientv3.EventTypeDelete {
				r.DeleteRpcClient(watcherCtx, event.Kv)
			}
		}
	}
}

func (r *RpcClientRouter[T]) sync(ctx context.Context) {
	_sync := func() {
		val, err := r.etcdCli.Get(ctx, r.key, clientv3.WithPrefix())
		if err != nil {
			logc.Errorf(ctx, "Get Key[%v] Error: %v", r.key, err)
			return
		}
		for _, kv := range val.Kvs {
			r.AddRpcClient(ctx, kv)
		}
	}

	_sync()
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_sync()
		}
	}
}
