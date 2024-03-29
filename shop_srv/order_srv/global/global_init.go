package global

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	_ "github.com/mbobakov/grpc-consul-resolver"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"order_srv/proto/goods_srv"
	"order_srv/proto/inventory_srv"
)

// 初始化服务依赖
func init() {
	InitEnv()
	InitViper()
	InitZap()
	InitNaCos()
	InitServer()
}

// nocos更新后，重新初始化服务依赖
func InitServer() {
	//InitNaCos()
	InitRpc()
	InitMysql()
	InitRedis()
	InitRocketMq()
	InitElastic()
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
	var sc = []constant.ServerConfig{{
		IpAddr: NacosConfig["nacos"].(map[string]interface{})["addr"].(string),
		Port:   uint64(NacosConfig["nacos"].(map[string]interface{})["port"].(int)),
	}}
	var cc = constant.ClientConfig{
		NamespaceId:         NacosConfig["nacos"].(map[string]interface{})["namespaceid"].(string),
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "tmp/logs",
		CacheDir:            "tmp",
		LogLevel:            "debug",
	}

	client, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": sc,
		"clientConfig":  cc,
	})
	if err != nil {
		zap.S().Panic(err)
	}
	content, err := client.GetConfig(vo.ConfigParam{
		DataId: NacosConfig["nacos"].(map[string]interface{})["dataid"].(string),
		Group:  Env,
	})

	if err != nil {
		zap.S().Panic(err)
	}

	err = yaml.Unmarshal([]byte(content), &Nacos)
	if err != nil {
		zap.S().Panic(err)
	}

	err = client.ListenConfig(vo.ConfigParam{
		DataId: NacosConfig["nacos"].(map[string]interface{})["dataid"].(string),
		Group:  Env,
		OnChange: func(namespace, group, dataId, data string) {
			err = yaml.Unmarshal([]byte(data), &Nacos)
			zap.S().Debugf("%v", data)
			zap.S().Debugf("%v", Nacos)
			zap.S().Debugf("%v", err)
			InitConsul()
			InitServer()
		},
	})

	if err != nil {
		zap.S().Panic(err)
	}

}

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
	MysqlConf.DB, err = gorm.Open(mysql.Open(MysqlConf.Dsn), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		zap.S().Panic(err)
	}

	sqlDB, err := MysqlConf.DB.DB()
	if err != nil {
		zap.S().Panic(err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

}
func InitRocketMq() {
	//把信息追加到延迟队列中
	//连接rocketmq

	RocketMqProducer, _ = rocketmq.NewProducer(
		producer.WithNsResolver(primitive.NewPassthroughResolver([]string{fmt.Sprintf("%s:%d", Nacos["rocketmq"].(map[string]interface{})["host"].(string), Nacos["rocketmq"].(map[string]interface{})["port"].(int))})),
		producer.WithRetry(2),
	)
	//开始生产
	err := RocketMqProducer.Start()
	if err != nil {
		zap.S().Panic("rocketmq生产者开启失败")
	}

}

// 初始化redis
func InitRedis() {
	RedisConf = &RedisConfig{
		Addr:     Nacos["redis"].(map[string]interface{})["addr"].(string),
		Port:     Nacos["redis"].(map[string]interface{})["port"].(int),
		DataBase: Nacos["redis"].(map[string]interface{})["database"].(string),
		Dsn:      Nacos["redis"].(map[string]interface{})["addr"].(string) + ":" + strconv.Itoa(Nacos["redis"].(map[string]interface{})["port"].(int)),
	}
	Rdb := redis.NewClient(&redis.Options{
		Addr: RedisConf.Dsn,
	})
	RedisConf.DB = Rdb
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

	server := Nacos["Server"].(map[string]interface{})
	for _, v := range server["name"].([]interface{}) {
		//serviceName := v.(map[string]interface{})["name"].(string)
		// 使用 Consul 客户端进行服务发现
		entries, _, err := consulClient.Health().Service(v.(string), "", true, nil)
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

		if v == "goods_srv" {
			// 创建 gRPC 客户端
			GoodsClient = goods.NewGoodsClient(conn)
		}
		if v == "inventory_srv" {
			InventoryClient = inventory.NewInventoryClient(conn)
		}
	}

}

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

// 注册服务到consul
func InitConsul() (*api.Client, string) {
	data, err := yaml.Marshal(Nacos["orderServer"].(map[string]interface{}))
	if err != nil {
		zap.S().Panic(err)
	}
	err = yaml.Unmarshal(data, &UserServerConf)

	if err != nil {
		zap.S().Panic(err)
	}
	// 注册服务到 Consul
	consulCfg := api.DefaultConfig()
	consulCfg.Address = Nacos["cousulAddress"].(string) + ":" + strconv.Itoa(Nacos["cousulPort"].(int))
	ConsulClient, err = api.NewClient(consulCfg)
	if err != nil {
		log.Fatalf("Failed to create Consul client: %v", err)
	}
	id := uuidToStr()

	registration := new(api.AgentServiceRegistration)
	registration.ID = id
	registration.Name = UserServerConf.Name
	registration.Address = UserServerConf.Address
	registration.Port = UserServerConf.Port
	registration.Tags = UserServerConf.Tags

	//添加健康检查
	registration.Check = &api.AgentServiceCheck{
		GRPC:     fmt.Sprintf("%s:%d", UserServerConf.Address, UserServerConf.Port), // 健康检查端点
		Interval: "10s",                                                             // 检查间隔
	}

	err = ConsulClient.Agent().ServiceRegister(registration)
	if err != nil {
		log.Fatalf("Failed to register service with Consul: %v", err)
	}
	return ConsulClient, id
}

func InitRPCServer(g *grpc.Server, c rocketmq.PushConsumer) {
	port := strconv.Itoa(Nacos["orderServer"].(map[string]interface{})["port"].(int))
	listen, err := net.Listen("tcp", ":"+port)
	if err != nil {
		panic(err)
	}

	// 创建 gRPC 健康检查服务
	grpc_health_v1.RegisterHealthServer(g, health.NewServer())

	go func() {
		if err := g.Serve(listen); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	zap.S().Infof("order_srv start success listen on " + port)

	consulClient, id := InitConsul()

	// 等待中断信号，然后注销服务
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Shutting down...")
	//释放rocketmq连接
	_ = RocketMqProducer.Shutdown()
	_ = c.Shutdown()
	err = consulClient.Agent().ServiceDeregister(id)
	if err != nil {
		zap.S().Fatal(err)
	}
	g.GracefulStop()
}

func uuidToStr() string { // 生成随机的 UUID
	randomUUID := uuid.New()

	// 将 UUID 转换为字符串形式
	serviceID := randomUUID.String()

	return serviceID
}

func InitViper() {
	// 初始化Viper
	viper.SetConfigName("order_srv") // 配置文件名（不带扩展名）
	viper.SetConfigType("yaml")      // 配置文件类型
	viper.AddConfigPath("./conf")    // 配置文件路径

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		zap.S().Panic("Error reading config file: %v", err)
	}

	fmt.Println("nocosmap config:", viper.GetStringMap("nacos"))
	//mysqlConfig := Config{}
	//var NacosConfig map[string]interface{}

	err := viper.Unmarshal(&NacosConfig)
	if err != nil {
		zap.S().Panic("Error unmarshal config file: %v", err)
	}

	fmt.Println("nocos config:", NacosConfig)

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
	fmt.Println("Config reloaded. New value of foo is:", viper.Get("nocos"))
	err := viper.Unmarshal(&NacosConfig)
	if err != nil {
		zap.S().Panic("Error unmarshal config file: %v", err)
	}
	InitNaCos()
}

func InitElastic() {
	//var err error
	//addr := Nacos["elasticsearch"].(map[string]interface{})["addr"].(string)
	//port := Nacos["elasticsearch"].(map[string]interface{})["port"].(int)
	//url := fmt.Sprintf("http://%s:%d", addr, port)
	//esClient, err = elastic.NewClient(elastic.SetURL(url), elastic.SetSniff(false))
	//if err != nil {
	//	panic(err)
	//}
}
