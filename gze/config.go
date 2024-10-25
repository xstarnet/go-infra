package gze

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type (
	StoreConf struct {
		Redis []redis.RedisConf
		Mongo MongoConf
	}
)
