package gze

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
)

type (
	StoreConf struct {
		Redis []cache.NodeConf
		Mongo MongoConf
	}
)
