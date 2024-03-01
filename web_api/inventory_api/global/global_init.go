package global

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"

	"inventory_api/proto"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 初始化服务依赖
func init() {
	InitEnv()
	InitZap()
	InitViper()
	InitNaCos()
	//InitServer()

}

// nocos更新配置文件后，重新初始化服务依赖
func InitServer() {
	//InitRpc()
	//InitTrans()
	//InitConsul()
}

// 查看当前环境
func InitEnv() {
	Env = os.Getenv("ENV")
	if Env == "" {
		Env = "dev"
	}
	fmt.Println("Env: ", Env)
}

// 初始化nacos
func InitNaCos() {
	// 设置nacos服务地址
	var sc = []constant.ServerConfig{{
		IpAddr: NacosConf["nacos"].(map[string]interface{})["addr"].(string),
		Port:   uint64(NacosConf["nacos"].(map[string]interface{})["port"].(int)),
	}}
	// 设置nacos客户端配置
	var cc = constant.ClientConfig{
		NamespaceId:         NacosConf["nacos"].(map[string]interface{})["namespaceid"].(string),
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "tmp/logs",
		CacheDir:            "tmp",
		LogLevel:            "debug",
	}

	// 创建nacos客户端
	client, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": sc,
		"clientConfig":  cc,
	})
	if err != nil {
		zap.S().Panic(err)
	}
	// 从nacos获取配置文件
	data := NacosConf["nacos"].(map[string]interface{})["dataid"].(string)
	content, err := client.GetConfig(vo.ConfigParam{
		DataId: data,
		Group:  Env,
	})

	if err != nil {
		zap.S().Panic(err)
	}

	// 解析配置文件
	err = yaml.Unmarshal([]byte(content), &Nacos)
	if err != nil {
		zap.S().Panic(err)
	}

	// 监听nacos配置文件变化
	err = client.ListenConfig(vo.ConfigParam{
		DataId: data,
		Group:  Env,
		OnChange: func(namespace, group, dataId, data string) {
			//配置文件变化后，重新解析配置文件
			err = yaml.Unmarshal([]byte(data), &Nacos)
			InitServer()
			DeregisterService()
			InitConsul()
		},
	})

	if err != nil {
		zap.S().Panic(err)
	}

}

// 初始化mysql
func InitMysql() {
	var err error
	MysqlConf = &MysqlConfig{
		Addr:     Nacos["mysql"].(map[string]interface{})["addr"].(string),
		Port:     Nacos["mysql"].(map[string]interface{})["port"].(int),
		User:     Nacos["mysql"].(map[string]interface{})["username"].(string),
		Password: Nacos["mysql"].(map[string]interface{})["password"].(string),
		DataBase: Nacos["mysql"].(map[string]interface{})["database"].(string),
	}

	MysqlConf.Dsn = MysqlConf.User + ":" + MysqlConf.Password + "@tcp(" + MysqlConf.Addr + ":" + strconv.Itoa(MysqlConf.Port) + ")/" + MysqlConf.DataBase + "?charset=utf8mb4&parseTime=True&loc=Local"

	// 初始化gorm logger
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,          // Don't include params in the SQL log
			Colorful:                  true,          // Disable color
		},
	)
	// 初始化gorm DB
	MysqlConf.DB, err = gorm.Open(mysql.Open(MysqlConf.Dsn), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		zap.S().Panic(err)
	}

	// 设置gorm连接池
	sqlDB, err := MysqlConf.DB.DB()
	if err != nil {
		zap.S().Panic(err)
	}

	// 设置最大空闲连接数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

}

func InitLog() {
	logFilePath = Nacos["logFilePath"].(string)
	// 获取日志文件保存路径的目录部分
	logDir := filepath.Dir(logFilePath)

	// 创建目录（包括不存在的父目录）
	err := os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		zap.S().Panic("无法创建日志文件保存路径：", err)
	}

	// 打开日志文件，如果不存在则创建
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		zap.S().Panic("无法打开日志文件：", err)
	}
	defer logFile.Close()

	// 配置日志输出
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // 设置日志格式
}

// 初始化consul
func InitRpc() {
	// 创建 Consul 客户端
	consulCfg := api.DefaultConfig()
	consulCfg.Address = Nacos["cousulAddress"].(string) + ":" + strconv.Itoa(Nacos["cousulPort"].(int))
	consulClient, err := api.NewClient(consulCfg)
	if err != nil {
		zap.S().Panic("Failed to create Consul client: %v", err)
	}

	// 使用 Consul 客户端进行服务发现
	serviceName := Nacos["Server"].(map[string]interface{})["name"].(string)
	entries, _, err := consulClient.Health().Service(serviceName, "", true, nil)
	if err != nil {
		zap.S().Panic("Failed to discover service with Consul: %v", err)
	}

	// 构建 gRPC 连接
	//var conn *grpc.ClientConn
	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%d", entries[0].Service.Address, entries[0].Service.Port),
		//"consul://10.2.178.13:8500/user_srv?wait=14s",
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		//grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
	)
	if err != nil {
		zap.S().Panic("Failed to connect to gRPC server: %v", err)
	}

	// 创建 gRPC 客户端
	client := proto.NewInventoryClient(conn)
	GoodsClient = client
}
func InitTrans() {
	locale := Nacos["language"].(string)
	//修改gin框架中的validator引擎属性, 实现定制
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		//注册一个获取json的tag的自定义方法
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})

		zhT := zh.New() //中文翻译器
		enT := en.New() //英文翻译器
		//第一个参数是备用的语言环境，后面的参数是应该支持的语言环境
		uni := ut.New(enT, zhT, enT)
		Trans, ok = uni.GetTranslator(locale)
		if !ok {
			zap.S().Fatal("uni.GetTranslator(%s)", locale)
			return
		}

		switch locale {
		case "en":
			en_translations.RegisterDefaultTranslations(v, Trans)
		case "zh":
			zh_translations.RegisterDefaultTranslations(v, Trans)
		default:
			en_translations.RegisterDefaultTranslations(v, Trans)
		}
		return
	}
	return
}

// 初始化zap日志库
func InitZap() {
	var logger *zap.Logger
	if Env == "dev" {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}

	// flushes buffer
	defer logger.Sync()
	zap.ReplaceGlobals(logger)
}

func InitViper() {
	// 初始化Viper
	viper.SetConfigName("inventory_web") // 配置文件名（不带扩展名）
	viper.SetConfigType("yaml")          // 配置文件类型
	viper.AddConfigPath("./conf")        // 配置文件路径

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		zap.S().Panic("Error reading config file: %v", err)
	}

	//var NocosConfig map[string]interface{}

	err := viper.Unmarshal(&NacosConf)
	if err != nil {
		zap.S().Panic("Error unmarshal config file: %v", err)
	}

	fmt.Println("nocos config:", NacosConf)

	// 创建一个fsnotify监视器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		zap.S().Panic("Error creating watcher:", err)
	}
	defer watcher.Close()

	// 添加配置文件路径到监视器
	if err := watcher.Add("./conf"); err != nil {
		zap.S().Panic("Error adding path to watcher:", err)
	}

	// 启动一个goroutine来处理文件变化事件
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					zap.S().Infof("Config file modified. Reloading...")
					// 重新读取配置文件
					if err := viper.ReadInConfig(); err != nil {
						zap.S().Infof("Error reloading config:", err)
					}
					// 处理配置文件变化，例如更新应用程序的配置
					handleConfigChange()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				zap.S().Infof("Error watching for changes:", err)
			}
		}
	}()

	// 无限循环，保持程序运行
	//select {}
}

func handleConfigChange() {
	// 在这里处理配置文件变化
	err := viper.Unmarshal(&NacosConf)
	if err != nil {
		zap.S().Panic("Error unmarshal config file: %v", err)
	}
	InitNaCos()
}

func InitConsul() (*api.Client, string) {
	// 创建Consul客户端
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		fmt.Println("Failed to create Consul client:", err)
		return nil, ""
	}

	id := uuidToStr()
	userconf := Nacos["Client"].(map[string]interface{})
	//var tags []string
	//for _, v := range userconf {
	//	tags = append(tags, v.(string))
	//}
	//fmt.Println(tags)
	// 服务注册信息
	service := &api.AgentServiceRegistration{
		ID:      id,
		Address: userconf["address"].(string),
		Name:    userconf["name"].(string),
		Port:    userconf["port"].(int),
		Check: &api.AgentServiceCheck{
			HTTP:                           "http://" + Nacos["Client"].(map[string]interface{})["address"].(string) + ":" + strconv.Itoa(Nacos["Client"].(map[string]interface{})["port"].(int)) + "/health",
			Interval:                       "10s",
			Timeout:                        "1s",
			DeregisterCriticalServiceAfter: "30s",
		},
	}

	// 注册服务
	err = client.Agent().ServiceRegister(service)
	if err != nil {
		fmt.Println("Failed to register service with Consul:", err)
	}
	return client, id
}

func InitWebServer(router *gin.Engine) {
	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(Nacos["Client"].(map[string]interface{})["port"].(int)),
		Handler: router,
	}

	// 启动 HTTP 服务器
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	ConsulClient, ServerId = InitConsul()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	zap.S().Infof("Received signal %v, exiting...")
	<-sigChan
	ConsulClient.Agent().ServiceDeregister(ServerId)
	zap.S().Infof("Shutting down server...")

	// 创建一个超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭服务器，允许未完成的请求完成处理
	if err := server.Shutdown(ctx); err != nil {
		// 处理错误
	}
}

func uuidToStr() string { // 生成随机的 UUID
	randomUUID := uuid.New()

	// 将 UUID 转换为字符串形式
	serviceID := randomUUID.String()

	return serviceID
}

// 注销服务
func DeregisterService() {
	// 注销服务
	err := ConsulClient.Agent().ServiceDeregister(ServerId)
	if err != nil {
		fmt.Println("Failed to deregister service with Consul:", err)
	}

}
