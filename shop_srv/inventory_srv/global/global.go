package global

import (
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/hashicorp/consul/api"
	"github.com/olivere/elastic/v7"
)

var (
	Nacos            map[string]interface{}
	NacosConfig      map[string]interface{}
	MysqlConf        *MysqlConfig
	RedisConf        *RedisConfig
	logFilePath      string
	Env              string
	ConsulClient     *api.Client
	RocketMqProducer rocketmq.Producer
	UserServerConf   *UserServerConfig
	esClient         *elastic.Client
)
