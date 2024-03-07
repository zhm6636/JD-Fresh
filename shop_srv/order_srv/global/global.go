package global

import (
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/hashicorp/consul/api"

	goodssrv "order_srv/proto/goods_srv"
	inventorysrv "order_srv/proto/inventory_srv"
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
	//esClient       *elastic.Client
	RocketMqProducer rocketmq.Producer
	GoodsClient      goodssrv.GoodsClient
	InventoryClient  inventorysrv.InventoryClient
)
