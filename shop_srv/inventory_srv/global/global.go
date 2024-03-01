package global

import (
	"github.com/hashicorp/consul/api"
	"github.com/olivere/elastic/v7"
)

var (
	Nacos          map[string]interface{}
	NacosConfig    map[string]interface{}
	MysqlConf      *MysqlConfig
	RedisConf      *RedisConfig
	logFilePath    string
	Env            string
	ConsulClient   *api.Client
	UserServerConf *UserServerConfig
	esClient       *elastic.Client
)
