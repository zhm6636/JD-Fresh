package global

import (
	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"github.com/hashicorp/consul/api"

	"user_api/proto"
)

var (
	Router       *gin.Engine
	Nacos        map[string]interface{}
	NacosConf    map[string]interface{}
	MysqlConf    *MysqlConfig
	logFilePath  string
	UserClient   proto.UserClient
	Trans        ut.Translator
	Env          string
	ConsulClient *api.Client
	ServerId     string
)

const JWT = "123123"
