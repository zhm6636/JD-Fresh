package global

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type MysqlConfig struct {
	DB       *gorm.DB
	Addr     string
	Port     int
	User     string
	Password string
	Dsn      string
	DataBase string
}

type RedisConfig struct {
	DB       *redis.Client
	Addr     string
	Port     int
	Dsn      string
	DataBase string
}

type UserServerConfig struct {
	Name    string
	Address string
	Port    int
	Tags    []string
}
