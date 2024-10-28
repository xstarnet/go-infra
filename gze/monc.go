package gze

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
)

type (
	MongoConf struct {
		Host string
		DB   string
	}
)

func LoadModel[T interface{}](modelCreator func(url, db, collection string, c cache.CacheConf) T, c StoreConf, coll string) T {
	return modelCreator(c.Mongo.Host, c.Mongo.DB, coll, c.Redis)
}
