package gze

import (
	"context"
	"net/http"
)

func RpcRouterMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		meta, ok := ctx.Value(CtxMeta).(ContextMeta)
		if !ok {
			meta = ContextMeta{}
		}
		meta.Env = r.Header.Get(CTX_META_ENV)
		ctx = context.WithValue(ctx, CtxMeta, meta)
		next(w, r.WithContext(ctx))
	}
}
