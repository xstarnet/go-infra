package gze

type key string

const (
	CtxMeta key = "CTX_META"
)
const CtxEtcdClient = "CTX_ETCD_CLIENT"
const (
	CTX_META_ENV = "x-ms-env"
)

type ContextMeta struct {
	Env string
}
