package global

import "github.com/hashicorp/consul/api"

var (
	Nacos          map[string]interface{}
	NacosConfig    map[string]interface{}
	MysqlConf      *MysqlConfig
	RedisConf      *RedisConfig
	logFilePath    string
	Env            string
	ConsulClient   *api.Client
	UserServerConf *UserServerConfig
)
